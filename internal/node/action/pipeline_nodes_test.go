package action

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/FlameInTheDark/automator/internal/db/models"
	"github.com/FlameInTheDark/automator/internal/node"
	"github.com/FlameInTheDark/automator/internal/pipeline"
)

type stubPipelineRunner struct {
	pipelineID string
	input      map[string]any
}

func (s *stubPipelineRunner) Run(_ context.Context, pipelineID string, input map[string]any) (*pipeline.RunResult, error) {
	s.pipelineID = pipelineID
	s.input = input

	return &pipeline.RunResult{
		ExecutionID: "exec-1",
		PipelineID:  pipelineID,
		Status:      "completed",
		Returned:    true,
		ReturnValue: input,
	}, nil
}

type stubPipelineCatalog struct {
	byID map[string]*models.Pipeline
}

func (s *stubPipelineCatalog) List(context.Context) ([]models.Pipeline, error) {
	return nil, nil
}

func (s *stubPipelineCatalog) GetByID(_ context.Context, id string) (*models.Pipeline, error) {
	if pipelineModel, ok := s.byID[id]; ok {
		return pipelineModel, nil
	}

	return nil, nil
}

func TestPipelineRunToolDefinitionIncludesDynamicPipelineID(t *testing.T) {
	t.Parallel()

	executor := &PipelineRunToolNode{}
	config, err := json.Marshal(runPipelineConfig{
		ToolName:             "Run Support Pipeline",
		ToolDescription:      "Run any support pipeline by id.",
		AllowModelPipelineID: true,
	})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}

	definition, err := executor.ToolDefinition(context.Background(), node.ToolNodeMetadata{Label: "Pipeline Runner"}, config)
	if err != nil {
		t.Fatalf("tool definition: %v", err)
	}

	if got, want := definition.Function.Name, "run_support_pipeline"; got != want {
		t.Fatalf("tool name = %q, want %q", got, want)
	}
	if got, want := definition.Function.Description, "Run any support pipeline by id."; got != want {
		t.Fatalf("tool description = %q, want %q", got, want)
	}

	properties, ok := definition.Function.Parameters["properties"].(map[string]any)
	if !ok {
		t.Fatalf("tool properties missing or wrong type: %#v", definition.Function.Parameters["properties"])
	}
	if _, ok := properties["pipelineId"]; !ok {
		t.Fatalf("pipelineId property missing from tool definition")
	}

	required, ok := definition.Function.Parameters["required"].([]string)
	if ok {
		if len(required) != 1 || required[0] != "pipelineId" {
			t.Fatalf("required fields = %#v, want [pipelineId]", required)
		}
		return
	}

	requiredAny, ok := definition.Function.Parameters["required"].([]any)
	if !ok {
		t.Fatalf("required fields missing or wrong type: %#v", definition.Function.Parameters["required"])
	}
	if len(requiredAny) != 1 || requiredAny[0] != "pipelineId" {
		t.Fatalf("required fields = %#v, want [pipelineId]", requiredAny)
	}
}

func TestPipelineRunToolExecuteUsesProvidedPipelineID(t *testing.T) {
	t.Parallel()

	runner := &stubPipelineRunner{}
	executor := &PipelineRunToolNode{
		Runner: runner,
	}
	config, err := json.Marshal(runPipelineConfig{
		AllowModelPipelineID: true,
	})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}

	result, err := executor.ExecuteTool(context.Background(), config, json.RawMessage(`{"pipelineId":"pipe-123","params":{"status":"ok"}}`), nil)
	if err != nil {
		t.Fatalf("execute tool: %v", err)
	}

	if got, want := runner.pipelineID, "pipe-123"; got != want {
		t.Fatalf("runner pipeline id = %q, want %q", got, want)
	}
	if got, want := runner.input, map[string]any{"status": "ok"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("runner input = %#v, want %#v", got, want)
	}

	output, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("tool result has unexpected type %T", result)
	}
	if got, want := output["pipeline_id"], "pipe-123"; got != want {
		t.Fatalf("tool output pipeline_id = %#v, want %#v", got, want)
	}
}
