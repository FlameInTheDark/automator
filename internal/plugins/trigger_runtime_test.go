package plugins

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/FlameInTheDark/emerald/pkg/pluginapi"
)

type fakeTriggerPlugin struct {
	runtime pluginapi.TriggerRuntime
}

func (p *fakeTriggerPlugin) Describe(context.Context) (pluginapi.PluginInfo, error) {
	return pluginapi.PluginInfo{}, nil
}

func (p *fakeTriggerPlugin) ValidateConfig(context.Context, string, json.RawMessage) error {
	return nil
}

func (p *fakeTriggerPlugin) ExecuteAction(context.Context, string, json.RawMessage, map[string]any) (any, error) {
	return map[string]any{}, nil
}

func (p *fakeTriggerPlugin) ToolDefinition(context.Context, string, pluginapi.ToolNodeMetadata, json.RawMessage) (*pluginapi.ToolDefinition, error) {
	return nil, nil
}

func (p *fakeTriggerPlugin) ExecuteTool(context.Context, string, json.RawMessage, json.RawMessage, map[string]any) (any, error) {
	return map[string]any{}, nil
}

func (p *fakeTriggerPlugin) OpenTriggerRuntime(context.Context) (pluginapi.TriggerRuntime, error) {
	return p.runtime, nil
}

type fakeTriggerRuntime struct {
	mu        sync.Mutex
	snapshots []pluginapi.TriggerSubscriptionSnapshot
	events    chan *pluginapi.TriggerEvent
	closeCh   chan struct{}
	closeOnce sync.Once
}

func newFakeTriggerRuntime() *fakeTriggerRuntime {
	return &fakeTriggerRuntime{
		events:  make(chan *pluginapi.TriggerEvent, 8),
		closeCh: make(chan struct{}),
	}
}

func (r *fakeTriggerRuntime) SendSnapshot(_ context.Context, snapshot pluginapi.TriggerSubscriptionSnapshot) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.snapshots = append(r.snapshots, snapshot)
	return nil
}

func (r *fakeTriggerRuntime) Recv(ctx context.Context) (*pluginapi.TriggerEvent, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-r.closeCh:
		return nil, io.EOF
	case event := <-r.events:
		return event, nil
	}
}

func (r *fakeTriggerRuntime) Close() error {
	r.closeOnce.Do(func() {
		close(r.closeCh)
	})
	return nil
}

func (r *fakeTriggerRuntime) emit(event *pluginapi.TriggerEvent) {
	r.events <- event
}

func TestTriggerRuntimeServiceIgnoresStaleSubscriptionEvents(t *testing.T) {
	t.Parallel()

	runtime := newFakeTriggerRuntime()
	manager := &Manager{
		bundles: map[string]*bundleRuntime{
			"acme": {
				manifest: Manifest{ID: "acme"},
				plugin:   &fakeTriggerPlugin{runtime: runtime},
			},
		},
	}

	handled := make(chan string, 4)
	service := NewTriggerRuntimeService(manager, func(_ context.Context, subscription pluginapi.TriggerSubscription, event *pluginapi.TriggerEvent) error {
		handled <- subscription.SubscriptionID + ":" + event.SubscriptionID
		return nil
	})
	defer service.Stop()

	oldSubscription := pluginapi.TriggerSubscription{
		SubscriptionID: "pipeline-1:old",
		NodeType:       BuildNodeType(pluginapi.NodeKindTrigger, "acme", "pulse"),
		NodeID:         "pulse",
		NodeInstanceID: "old",
		PipelineID:     "pipeline-1",
	}
	newSubscription := pluginapi.TriggerSubscription{
		SubscriptionID: "pipeline-1:new",
		NodeType:       BuildNodeType(pluginapi.NodeKindTrigger, "acme", "pulse"),
		NodeID:         "pulse",
		NodeInstanceID: "new",
		PipelineID:     "pipeline-1",
	}

	if err := service.Reload(context.Background(), []pluginapi.TriggerSubscription{oldSubscription}); err != nil {
		t.Fatalf("Reload(old) error = %v", err)
	}

	runtime.emit(&pluginapi.TriggerEvent{
		SubscriptionID: oldSubscription.SubscriptionID,
		Payload:        map[string]any{"message": "first"},
	})

	select {
	case got := <-handled:
		if got != "pipeline-1:old:pipeline-1:old" {
			t.Fatalf("first handled event = %q, want pipeline-1:old:pipeline-1:old", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for initial trigger event")
	}

	if err := service.Reload(context.Background(), []pluginapi.TriggerSubscription{newSubscription}); err != nil {
		t.Fatalf("Reload(new) error = %v", err)
	}

	runtime.emit(&pluginapi.TriggerEvent{
		SubscriptionID: oldSubscription.SubscriptionID,
		Payload:        map[string]any{"message": "stale"},
	})

	select {
	case got := <-handled:
		t.Fatalf("unexpected stale event delivery: %q", got)
	case <-time.After(200 * time.Millisecond):
	}

	runtime.emit(&pluginapi.TriggerEvent{
		SubscriptionID: newSubscription.SubscriptionID,
		Payload:        map[string]any{"message": "fresh"},
	})

	select {
	case got := <-handled:
		if got != "pipeline-1:new:pipeline-1:new" {
			t.Fatalf("fresh handled event = %q, want pipeline-1:new:pipeline-1:new", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for refreshed trigger event")
	}
}
