package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

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
	providerStore   *query.LLMProviderStore
	clusterStore    *query.ClusterStore
	kubernetesStore *query.KubernetesClusterStore
	pipelineStore   *query.PipelineStore
	runner          *pipeline.ExecutionRunner
	scheduler       llm.ToolPipelineReloader
	skillStore      skills.Reader
	shellRunner     shellcmd.Runner
}

func NewLLMChatHandler(
	providerStore *query.LLMProviderStore,
	clusterStore *query.ClusterStore,
	kubernetesStore *query.KubernetesClusterStore,
	pipelineStore *query.PipelineStore,
	runner *pipeline.ExecutionRunner,
	scheduler llm.ToolPipelineReloader,
	skillStore skills.Reader,
	shellRunner shellcmd.Runner,
) *LLMChatHandler {
	return &LLMChatHandler{
		providerStore:   providerStore,
		clusterStore:    clusterStore,
		kubernetesStore: kubernetesStore,
		pipelineStore:   pipelineStore,
		runner:          runner,
		scheduler:       scheduler,
		skillStore:      skillStore,
		shellRunner:     shellRunner,
	}
}

func (h *LLMChatHandler) Chat(c *fiber.Ctx) error {
	var req struct {
		Message      string `json:"message"`
		ProviderID   string `json:"provider_id,omitempty"`
		ClusterID    string `json:"cluster_id,omitempty"`
		Integrations struct {
			Proxmox struct {
				Enabled   *bool  `json:"enabled,omitempty"`
				ClusterID string `json:"cluster_id,omitempty"`
			} `json:"proxmox"`
			Kubernetes struct {
				Enabled   *bool  `json:"enabled,omitempty"`
				ClusterID string `json:"cluster_id,omitempty"`
			} `json:"kubernetes"`
		} `json:"integrations"`
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

	proxmoxExplicit := req.Integrations.Proxmox.Enabled != nil || strings.TrimSpace(req.Integrations.Proxmox.ClusterID) != ""
	kubernetesExplicit := req.Integrations.Kubernetes.Enabled != nil || strings.TrimSpace(req.Integrations.Kubernetes.ClusterID) != ""

	proxmoxEnabled := true
	if proxmoxExplicit {
		proxmoxEnabled = false
		if req.Integrations.Proxmox.Enabled != nil {
			proxmoxEnabled = *req.Integrations.Proxmox.Enabled
		}
		if strings.TrimSpace(req.Integrations.Proxmox.ClusterID) != "" {
			proxmoxEnabled = true
		}
	}

	kubernetesEnabled := false
	if kubernetesExplicit {
		if req.Integrations.Kubernetes.Enabled != nil {
			kubernetesEnabled = *req.Integrations.Kubernetes.Enabled
		}
		if strings.TrimSpace(req.Integrations.Kubernetes.ClusterID) != "" {
			kubernetesEnabled = true
		}
	}

	selectedProxmoxClusterID := strings.TrimSpace(req.Integrations.Proxmox.ClusterID)
	if selectedProxmoxClusterID == "" && strings.TrimSpace(req.ClusterID) != "" && !proxmoxExplicit {
		selectedProxmoxClusterID = strings.TrimSpace(req.ClusterID)
		proxmoxEnabled = true
	}
	selectedKubernetesClusterID := strings.TrimSpace(req.Integrations.Kubernetes.ClusterID)

	var selectedProxmoxCluster *models.Cluster
	if proxmoxEnabled && selectedProxmoxClusterID != "" {
		cluster, err := h.clusterStore.GetByID(ctx, selectedProxmoxClusterID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "cluster not found",
			})
		}
		selectedProxmoxCluster = cluster
	}

	var selectedKubernetesCluster *models.KubernetesCluster
	if kubernetesEnabled && selectedKubernetesClusterID != "" {
		cluster, err := h.kubernetesStore.GetByID(ctx, selectedKubernetesClusterID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "kubernetes cluster not found",
			})
		}
		selectedKubernetesCluster = cluster
	}

	toolRegistry := llm.NewToolRegistryWithOptions(llm.ToolRegistryOptions{
		ProxmoxStore:               h.clusterStore,
		KubernetesStore:            h.kubernetesStore,
		PipelineStore:              h.pipelineStore,
		PipelineReloader:           h.scheduler,
		PipelineRunner:             &chatPipelineRunner{store: h.pipelineStore, runner: h.runner},
		DefaultProxmoxClusterID:    selectedProxmoxClusterID,
		DefaultKubernetesClusterID: selectedKubernetesClusterID,
		EnableProxmox:              proxmoxEnabled,
		EnableKubernetes:           kubernetesEnabled,
		SkillStore:                 h.skillStore,
		ShellRunner:                h.shellRunner,
	})

	systemPrompt := "You are an automation assistant for infrastructure and Automator pipelines. Use the available tools to manage enabled integrations, inspect local skills, run shell commands when appropriate, and create, edit, run, activate, or deactivate pipelines when the user asks."
	integrationStatements := make([]string, 0, 2)
	if proxmoxEnabled {
		if selectedProxmoxCluster != nil {
			integrationStatements = append(integrationStatements, "Proxmox integration is enabled and the selected cluster is "+selectedProxmoxCluster.Name+" ("+selectedProxmoxCluster.ID+"). Use it for Proxmox tool calls unless the user explicitly asks for a different configured cluster.")
		} else {
			integrationStatements = append(integrationStatements, "Proxmox integration is enabled.")
		}
	}
	if kubernetesEnabled {
		if selectedKubernetesCluster != nil {
			integrationStatements = append(integrationStatements, "Kubernetes integration is enabled and the selected cluster is "+selectedKubernetesCluster.Name+" ("+selectedKubernetesCluster.ID+") using context "+selectedKubernetesCluster.ContextName+". Use it for Kubernetes tool calls unless the user explicitly asks for a different configured cluster.")
		} else {
			integrationStatements = append(integrationStatements, "Kubernetes integration is enabled.")
		}
	}
	if len(integrationStatements) == 0 {
		systemPrompt += " No infrastructure integration is enabled for this chat, so only local, pipeline, and skill tools are available."
	} else {
		systemPrompt += " " + strings.Join(integrationStatements, " ")
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
