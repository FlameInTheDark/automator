package action

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/FlameInTheDark/automator/internal/db/models"
	"github.com/FlameInTheDark/automator/internal/llm"
	"github.com/FlameInTheDark/automator/internal/node"
	"github.com/FlameInTheDark/automator/internal/pipeline"
	"github.com/FlameInTheDark/automator/internal/pipelineops"
	"github.com/FlameInTheDark/automator/internal/templating"
)

type PipelineRunner interface {
	Run(ctx context.Context, pipelineID string, input map[string]any) (*pipeline.RunResult, error)
}

type PipelineCatalog interface {
	List(ctx context.Context) ([]models.Pipeline, error)
	GetByID(ctx context.Context, id string) (*models.Pipeline, error)
}

type runPipelineConfig struct {
	PipelineID           string `json:"pipelineId"`
	Params               string `json:"params"`
	ToolName             string `json:"toolName"`
	ToolDescription      string `json:"toolDescription"`
	AllowModelPipelineID bool   `json:"allowModelPipelineId"`
}

type RunPipelineAction struct {
	Runner PipelineRunner
}

func (e *RunPipelineAction) Execute(ctx context.Context, config json.RawMessage, input map[string]any) (*node.NodeResult, error) {
	if e.Runner == nil {
		return nil, fmt.Errorf("pipeline runner is not configured")
	}

	cfg, err := parseRunPipelineConfig(config, input)
	if err != nil {
		return nil, err
	}

	params, err := parsePipelineParams(cfg.Params, input)
	if err != nil {
		return nil, err
	}

	result, err := e.Runner.Run(ctx, cfg.PipelineID, params)
	if err != nil {
		return nil, fmt.Errorf("run pipeline %s: %w", cfg.PipelineID, err)
	}

	data, _ := json.Marshal(buildRunPipelineOutput(result))
	return &node.NodeResult{Output: data}, nil
}

func (e *RunPipelineAction) Validate(config json.RawMessage) error {
	var cfg runPipelineConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	if strings.TrimSpace(cfg.PipelineID) == "" {
		return fmt.Errorf("pipelineId is required")
	}
	return nil
}

type PipelineListToolNode struct {
	Pipelines PipelineCatalog
}

type pipelineListToolArgs struct {
	PipelineID        string `json:"pipelineId"`
	PipelineName      string `json:"pipelineName"`
	IncludeDefinition bool   `json:"includeDefinition"`
}

func (e *PipelineListToolNode) Execute(ctx context.Context, _ json.RawMessage, _ map[string]any) (*node.NodeResult, error) {
	output, err := e.listPipelinesOutput(ctx, pipelineListToolArgs{})
	if err != nil {
		return nil, err
	}

	data, _ := json.Marshal(output)
	return &node.NodeResult{Output: data}, nil
}

func (e *PipelineListToolNode) Validate(config json.RawMessage) error {
	if len(config) == 0 {
		return nil
	}

	var raw map[string]any
	if err := json.Unmarshal(config, &raw); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	return nil
}

func (e *PipelineListToolNode) ToolDefinition(_ context.Context, meta node.ToolNodeMetadata, _ json.RawMessage) (*llm.ToolDefinition, error) {
	return &llm.ToolDefinition{
		Type: "function",
		Function: llm.ToolSpec{
			Name:        sanitizeToolName(meta.Label, "list_pipelines"),
			Description: "List available pipelines. Optionally filter by pipelineId or pipelineName, and set includeDefinition to true when you need nodes, edges, and viewport data for editing.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"pipelineId": map[string]any{
						"type":        "string",
						"description": "Optional pipeline ID to filter by.",
					},
					"pipelineName": map[string]any{
						"type":        "string",
						"description": "Optional pipeline name to filter by.",
					},
					"includeDefinition": map[string]any{
						"type":        "boolean",
						"description": "When true, include nodes, edges, and viewport in the response.",
					},
				},
			},
		},
	}, nil
}

func (e *PipelineListToolNode) ExecuteTool(ctx context.Context, _ json.RawMessage, args json.RawMessage, _ map[string]any) (any, error) {
	toolArgs, err := parsePipelineListToolArgs(args)
	if err != nil {
		return nil, err
	}

	return e.listPipelinesOutput(ctx, toolArgs)
}

