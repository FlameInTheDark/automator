package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/FlameInTheDark/emerald/pkg/pluginapi"
	"github.com/FlameInTheDark/emerald/pkg/pluginsdk"
)

type heartbeatConfig struct {
	IntervalSeconds int    `json:"intervalSeconds"`
	Message         string `json:"message"`
	EventType       string `json:"eventType"`
}

type heartbeatTrigger struct{}

func (t *heartbeatTrigger) ValidateConfig(_ context.Context, config json.RawMessage) error {
	_, err := decodeHeartbeatConfig(config)
	return err
}

type runtimeProvider struct{}

func (p *runtimeProvider) OpenTriggerRuntime(ctx context.Context) (pluginapi.TriggerRuntime, error) {
	return newHeartbeatRuntime(ctx), nil
}

type heartbeatRuntime struct {
	ctx    context.Context
	cancel context.CancelFunc
	events chan *pluginapi.TriggerEvent

	mu            sync.Mutex
	subscriptions map[string]context.CancelFunc
}

func newHeartbeatRuntime(parent context.Context) *heartbeatRuntime {
	ctx, cancel := context.WithCancel(parent)
	return &heartbeatRuntime{
		ctx:           ctx,
		cancel:        cancel,
		events:        make(chan *pluginapi.TriggerEvent, 32),
		subscriptions: make(map[string]context.CancelFunc),
	}
}

func (r *heartbeatRuntime) SendSnapshot(_ context.Context, snapshot pluginapi.TriggerSubscriptionSnapshot) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for subscriptionID, cancel := range r.subscriptions {
		cancel()
		delete(r.subscriptions, subscriptionID)
	}

	for _, subscription := range snapshot.Subscriptions {
		cfg, err := decodeHeartbeatConfig(subscription.Config)
		if err != nil {
			return fmt.Errorf("subscription %s: %w", subscription.SubscriptionID, err)
		}

		subscriptionCtx, cancel := context.WithCancel(r.ctx)
		r.subscriptions[subscription.SubscriptionID] = cancel

		go r.runSubscription(subscriptionCtx, subscription, cfg)
	}

	return nil
}

func (r *heartbeatRuntime) Recv(ctx context.Context) (*pluginapi.TriggerEvent, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-r.ctx.Done():
		return nil, r.ctx.Err()
	case event := <-r.events:
		return event, nil
	}
}

func (r *heartbeatRuntime) Close() error {
	r.cancel()

	r.mu.Lock()
	defer r.mu.Unlock()

	for subscriptionID, cancel := range r.subscriptions {
		cancel()
		delete(r.subscriptions, subscriptionID)
	}

	return nil
}

func (r *heartbeatRuntime) runSubscription(ctx context.Context, subscription pluginapi.TriggerSubscription, cfg heartbeatConfig) {
	ticker := time.NewTicker(time.Duration(cfg.IntervalSeconds) * time.Second)
	defer ticker.Stop()

	sequence := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			sequence++

			payload := map[string]any{
				"event_type":       cfg.EventType,
				"message":          cfg.Message,
				"interval_seconds": cfg.IntervalSeconds,
				"sequence":         sequence,
				"fired_at":         time.Now().UTC().Format(time.RFC3339Nano),
				"pipeline_id":      subscription.PipelineID,
				"node_instance_id": subscription.NodeInstanceID,
				"node_type":        subscription.NodeType,
			}

			select {
			case <-ctx.Done():
				return
			case <-r.ctx.Done():
				return
			case r.events <- &pluginapi.TriggerEvent{
				SubscriptionID: subscription.SubscriptionID,
				Payload:        payload,
			}:
			}
		}
	}
}

func decodeHeartbeatConfig(raw json.RawMessage) (heartbeatConfig, error) {
	cfg := heartbeatConfig{
		IntervalSeconds: 10,
		Message:         "Hello from Sample Trigger Kit",
		EventType:       "heartbeat",
	}
	if len(raw) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return heartbeatConfig{}, fmt.Errorf("decode heartbeat config: %w", err)
	}

	cfg.Message = strings.TrimSpace(cfg.Message)
	cfg.EventType = strings.TrimSpace(cfg.EventType)
	if cfg.IntervalSeconds <= 0 {
		return heartbeatConfig{}, fmt.Errorf("intervalSeconds must be greater than 0")
	}
	if cfg.Message == "" {
		cfg.Message = "Hello from Sample Trigger Kit"
	}
	if cfg.EventType == "" {
		cfg.EventType = "heartbeat"
	}

	return cfg, nil
}

func main() {
	bundle := &pluginapi.Bundle{
		Info: pluginapi.PluginInfo{
			ID:         "sample-trigger-kit",
			Name:       "Sample Trigger Kit",
			Version:    "0.1.0",
			APIVersion: pluginapi.APIVersion,
			Nodes: []pluginapi.NodeSpec{
				{
					ID:          "heartbeat",
					Kind:        pluginapi.NodeKindTrigger,
					Label:       "Heartbeat Trigger",
					Description: "Emit periodic trigger events to demonstrate the subscription snapshot runtime.",
					Icon:        "radio",
					Color:       "#f59e0b",
					MenuPath:    []string{"Sample Trigger Kit"},
					DefaultConfig: map[string]any{
						"intervalSeconds": 10,
						"message":         "Hello from Sample Trigger Kit",
						"eventType":       "heartbeat",
					},
					Fields: []pluginapi.FieldSpec{
						{
							Name:               "intervalSeconds",
							Label:              "Interval Seconds",
							Type:               pluginapi.FieldTypeNumber,
							Required:           true,
							DefaultNumberValue: float64Ptr(10),
							Description:        "How often the plugin emits an event for this subscription.",
						},
						{
							Name:               "message",
							Label:              "Message",
							Type:               pluginapi.FieldTypeString,
							DefaultStringValue: "Hello from Sample Trigger Kit",
							Description:        "Message included in the emitted payload.",
						},
						{
							Name:               "eventType",
							Label:              "Event Type",
							Type:               pluginapi.FieldTypeSelect,
							DefaultStringValue: "heartbeat",
							Options: []pluginapi.FieldOption{
								{Value: "heartbeat", Label: "Heartbeat"},
								{Value: "reminder", Label: "Reminder"},
								{Value: "demo", Label: "Demo"},
							},
							Description: "Simple event label included in the payload for downstream routing.",
						},
					},
					OutputHints: []pluginapi.OutputHint{
						{Expression: "input.event_type", Label: "Emitted event type"},
						{Expression: "input.message", Label: "Configured message"},
						{Expression: "input.sequence", Label: "Per-subscription event counter"},
						{Expression: "input.fired_at", Label: "UTC timestamp when the event fired"},
					},
				},
			},
		},
		Triggers: map[string]pluginapi.TriggerNode{
			"heartbeat": &heartbeatTrigger{},
		},
		TriggerRuntimeProvider: &runtimeProvider{},
	}

	pluginsdk.Serve(bundle)
}

func float64Ptr(value float64) *float64 {
	return &value
}
