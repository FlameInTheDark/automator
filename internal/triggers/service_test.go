package triggers

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/FlameInTheDark/emerald/internal/db/models"
	"github.com/FlameInTheDark/emerald/internal/node"
	logicnode "github.com/FlameInTheDark/emerald/internal/node/logic"
	triggernode "github.com/FlameInTheDark/emerald/internal/node/trigger"
	"github.com/FlameInTheDark/emerald/internal/pipeline"
	"github.com/FlameInTheDark/emerald/internal/plugins"
	"github.com/FlameInTheDark/emerald/pkg/pluginapi"
)

type recordingExecutionStore struct {
	mu             sync.Mutex
	executionCount int
	nodeCount      int
	executions     map[string]*models.Execution
	nodeExecutions map[string]*models.NodeExecution
}

func newRecordingExecutionStore() *recordingExecutionStore {
	return &recordingExecutionStore{
		executions:     make(map[string]*models.Execution),
		nodeExecutions: make(map[string]*models.NodeExecution),
	}
}

func (s *recordingExecutionStore) Create(_ context.Context, execution *models.Execution) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.executionCount++
	copyValue := *execution
	copyValue.ID = "exec-" + strconv.Itoa(s.executionCount)
	execution.ID = copyValue.ID
	if copyValue.StartedAt.IsZero() {
		copyValue.StartedAt = time.Now()
		execution.StartedAt = copyValue.StartedAt
	}

	s.executions[copyValue.ID] = &copyValue
	return nil
}

func (s *recordingExecutionStore) UpdateStatus(_ context.Context, id, status string, completedAt *time.Time, errMsg *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	execution := s.executions[id]
	if execution == nil {
		return errors.New("execution not found")
	}

	execution.Status = status
	execution.CompletedAt = completedAt
	execution.Error = errMsg
	return nil
}

func (s *recordingExecutionStore) CreateNodeExecution(_ context.Context, nodeExecution *models.NodeExecution) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nodeCount++
	copyValue := *nodeExecution
	copyValue.ID = "node-" + strconv.Itoa(s.nodeCount)
	nodeExecution.ID = copyValue.ID
	s.nodeExecutions[copyValue.NodeID] = &copyValue
	return nil
}

func (s *recordingExecutionStore) UpdateNodeExecution(_ context.Context, id string, status string, output json.RawMessage, errMsg *string, completedAt *time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for key, nodeExecution := range s.nodeExecutions {
		if nodeExecution.ID != id {
			continue
		}

		nodeExecution.Status = status
		if len(output) > 0 {
			outputValue := string(output)
			nodeExecution.Output = &outputValue
		}
		nodeExecution.Error = errMsg
		nodeExecution.CompletedAt = completedAt
		s.nodeExecutions[key] = nodeExecution
		return nil
	}

	return errors.New("node execution not found")
}

func (s *recordingExecutionStore) execution(id string) *models.Execution {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.executions[id]
}

func (s *recordingExecutionStore) nodeExecution(nodeID string) *models.NodeExecution {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.nodeExecutions[nodeID]
}

func TestBuildActiveSnapshotRejectsDuplicateWebhookEndpoints(t *testing.T) {
	t.Parallel()

	activePipelines := []models.Pipeline{
		{
			ID:     "pipeline-a",
			Status: "active",
			Nodes:  `[{"id":"webhook-a","data":{"type":"trigger:webhook","config":{"path":"orders","method":"POST"}}}]`,
			Edges:  `[]`,
		},
		{
			ID:     "pipeline-b",
			Status: "active",
			Nodes:  `[{"id":"webhook-b","data":{"type":"trigger:webhook","config":{"path":"/webhook/orders","method":"post"}}}]`,
			Edges:  `[]`,
		},
	}

	_, err := buildActiveSnapshot(activePipelines)
	if err == nil {
		t.Fatal("expected duplicate webhook endpoint validation to fail")
	}
	if !strings.Contains(err.Error(), "POST /webhook/orders") {
		t.Fatalf("expected normalized webhook key in error, got %v", err)
	}
}