func (e *PipelineListToolNode) listPipelinesOutput(ctx context.Context, args pipelineListToolArgs) (map[string]any, error) {
	if e.Pipelines == nil {
		return nil, fmt.Errorf("pipeline store is not configured")
	}

	pipelines, err := e.Pipelines.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list pipelines: %w", err)
	}

	filtered := pipelineops.FilterPipelines(pipelines, pipelineops.Reference{
		ID:   args.PipelineID,
		Name: args.PipelineName,
	})

	return pipelineops.BuildListOutput(filtered, args.IncludeDefinition)
}

type PipelineRunToolNode struct {
	Pipelines PipelineCatalog
	Runner    PipelineRunner
}

func (e *PipelineRunToolNode) Execute(ctx context.Context, config json.RawMessage, input map[string]any) (*node.NodeResult, error) {
	if e.Runner == nil {
		return nil, fmt.Errorf("pipeline runner is not configured")
	}

	cfg, err := parseRunPipelineToolConfig(config)
	if err != nil {
		return nil, err
	}

	result, err := e.Runner.Run(ctx, cfg.PipelineID, copyParamsMap(input))
	if err != nil {
		return nil, fmt.Errorf("run pipeline %s: %w", cfg.PipelineID, err)
	}

	data, _ := json.Marshal(buildRunPipelineOutput(result))
	return &node.NodeResult{Output: data}, nil
}

func (e *PipelineRunToolNode) Validate(config json.RawMessage) error {
	_, err := parseRunPipelineToolConfig(config)
	return err
}

func (e *PipelineRunToolNode) ToolDefinition(ctx context.Context, meta node.ToolNodeMetadata, config json.RawMessage) (*llm.ToolDefinition, error) {
	cfg, err := parseRunPipelineToolConfig(config)
	if err != nil {
		return nil, err
	}

	description := strings.TrimSpace(cfg.ToolDescription)
	if description == "" {
		description = "Run the configured pipeline manually and optionally pass a params object as input."
	}
	if e.Pipelines != nil && strings.TrimSpace(cfg.PipelineID) != "" {
		if pipelineModel, err := e.Pipelines.GetByID(ctx, cfg.PipelineID); err == nil && pipelineModel != nil {
			description = fmt.Sprintf(
				"Run the pipeline %q manually. Pass params as an object. If that pipeline reaches its Return node, the returned data will be included in the tool result.",
				pipelineModel.Name,
			)
		}
	}
	if cfg.ToolDescription != "" {
		description = strings.TrimSpace(cfg.ToolDescription)
	}

	properties := map[string]any{
		"params": map[string]any{
			"type":        "object",
			"description": "Optional input object passed to the target pipeline as manual execution parameters.",
		},
	}
	required := make([]string, 0, 1)
	if cfg.AllowModelPipelineID {
		properties["pipelineId"] = map[string]any{
			"type":        "string",
			"description": "Pipeline ID to run. Use this when you need to choose the target pipeline dynamically.",
		}
		if strings.TrimSpace(cfg.PipelineID) == "" {
			required = append(required, "pipelineId")
		}
	}

	parameters := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		parameters["required"] = required
	}

	return &llm.ToolDefinition{
		Type: "function",
		Function: llm.ToolSpec{
			Name:        sanitizeToolName(strings.TrimSpace(cfg.ToolName), sanitizeToolName(meta.Label, "run_pipeline")),
			Description: description,
			Parameters:  parameters,
		},
	}, nil
}

func (e *PipelineRunToolNode) ExecuteTool(ctx context.Context, config json.RawMessage, args json.RawMessage, _ map[string]any) (any, error) {
	if e.Runner == nil {
		return nil, fmt.Errorf("pipeline runner is not configured")
	}

	cfg, err := parseRunPipelineToolConfig(config)
	if err != nil {
		return nil, err
	}

	toolArgs, err := parseRunPipelineToolArgs(args)
	if err != nil {
		return nil, err
	}

	targetPipelineID := strings.TrimSpace(cfg.PipelineID)
	if cfg.AllowModelPipelineID && strings.TrimSpace(toolArgs.PipelineID) != "" {
		targetPipelineID = strings.TrimSpace(toolArgs.PipelineID)
	}
	if targetPipelineID == "" {
		return nil, fmt.Errorf("pipelineId is required")
	}

	result, err := e.Runner.Run(ctx, targetPipelineID, toolArgs.Params)
	if err != nil {
		return nil, fmt.Errorf("run pipeline %s: %w", targetPipelineID, err)
	}

	return buildRunPipelineOutput(result), nil
}

