package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"

	"github.com/FlameInTheDark/automator/internal/db/models"
	"github.com/FlameInTheDark/automator/internal/db/query"
	"github.com/FlameInTheDark/automator/internal/llm"
	"github.com/FlameInTheDark/automator/internal/pipeline"
	"github.com/FlameInTheDark/automator/internal/shellcmd"
	"github.com/FlameInTheDark/automator/internal/skills"
)

type PipelineRunHandler struct {
	pipelineStore *query.PipelineStore
	runner        *pipeline.ExecutionRunner
}

func NewPipelineRunHandler(
	pipelineStore *query.PipelineStore,
	runner *pipeline.ExecutionRunner,
) *PipelineRunHandler {
	return &PipelineRunHandler{
		pipelineStore: pipelineStore,
		runner:        runner,
	}
}

func (h *PipelineRunHandler) Run(c *fiber.Ctx) error {
	id := c.Params("id")
	ctx := c.Context()

	p, err := h.pipelineStore.GetByID(ctx, id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "pipeline not found",
		})
	}

	var flowData pipeline.FlowData
	if err := json.Unmarshal([]byte(p.Nodes), &flowData.Nodes); err != nil {
		log.Printf("failed to parse nodes: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid pipeline nodes format",
		})
	}

	if err := json.Unmarshal([]byte(p.Edges), &flowData.Edges); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid pipeline edges format",
		})
	}

	result, err := h.runner.Run(context.Background(), p.ID, flowData, "manual", nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	response := fiber.Map{
		"execution_id": result.ExecutionID,
		"status":       result.Status,
		"duration":     result.Duration.String(),
		"nodes_run":    result.NodesRun,
	}
	if result.ErrorMessage != "" {
		response["error"] = result.ErrorMessage
	}
	if result.Returned {
		response["returned"] = true
		response["return_value"] = result.ReturnValue
	}

	return c.JSON(response)
}

type ExecutionHandler struct {
	store  *query.ExecutionStore
	runner *pipeline.ExecutionRunner
}

func NewExecutionHandler(store *query.ExecutionStore, runner *pipeline.ExecutionRunner) *ExecutionHandler {
	return &ExecutionHandler{store: store, runner: runner}
}

func (h *ExecutionHandler) ListByPipeline(c *fiber.Ctx) error {
	pipelineID := c.Params("id")
	ctx := c.Context()

	executions, err := h.store.ListByPipeline(ctx, pipelineID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(executions)
}

func (h *ExecutionHandler) Get(c *fiber.Ctx) error {
	executionID := c.Params("executionId")
	ctx := c.Context()

	execution, err := h.store.GetByID(ctx, executionID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "execution not found",
		})
	}

	nodeExecutions, err := h.store.ListByExecution(ctx, executionID)
	if err != nil {
		log.Printf("failed to list node executions: %v", err)
		nodeExecutions = make([]models.NodeExecution, 0)
	}

	return c.JSON(fiber.Map{
		"execution":       execution,
		"node_executions": nodeExecutions,
	})
}

func (h *ExecutionHandler) ListActiveByPipeline(c *fiber.Ctx) error {
	pipelineID := c.Params("id")
	if h.runner == nil {
		return c.JSON([]pipeline.ActiveExecutionInfo{})
	}

	return c.JSON(h.runner.ActiveByPipeline(pipelineID))
}

func (h *ExecutionHandler) Cancel(c *fiber.Ctx) error {
	executionID := c.Params("executionId")
	if h.runner == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "execution is not active",
		})
	}

	info, ok := h.runner.Cancel(executionID)
	if !ok {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "execution is not active",
		})
	}

	return c.JSON(info)
}

type chatPipelineRunner struct {
	store  *query.PipelineStore
	runner *pipeline.ExecutionRunner
}

func (r *chatPipelineRunner) Run(ctx context.Context, pipelineID string, input map[string]any) (*llm.ToolPipelineRunResult, error) {
	if r == nil || r.store == nil {
		return nil, fmt.Errorf("pipeline store is not configured")
	}
	if r.runner == nil {
		return nil, fmt.Errorf("pipeline runner is not configured")
	}

	pipelineModel, err := r.store.GetByID(ctx, pipelineID)
	if err != nil {
		return nil, err
	}

	flowData, err := pipeline.ParseFlowData(pipelineModel.Nodes, pipelineModel.Edges)
	if err != nil {
		return nil, err
	}
	if err := pipeline.ValidateFlowData(*flowData); err != nil {
		return nil, err
	}

	result, err := r.runner.Run(ctx, pipelineModel.ID, *flowData, "manual", copyToolExecutionInput(input))
	if err != nil {
		return nil, err
	}
	if result.Status == "failed" {
		if result.Error != nil {
			return nil, result.Error
		}
		return nil, fmt.Errorf("%s", result.ErrorMessage)
	}
	if result.Status == "cancelled" {
		return nil, fmt.Errorf("%s", result.ErrorMessage)
	}

	return &llm.ToolPipelineRunResult{
		ExecutionID:  result.ExecutionID,
		PipelineID:   pipelineModel.ID,
		PipelineName: pipelineModel.Name,
		Status:       result.Status,
		NodesRun:     result.NodesRun,
		Returned:     result.Returned,
		ReturnValue:  result.ReturnValue,
	}, nil
}