func TestBuildActiveSnapshotIncludesPluginTriggerSubscriptions(t *testing.T) {
	t.Parallel()

	activePipelines := []models.Pipeline{
		{
			ID:     "pipeline-plugin",
			Status: "active",
			Nodes:  `[{"id":"pulse-node","data":{"type":"trigger:plugin/acme/pulse","config":{"intervalSeconds":15,"message":"hello"}}}]`,
			Edges:  `[]`,
		},
	}

	snapshot, err := buildActiveSnapshot(activePipelines)
	if err != nil {
		t.Fatalf("buildActiveSnapshot() error = %v", err)
	}

	if len(snapshot.subscriptions) != 1 {
		t.Fatalf("subscription count = %d, want 1", len(snapshot.subscriptions))
	}

	subscription := snapshot.subscriptions[0]
	if subscription.SubscriptionID != "pipeline-plugin:pulse-node" {
		t.Fatalf("subscription id = %q, want pipeline-plugin:pulse-node", subscription.SubscriptionID)
	}
	if subscription.NodeType != "trigger:plugin/acme/pulse" {
		t.Fatalf("node type = %q, want trigger:plugin/acme/pulse", subscription.NodeType)
	}
	if subscription.NodeID != "pulse" {
		t.Fatalf("node id = %q, want pulse", subscription.NodeID)
	}
	if subscription.NodeInstanceID != "pulse-node" {
		t.Fatalf("node instance id = %q, want pulse-node", subscription.NodeInstanceID)
	}
	if _, ok := snapshot.pluginDispatch[subscription.SubscriptionID]; !ok {
		t.Fatalf("expected plugin dispatch entry for %q", subscription.SubscriptionID)
	}
}

func TestServiceDispatchWebhookParsesRequestAndReturnsPipelineOutput(t *testing.T) {
	t.Parallel()

	registry := node.NewRegistry()
	registry.Register("trigger:webhook", &triggernode.WebhookTrigger{})
	registry.Register("logic:return", &logicnode.ReturnNode{})

	store := newRecordingExecutionStore()
	runner := pipeline.NewExecutionRunner(store, pipeline.NewEngine(registry), nil)

	webhookConfig, err := json.Marshal(triggernode.WebhookConfig{
		Path:   "/webhook/sample",
		Method: "POST",
		Token:  "shared-secret",
	})
	if err != nil {
		t.Fatalf("marshal webhook config: %v", err)
	}

	flowData := pipeline.FlowData{
		Nodes: []pipeline.FlowNode{
			{ID: "webhook", Type: "trigger:webhook", Data: webhookConfig},
			{ID: "return", Type: "logic:return", Data: json.RawMessage(`{}`)},
		},
		Edges: []pipeline.FlowEdge{
			{ID: "edge-1", Source: "webhook", Target: "return"},
		},
	}

	service := &Service{
		runner: runner,
		webhooks: map[string]webhookEntry{
			webhookKey("POST", "/webhook/sample"): {
				Method:     "POST",
				Path:       "/webhook/sample",
				Token:      "shared-secret",
				PipelineID: "pipeline-webhook",
				NodeID:     "webhook",
				FlowData:   flowData,
			},
		},
		pluginDispatch: make(map[string]pluginDispatchEntry),
	}

	result, err := service.DispatchWebhook(context.Background(), WebhookRequest{
		Method:      "post",
		Path:        "sample",
		Token:       "shared-secret",
		ContentType: "application/json; charset=utf-8",
		Headers: map[string][]string{
			"X-Test": {"alpha"},
		},
		Query: map[string][]string{
			"attempt": {"1"},
		},
		Body:      []byte(`{"message":"hello","count":2}`),
		RemoteIP:  "127.0.0.1",
		UserAgent: "trigger-test",
	})
	if err != nil {
		t.Fatalf("DispatchWebhook() error = %v", err)
	}

	if result.Status != "completed" {
		t.Fatalf("result status = %q, want completed", result.Status)
	}
	if !result.Returned {
		t.Fatal("expected webhook run to return a pipeline value")
	}

	returnValue, ok := result.ReturnValue.(map[string]any)
	if !ok {
		t.Fatalf("return value type = %T, want map[string]any", result.ReturnValue)
	}

	if got := returnValue["triggered_by"]; got != "webhook" {
		t.Fatalf("triggered_by = %#v, want webhook", got)
	}
	if got := returnValue["path"]; got != "/webhook/sample" {
		t.Fatalf("path = %#v, want /webhook/sample", got)
	}
	if got := returnValue["method"]; got != "POST" {
		t.Fatalf("method = %#v, want POST", got)
	}
	if got := returnValue["raw_body"]; got != `{"message":"hello","count":2}` {
		t.Fatalf("raw_body = %#v, want JSON string", got)
	}

	body, ok := returnValue["body"].(map[string]any)
	if !ok {
		t.Fatalf("body type = %T, want map[string]any", returnValue["body"])
	}
	if got := body["message"]; got != "hello" {
		t.Fatalf("body.message = %#v, want hello", got)
	}
	if got := body["count"]; got != float64(2) {
		t.Fatalf("body.count = %#v, want 2", got)
	}

	headers, ok := returnValue["headers"].(map[string]any)
	if !ok || headers["X-Test"] != "alpha" {
		t.Fatalf("headers = %#v, want X-Test=alpha", returnValue["headers"])
	}
	query, ok := returnValue["query"].(map[string]any)
	if !ok || query["attempt"] != "1" {
		t.Fatalf("query = %#v, want attempt=1", returnValue["query"])
	}
	if got := returnValue["content_type"]; got != "application/json; charset=utf-8" {
		t.Fatalf("content_type = %#v, want application/json; charset=utf-8", got)
	}
	if got := returnValue["remote_ip"]; got != "127.0.0.1" {
		t.Fatalf("remote_ip = %#v, want 127.0.0.1", got)
	}
	if got := returnValue["user_agent"]; got != "trigger-test" {
		t.Fatalf("user_agent = %#v, want trigger-test", got)
	}

	execution := store.execution(result.ExecutionID)
	if execution == nil || execution.Context == nil {
		t.Fatal("expected execution context to be persisted")
	}
	if !strings.Contains(*execution.Context, `"root_node_ids":["webhook"]`) {
		t.Fatalf("expected persisted trigger selection to include exact root node ids, got %s", *execution.Context)
	}
}