func parseRunPipelineConfig(config json.RawMessage, input map[string]any) (runPipelineConfig, error) {
	var cfg runPipelineConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return runPipelineConfig{}, fmt.Errorf("parse config: %w", err)
	}
	if err := templating.RenderStrings(&cfg, input); err != nil {
		return runPipelineConfig{}, fmt.Errorf("render config: %w", err)
	}
	if strings.TrimSpace(cfg.PipelineID) == "" {
		return runPipelineConfig{}, fmt.Errorf("pipelineId is required")
	}
	return cfg, nil
}

func parseRunPipelineToolConfig(config json.RawMessage) (runPipelineConfig, error) {
	var cfg runPipelineConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return runPipelineConfig{}, fmt.Errorf("parse config: %w", err)
	}
	if strings.TrimSpace(cfg.PipelineID) == "" && !cfg.AllowModelPipelineID {
		return runPipelineConfig{}, fmt.Errorf("pipelineId is required")
	}
	return cfg, nil
}

func parsePipelineParams(raw string, fallback map[string]any) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return copyParamsMap(fallback), nil
	}

	var decoded any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, fmt.Errorf("parse params JSON: %w", err)
	}

	params, ok := decoded.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("params must be a JSON object")
	}

	return params, nil
}

func parseToolParams(args json.RawMessage) (map[string]any, error) {
	if len(args) == 0 {
		return make(map[string]any), nil
	}

	var payload struct {
		Params map[string]any `json:"params"`
	}
	if err := json.Unmarshal(args, &payload); err != nil {
		return nil, fmt.Errorf("parse tool args: %w", err)
	}
	if payload.Params == nil {
		return make(map[string]any), nil
	}

	return payload.Params, nil
}

type runPipelineToolArgs struct {
	PipelineID string         `json:"pipelineId"`
	Params     map[string]any `json:"params"`
}

func parseRunPipelineToolArgs(args json.RawMessage) (runPipelineToolArgs, error) {
	if len(args) == 0 {
		return runPipelineToolArgs{Params: make(map[string]any)}, nil
	}

	var payload runPipelineToolArgs
	if err := json.Unmarshal(args, &payload); err != nil {
		return runPipelineToolArgs{}, fmt.Errorf("parse tool args: %w", err)
	}
	if payload.Params == nil {
		payload.Params = make(map[string]any)
	}

	return payload, nil
}

func parsePipelineListToolArgs(args json.RawMessage) (pipelineListToolArgs, error) {
	if len(args) == 0 {
		return pipelineListToolArgs{}, nil
	}

	var payload pipelineListToolArgs
	if err := json.Unmarshal(args, &payload); err != nil {
		return pipelineListToolArgs{}, fmt.Errorf("parse tool args: %w", err)
	}

	return payload, nil
}

func buildRunPipelineOutput(result *pipeline.RunResult) map[string]any {
	if result == nil {
		return map[string]any{
			"status": "completed",
		}
	}

	output := map[string]any{
		"execution_id":  result.ExecutionID,
		"pipeline_id":   result.PipelineID,
		"pipeline_name": result.PipelineName,
		"status":        result.Status,
		"nodes_run":     result.NodesRun,
		"returned":      result.Returned,
	}
	if result.Returned {
		output["return_value"] = result.ReturnValue
	}

	return output
}

func copyParamsMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return make(map[string]any)
	}

	copied := make(map[string]any, len(input))
	for key, value := range input {
		copied[key] = value
	}

	return copied
}

func sanitizeToolName(label string, fallback string) string {
	base := strings.TrimSpace(label)
	if base == "" {
		base = fallback
	}

	var builder strings.Builder
	lastUnderscore := false
	for _, r := range strings.ToLower(base) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			lastUnderscore = false
		case !lastUnderscore:
			builder.WriteRune('_')
			lastUnderscore = true
		}
	}

	name := strings.Trim(builder.String(), "_")
	if name == "" {
		name = fallback
	}
	if len(name) > 0 && unicode.IsDigit(rune(name[0])) {
		name = "tool_" + name
	}

	return name
}
