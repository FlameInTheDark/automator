package node

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/FlameInTheDark/automator/internal/llm"
)

type NodeType string

const (
	TypeTriggerManual      NodeType = "trigger:manual"
	TypeTriggerCron        NodeType = "trigger:cron"
	TypeTriggerWebhook     NodeType = "trigger:webhook"
	TypeTriggerChannel     NodeType = "trigger:channel_message"
	TypeActionListNodes    NodeType = "action:proxmox_list_nodes"
	TypeActionListVMsCTs   NodeType = "action:proxmox_list_workloads"
	TypeActionVMStart      NodeType = "action:vm_start"
	TypeActionVMStop       NodeType = "action:vm_stop"
	TypeActionVMClone      NodeType = "action:vm_clone"
	TypeActionHTTP         NodeType = "action:http"
	TypeActionShell        NodeType = "action:shell_command"
	TypeActionLua          NodeType = "action:lua"
	TypeActionChannelSend  NodeType = "action:channel_send_message"
	TypeActionChannelWait  NodeType = "action:channel_send_and_wait"
	TypeActionRunPipeline  NodeType = "action:pipeline_run"
	TypeToolListNodes      NodeType = "tool:proxmox_list_nodes"
	TypeToolListVMsCTs     NodeType = "tool:proxmox_list_workloads"
	TypeToolVMStart        NodeType = "tool:vm_start"
	TypeToolVMStop         NodeType = "tool:vm_stop"
	TypeToolVMClone        NodeType = "tool:vm_clone"
	TypeToolHTTP           NodeType = "tool:http"
	TypeToolShell          NodeType = "tool:shell_command"
	TypeToolListPipelines  NodeType = "tool:pipeline_list"
	TypeToolCreatePipeline NodeType = "tool:pipeline_create"
	TypeToolUpdatePipeline NodeType = "tool:pipeline_update"
	TypeToolDeletePipeline NodeType = "tool:pipeline_delete"
	TypeToolRunPipeline    NodeType = "tool:pipeline_run"
	TypeToolChannelWait    NodeType = "tool:channel_send_and_wait"
	TypeLogicCondition     NodeType = "logic:condition"
	TypeLogicSwitch        NodeType = "logic:switch"
	TypeLogicMerge         NodeType = "logic:merge"
	TypeLogicAggregate     NodeType = "logic:aggregate"
	TypeLogicReturn        NodeType = "logic:return"
	TypeLLMPrompt          NodeType = "llm:prompt"
	TypeLLMPromptLegacy    NodeType = "logic:llm_prompt"
	TypeLLMAgent           NodeType = "llm:agent"
)

type NodeConfig struct {
	ID     string          `json:"id"`
	Type   NodeType        `json:"type"`
	Label  string          `json:"label"`
	Config json.RawMessage `json:"config"`
}

type NodeResult struct {
	Output      json.RawMessage `json:"output"`
	Error       error           `json:"error,omitempty"`
	ReturnValue any             `json:"-"`
}

type NodeExecutor interface {
	Execute(ctx context.Context, config json.RawMessage, input map[string]any) (*NodeResult, error)
	Validate(config json.RawMessage) error
}

type ToolNodeMetadata struct {
	NodeID string
	Label  string
}

type ToolNodeExecutor interface {
	NodeExecutor
	ToolDefinition(ctx context.Context, meta ToolNodeMetadata, config json.RawMessage) (*llm.ToolDefinition, error)
	ExecuteTool(ctx context.Context, config json.RawMessage, args json.RawMessage, input map[string]any) (any, error)
}

type Registry struct {
	executors map[NodeType]NodeExecutor
}

func NewRegistry() *Registry {
	return &Registry{
		executors: make(map[NodeType]NodeExecutor),
	}
}

func (r *Registry) Register(nodeType NodeType, executor NodeExecutor) {
	r.executors[nodeType] = executor
}

func (r *Registry) Get(nodeType NodeType) (NodeExecutor, error) {
	exec, ok := r.executors[nodeType]
	if !ok {
		return nil, fmt.Errorf("unknown node type: %s", nodeType)
	}
	return exec, nil
}

func (r *Registry) ListTypes() []NodeType {
	types := make([]NodeType, 0, len(r.executors))
	for t := range r.executors {
		types = append(types, t)
	}
	return types
}