func TestServiceDispatchWebhookRejectsInvalidToken(t *testing.T) {
	t.Parallel()

	service := &Service{
		runner: pipeline.NewExecutionRunner(newRecordingExecutionStore(), pipeline.NewEngine(node.NewRegistry()), nil),
		webhooks: map[string]webhookEntry{
			webhookKey("POST", "/webhook/protected"): {
				Method: "POST",
				Path:   "/webhook/protected",
				Token:  "correct-token",
			},
		},
		pluginDispatch: make(map[string]pluginDispatchEntry),
	}

	_, err := service.DispatchWebhook(context.Background(), WebhookRequest{
		Method: "POST",
		Path:   "/webhook/protected",
		Token:  "wrong-token",
	})
	if !errors.Is(err, ErrWebhookUnauthorized) {
		t.Fatalf("DispatchWebhook() error = %v, want ErrWebhookUnauthorized", err)
	}
}

func TestServiceHandlePluginEventRunsExactTriggerNode(t *testing.T) {
	t.Parallel()

	registry := node.NewRegistry()
	registry.Register("trigger:plugin/acme/pulse", &plugins.TriggerExecutor{
		Manager:  &plugins.Manager{},
		NodeType: "trigger:plugin/acme/pulse",
	})
	registry.Register("logic:return", &logicnode.ReturnNode{})

	store := newRecordingExecutionStore()
	runner := pipeline.NewExecutionRunner(store, pipeline.NewEngine(registry), nil)

	flowData := pipeline.FlowData{
		Nodes: []pipeline.FlowNode{
			{ID: "pulse-instance", Type: "trigger:plugin/acme/pulse"},
			{ID: "return", Type: "logic:return", Data: json.RawMessage(`{}`)},
		},
		Edges: []pipeline.FlowEdge{
			{ID: "edge-1", Source: "pulse-instance", Target: "return"},
		},
	}

	service := &Service{
		runner:   runner,
		webhooks: map[string]webhookEntry{},
		pluginDispatch: map[string]pluginDispatchEntry{
			"pipeline-1:pulse-instance": {
				PipelineID: "pipeline-1",
				NodeID:     "pulse-instance",
				FlowData:   flowData,
			},
		},
	}

	err := service.HandlePluginEvent(context.Background(), pluginapi.TriggerSubscription{
		SubscriptionID: "pipeline-1:pulse-instance",
	}, &pluginapi.TriggerEvent{
		SubscriptionID: "pipeline-1:pulse-instance",
		Payload: map[string]any{
			"message": "ping",
			"count":   3,
		},
	})
	if err != nil {
		t.Fatalf("HandlePluginEvent() error = %v", err)
	}

	triggerExecution := store.nodeExecution("pulse-instance")
	if triggerExecution == nil || triggerExecution.Output == nil {
		t.Fatal("expected trigger node execution output to be stored")
	}

	var triggerOutput map[string]any
	if err := json.Unmarshal([]byte(*triggerExecution.Output), &triggerOutput); err != nil {
		t.Fatalf("unmarshal trigger output: %v", err)
	}

	if got := triggerOutput["triggered_by"]; got != "plugin" {
		t.Fatalf("triggered_by = %#v, want plugin", got)
	}
	if got := triggerOutput["subscription_id"]; got != "pipeline-1:pulse-instance" {
		t.Fatalf("subscription_id = %#v, want pipeline-1:pulse-instance", got)
	}
	if got := triggerOutput["message"]; got != "ping" {
		t.Fatalf("message = %#v, want ping", got)
	}
	if got := triggerOutput["count"]; got != float64(3) {
		t.Fatalf("count = %#v, want 3", got)
	}
}