func copyToolExecutionInput(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}

	result := make(map[string]any, len(input))
	for key, value := range input {
		result[key] = value
	}

	return result
}

type LLMChatHandler struct {
	providerStore *query.LLMProviderStore
	clusterStore  *query.ClusterStore
	pipelineStore *query.PipelineStore
	runner        *pipeline.ExecutionRunner
	scheduler     llm.ToolPipelineReloader
	skillStore    skills.Reader
	shellRunner   shellcmd.Runner
}

func NewLLMChatHandler(
	providerStore *query.LLMProviderStore,
	clusterStore *query.ClusterStore,
	pipelineStore *query.PipelineStore,
	runner *pipeline.ExecutionRunner,
	scheduler llm.ToolPipelineReloader,
	skillStore skills.Reader,
	shellRunner shellcmd.Runner,
) *LLMChatHandler {
	return &LLMChatHandler{
		providerStore: providerStore,
		clusterStore:  clusterStore,
		pipelineStore: pipelineStore,
		runner:        runner,
		scheduler:     scheduler,
		skillStore:    skillStore,
		shellRunner:   shellRunner,
	}
}

func (h *LLMChatHandler) Chat(c *fiber.Ctx) error {
	var req struct {
		Message    string `json:"message"`
		ProviderID string `json:"provider_id,omitempty"`
		ClusterID  string `json:"cluster_id,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	ctx := c.Context()

	var providerModel *llm.Config
	if req.ProviderID != "" {
		p, err := h.providerStore.GetByID(ctx, req.ProviderID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "provider not found",
			})
		}
		providerConfig, err := llm.ConfigFromModel(p)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid provider configuration: " + err.Error(),
			})
		}
		providerModel = &providerConfig
	} else {
		p, err := h.providerStore.GetDefault(ctx)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "no default LLM provider configured",
			})
		}
		providerConfig, err := llm.ConfigFromModel(p)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid provider configuration: " + err.Error(),
			})
		}
		providerModel = &providerConfig
	}

	provider, err := llm.NewProvider(*providerModel)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "failed to initialize provider: " + err.Error(),
		})
	}

	var selectedCluster *models.Cluster
	if req.ClusterID != "" {
		cluster, err := h.clusterStore.GetByID(ctx, req.ClusterID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "cluster not found",
			})
		}
		selectedCluster = cluster
	}

	toolRegistry := llm.NewToolRegistry(
		h.clusterStore,
		h.pipelineStore,
		h.scheduler,
		&chatPipelineRunner{store: h.pipelineStore, runner: h.runner},
		req.ClusterID,
		h.skillStore,
		h.shellRunner,
	)

	systemPrompt := "You are an automation assistant for Proxmox and Automator pipelines. Use the available tools to manage clusters, inspect local skills, run shell commands when appropriate, and create, edit, run, activate, or deactivate pipelines when the user asks."
	if selectedCluster != nil {
		systemPrompt += " The selected cluster is " + selectedCluster.Name + " (" + selectedCluster.ID + "). Use it for cluster-scoped tool calls unless the user explicitly asks for a different configured cluster."
	}
	if h.skillStore != nil {
		if skillsSummary := h.skillStore.SummaryText(); skillsSummary != "" {
			systemPrompt += " Local skills are available:\n" + skillsSummary + "\nUse the get_skill tool when you need the full contents of one."
			systemPrompt += " For pipeline creation or structural edits, read the pipeline-builder skill before building a new definition."
		}
	}
	if h.shellRunner != nil {
		systemPrompt += " A run_shell_command tool is available when you need to inspect or operate in the local workspace."
	}

	messages := []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: req.Message},
	}

	resp, toolCalls, toolResults, err := runToolChat(ctx, provider, providerModel.Model, messages, toolRegistry)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "LLM request failed: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"content":      resp.Content,
		"tool_calls":   toolCalls,
		"tool_results": toolResults,
		"usage":        resp.Usage,
	})
}
