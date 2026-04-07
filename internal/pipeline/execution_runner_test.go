package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/FlameInTheDark/automator/internal/db/models"
	"github.com/FlameInTheDark/automator/internal/node"
)

type blockingExecutor struct {
	started chan struct{}
}

func (e *blockingExecutor) Execute(ctx context.Context, _ json.RawMessage, _ map[string]any) (*node.NodeResult, error) {
	select {
	case e.started <- struct{}{}:
	default:
	}

	<-ctx.Done()
	return nil, ctx.Err()
}

func (e *blockingExecutor) Validate(_ json.RawMessage) error {
	return nil
}

type testExecutionStore struct {
	mu             sync.Mutex
	executions     map[string]*models.Execution
	nodeExecutions map[string]*models.NodeExecution
}

func newTestExecutionStore() *testExecutionStore {
	return &testExecutionStore{
		executions:     make(map[string]*models.Execution),
		nodeExecutions: make(map[string]*models.NodeExecution),
	}
}

func (s *testExecutionStore) Create(_ context.Context, e *models.Execution) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	copyValue := *e
	copyValue.ID = "exec-1"
	e.ID = copyValue.ID
	if copyValue.StartedAt.IsZero() {
		copyValue.StartedAt = time.Now()
		e.StartedAt = copyValue.StartedAt
	}
	s.executions[e.ID] = &copyValue
	return nil
}

func (s *testExecutionStore) UpdateStatus(_ context.Context, id, status string, completedAt *time.Time, errMsg *string) error {
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

func (s *testExecutionStore) CreateNodeExecution(_ context.Context, ne *models.NodeExecution) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	copyValue := *ne
	copyValue.ID = "node-" + ne.NodeID
	ne.ID = copyValue.ID
	s.nodeExecutions[ne.ID] = &copyValue
	return nil
}

func (s *testExecutionStore) UpdateNodeExecution(_ context.Context, id string, status string, output json.RawMessage, errMsg *string, completedAt *time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	nodeExecution := s.nodeExecutions[id]
	if nodeExecution == nil {
		return errors.New("node execution not found")
	}

	nodeExecution.Status = status
	if len(output) > 0 {
		outputValue := string(output)
		nodeExecution.Output = &outputValue
	}
	nodeExecution.Error = errMsg
	nodeExecution.CompletedAt = completedAt
	return nil
}

type broadcastRecorder struct {
	mu       sync.Mutex
	messages []map[string]any
}

func (r *broadcastRecorder) Broadcast(_ string, data any) {
	payload, ok := data.(map[string]any)
	if !ok {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.messages = append(r.messages, payload)
}

func (r *broadcastRecorder) hasMessageType(messageType string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, message := range r.messages {
		if message["type"] == messageType {
			return true
		}
	}

	return false
}

func TestExecutionRunner_CancelTracksActiveExecution(t *testing.T) {
	registry := node.NewRegistry()
	executor := &blockingExecutor{started: make(chan struct{}, 1)}
	registry.Register("test:blocking", executor)

	engine := NewEngine(registry)
	store := newTestExecutionStore()
	broadcaster := &broadcastRecorder{}
	runner := NewExecutionRunner(store, engine, broadcaster)

	flowData := FlowData{
		Nodes: []FlowNode{
			{ID: "blocking", Type: "test:blocking"},
		},
	}

	resultCh := make(chan *ExecutionRunResult, 1)
	errCh := make(chan error, 1)

	go func() {
		result, err := runner.Run(context.Background(), "pipeline-1", flowData, "manual", nil)
		resultCh <- result
		errCh <- err
	}()

	select {
	case <-executor.started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for node start")
	}

	deadline := time.Now().Add(2 * time.Second)
	var active []ActiveExecutionInfo
	for time.Now().Before(deadline) {
		active = runner.ActiveByPipeline("pipeline-1")
		if len(active) == 1 && active[0].CurrentNodeID == "blocking" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if len(active) != 1 {
		t.Fatalf("expected 1 active execution, got %d", len(active))
	}
	if active[0].CurrentNodeID != "blocking" {
		t.Fatalf("expected current node to be blocking, got %q", active[0].CurrentNodeID)
	}

	info, ok := runner.Cancel(active[0].ExecutionID)
	if !ok {
		t.Fatal("expected cancellation to succeed")
	}
	if info.Status != "cancelling" {
		t.Fatalf("expected status cancelling, got %q", info.Status)
	}

	var result *ExecutionRunResult
	select {
	case result = <-resultCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for run result")
	}

	if err := <-errCh; err != nil {
		t.Fatalf("runner returned unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected run result")
	}
	if result.Status != "cancelled" {
		t.Fatalf("expected cancelled status, got %q", result.Status)
	}
	if !errors.Is(result.Error, context.Canceled) {
		t.Fatalf("expected context canceled error, got %v", result.Error)
	}

	if active := runner.ActiveByPipeline("pipeline-1"); len(active) != 0 {
		t.Fatalf("expected no active executions after cancellation, got %d", len(active))
	}

	execution := store.executions[result.ExecutionID]
	if execution == nil {
		t.Fatal("expected execution to be stored")
	}
	if execution.Status != "cancelled" {
		t.Fatalf("expected stored execution status cancelled, got %q", execution.Status)
	}

	nodeExecution := store.nodeExecutions["node-blocking"]
	if nodeExecution == nil {
		t.Fatal("expected node execution to be stored")
	}
	if nodeExecution.Status != "cancelled" {
		t.Fatalf("expected node execution status cancelled, got %q", nodeExecution.Status)
	}

	if !broadcaster.hasMessageType("execution_started") {
		t.Fatal("expected execution_started broadcast")
	}
	if !broadcaster.hasMessageType("execution_cancelling") {
		t.Fatal("expected execution_cancelling broadcast")
	}
	if !broadcaster.hasMessageType("execution_completed") {
		t.Fatal("expected execution_completed broadcast")
	}
}
