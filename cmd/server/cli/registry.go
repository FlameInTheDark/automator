package cli

import (
	"github.com/FlameInTheDark/emerald/internal/channels"
	"github.com/FlameInTheDark/emerald/internal/node"
	"github.com/FlameInTheDark/emerald/internal/node/action"
	"github.com/FlameInTheDark/emerald/internal/node/logic"
	"github.com/FlameInTheDark/emerald/internal/node/lua"
	"github.com/FlameInTheDark/emerald/internal/node/trigger"
	"github.com/FlameInTheDark/emerald/internal/plugins"
	"github.com/FlameInTheDark/emerald/internal/shellcmd"
	"github.com/FlameInTheDark/emerald/internal/skills"
	"github.com/FlameInTheDark/emerald/pkg/pluginapi"
)

type registryDependencies struct {
	ClusterStore           action.ClusterStore
	KubernetesClusterStore action.KubernetesClusterStore
	LLMProviderStore       logic.LLMProviderStore
	ChannelStore           action.ChannelStore
	ChannelContactStore    action.ChannelContactStore
	ChannelService         *channels.Service
	PipelineStore          action.PipelineCatalog
	PipelineRunner         action.PipelineRunner
	PipelineManager        action.PipelineMutationManager
	SkillStore             skills.Reader
	ShellRunner            shellcmd.Runner
	PluginManager          *plugins.Manager
}

func buildNodeRegistry(deps registryDependencies) *node.Registry {
	registry := node.NewRegistry()

	registerTriggerNodes(registry)
	registerLogicNodes(registry)
	registerLLMAndLuaNodes(registry, deps)
	registerCommunicationNodes(registry, deps)
	registerProxmoxNodes(registry, deps)
	registerKubernetesNodes(registry, deps)
	registerShellAndHTTPNodes(registry, deps)
	registerPipelineNodes(registry, deps)
	registerPluginResolver(registry, deps.PluginManager)

	return registry
}

func registerTriggerNodes(registry *node.Registry) {
	registry.Register(node.TypeTriggerManual, &trigger.ManualTrigger{})
	registry.Register(node.TypeTriggerCron, &trigger.CronTrigger{})
	registry.Register(node.TypeTriggerWebhook, &trigger.WebhookTrigger{})
	registry.Register(node.TypeTriggerChannel, &trigger.ChannelMessageTrigger{})
}

func registerLogicNodes(registry *node.Registry) {
	registry.Register(node.TypeLogicReturn, &logic.ReturnNode{})
	registry.Register(node.TypeLogicCondition, &logic.ConditionNode{})
	registry.Register(node.TypeLogicSwitch, &logic.SwitchNode{})
	registry.Register(node.TypeLogicMerge, &logic.MergeNode{})
	registry.Register(node.TypeLogicAggregate, &logic.AggregateNode{})
	registry.Register(node.TypeLogicSort, &logic.SortNode{})
	registry.Register(node.TypeLogicLimit, &logic.LimitNode{})
	registry.Register(node.TypeLogicRemoveDuplicates, &logic.RemoveDuplicatesNode{})
	registry.Register(node.TypeLogicSummarize, &logic.SummarizeNode{})
}

func registerLLMAndLuaNodes(registry *node.Registry, deps registryDependencies) {
	llmPromptNode := &logic.LLMPromptNode{Providers: deps.LLMProviderStore}
	registry.Register(node.TypeLLMPrompt, llmPromptNode)
	registry.Register(node.TypeLLMPromptLegacy, llmPromptNode)
	registry.Register(node.TypeLLMAgent, &logic.LLMAgentNode{
		Providers: deps.LLMProviderStore,
		Skills:    deps.SkillStore,
	})
	registry.Register(node.TypeActionLua, &lua.LuaNode{})
}

func registerCommunicationNodes(registry *node.Registry, deps registryDependencies) {
	registry.Register(node.TypeActionChannelSend, &action.ChannelSendAction{
		Channels: deps.ChannelStore,
		Contacts: deps.ChannelContactStore,
		Sender:   deps.ChannelService,
	})
	registry.Register(node.TypeActionChannelReply, &action.ChannelReplyAction{
		Channels: deps.ChannelStore,
		Contacts: deps.ChannelContactStore,
		Sender:   deps.ChannelService,
	})
	registry.Register(node.TypeActionChannelEdit, &action.ChannelEditAction{
		Channels: deps.ChannelStore,
		Contacts: deps.ChannelContactStore,
		Sender:   deps.ChannelService,
	})
	registry.Register(node.TypeActionChannelWait, &action.ChannelSendAndWaitAction{
		Channels: deps.ChannelStore,
		Contacts: deps.ChannelContactStore,
		Sender:   deps.ChannelService,
		Waiter:   deps.ChannelService,
	})
	registry.Register(node.TypeToolChannelWait, &action.ChannelSendAndWaitToolNode{
		Channels: deps.ChannelStore,
		Contacts: deps.ChannelContactStore,
		Sender:   deps.ChannelService,
		Waiter:   deps.ChannelService,
	})
}

func registerProxmoxNodes(registry *node.Registry, deps registryDependencies) {
	registry.Register(node.TypeActionListNodes, &action.ListNodesAction{Clusters: deps.ClusterStore})
	registry.Register(node.TypeActionListVMsCTs, &action.ListVMsCTsAction{Clusters: deps.ClusterStore})
	registry.Register(node.TypeActionVMStart, &action.VMStartAction{Clusters: deps.ClusterStore})
	registry.Register(node.TypeActionVMStop, &action.VMStopAction{Clusters: deps.ClusterStore})
	registry.Register(node.TypeActionVMClone, &action.VMCloneAction{Clusters: deps.ClusterStore})

	registry.Register(node.TypeToolListNodes, &action.ListNodesToolNode{Clusters: deps.ClusterStore})
	registry.Register(node.TypeToolListVMsCTs, &action.ListVMsCTsToolNode{Clusters: deps.ClusterStore})
	registry.Register(node.TypeToolVMStart, &action.VMStartToolNode{Clusters: deps.ClusterStore})
	registry.Register(node.TypeToolVMStop, &action.VMStopToolNode{Clusters: deps.ClusterStore})
	registry.Register(node.TypeToolVMClone, &action.VMCloneToolNode{Clusters: deps.ClusterStore})
}

func registerKubernetesNodes(registry *node.Registry, deps registryDependencies) {
	registry.Register(node.TypeActionKubernetesAPIResources, action.NewKubernetesActionNode(deps.KubernetesClusterStore, action.KubernetesOperationAPIResources))
	registry.Register(node.TypeActionKubernetesListResources, action.NewKubernetesActionNode(deps.KubernetesClusterStore, action.KubernetesOperationListResources))
	registry.Register(node.TypeActionKubernetesGetResource, action.NewKubernetesActionNode(deps.KubernetesClusterStore, action.KubernetesOperationGetResource))
	registry.Register(node.TypeActionKubernetesApplyManifest, action.NewKubernetesActionNode(deps.KubernetesClusterStore, action.KubernetesOperationApplyManifest))
	registry.Register(node.TypeActionKubernetesPatchResource, action.NewKubernetesActionNode(deps.KubernetesClusterStore, action.KubernetesOperationPatchResource))
	registry.Register(node.TypeActionKubernetesDeleteResource, action.NewKubernetesActionNode(deps.KubernetesClusterStore, action.KubernetesOperationDeleteResource))
	registry.Register(node.TypeActionKubernetesScaleResource, action.NewKubernetesActionNode(deps.KubernetesClusterStore, action.KubernetesOperationScaleResource))
	registry.Register(node.TypeActionKubernetesRolloutRestart, action.NewKubernetesActionNode(deps.KubernetesClusterStore, action.KubernetesOperationRolloutRestart))
	registry.Register(node.TypeActionKubernetesRolloutStatus, action.NewKubernetesActionNode(deps.KubernetesClusterStore, action.KubernetesOperationRolloutStatus))
	registry.Register(node.TypeActionKubernetesPodLogs, action.NewKubernetesActionNode(deps.KubernetesClusterStore, action.KubernetesOperationPodLogs))
	registry.Register(node.TypeActionKubernetesPodExec, action.NewKubernetesActionNode(deps.KubernetesClusterStore, action.KubernetesOperationPodExec))
	registry.Register(node.TypeActionKubernetesEvents, action.NewKubernetesActionNode(deps.KubernetesClusterStore, action.KubernetesOperationEvents))

	registry.Register(node.TypeToolKubernetesAPIResources, action.NewKubernetesToolNode(deps.KubernetesClusterStore, action.KubernetesOperationAPIResources))
	registry.Register(node.TypeToolKubernetesListResources, action.NewKubernetesToolNode(deps.KubernetesClusterStore, action.KubernetesOperationListResources))
	registry.Register(node.TypeToolKubernetesGetResource, action.NewKubernetesToolNode(deps.KubernetesClusterStore, action.KubernetesOperationGetResource))
	registry.Register(node.TypeToolKubernetesApplyManifest, action.NewKubernetesToolNode(deps.KubernetesClusterStore, action.KubernetesOperationApplyManifest))
	registry.Register(node.TypeToolKubernetesPatchResource, action.NewKubernetesToolNode(deps.KubernetesClusterStore, action.KubernetesOperationPatchResource))
	registry.Register(node.TypeToolKubernetesDeleteResource, action.NewKubernetesToolNode(deps.KubernetesClusterStore, action.KubernetesOperationDeleteResource))
	registry.Register(node.TypeToolKubernetesScaleResource, action.NewKubernetesToolNode(deps.KubernetesClusterStore, action.KubernetesOperationScaleResource))
	registry.Register(node.TypeToolKubernetesRolloutRestart, action.NewKubernetesToolNode(deps.KubernetesClusterStore, action.KubernetesOperationRolloutRestart))
	registry.Register(node.TypeToolKubernetesRolloutStatus, action.NewKubernetesToolNode(deps.KubernetesClusterStore, action.KubernetesOperationRolloutStatus))
	registry.Register(node.TypeToolKubernetesPodLogs, action.NewKubernetesToolNode(deps.KubernetesClusterStore, action.KubernetesOperationPodLogs))
	registry.Register(node.TypeToolKubernetesPodExec, action.NewKubernetesToolNode(deps.KubernetesClusterStore, action.KubernetesOperationPodExec))
	registry.Register(node.TypeToolKubernetesEvents, action.NewKubernetesToolNode(deps.KubernetesClusterStore, action.KubernetesOperationEvents))
}

func registerShellAndHTTPNodes(registry *node.Registry, deps registryDependencies) {
	registry.Register(node.TypeActionHTTP, &action.HTTPAction{})
	registry.Register(node.TypeActionShell, &action.ShellCommandAction{Runner: deps.ShellRunner})
	registry.Register(node.TypeToolHTTP, &action.HTTPToolNode{})
	registry.Register(node.TypeToolShell, &action.ShellCommandToolNode{Runner: deps.ShellRunner})
}

func registerPipelineNodes(registry *node.Registry, deps registryDependencies) {
	registry.Register(node.TypeActionGetPipeline, &action.GetPipelineAction{Pipelines: deps.PipelineStore})
	registry.Register(node.TypeActionRunPipeline, &action.RunPipelineAction{Runner: deps.PipelineRunner})

	registry.Register(node.TypeToolListPipelines, &action.PipelineListToolNode{Pipelines: deps.PipelineStore})
	registry.Register(node.TypeToolGetPipeline, &action.PipelineGetToolNode{Pipelines: deps.PipelineStore})
	registry.Register(node.TypeToolCreatePipeline, &action.PipelineCreateToolNode{Manager: deps.PipelineManager})
	registry.Register(node.TypeToolUpdatePipeline, &action.PipelineUpdateToolNode{Manager: deps.PipelineManager})
	registry.Register(node.TypeToolDeletePipeline, &action.PipelineDeleteToolNode{Manager: deps.PipelineManager})
	registry.Register(node.TypeToolRunPipeline, &action.PipelineRunToolNode{
		Pipelines: deps.PipelineStore,
		Runner:    deps.PipelineRunner,
	})
}

func registerPluginResolver(registry *node.Registry, pluginManager *plugins.Manager) {
	if pluginManager == nil {
		return
	}

	registry.SetDynamicResolver(func(nodeType node.NodeType) (node.NodeExecutor, bool) {
		binding, ok := pluginManager.Binding(string(nodeType))
		if !ok {
			return nil, false
		}

		switch binding.Kind {
		case pluginapi.NodeKindTrigger:
			return &plugins.TriggerExecutor{
				Manager:  pluginManager,
				NodeType: binding.Type,
			}, true
		case pluginapi.NodeKindTool:
			return &plugins.ToolExecutor{
				Manager:  pluginManager,
				NodeType: binding.Type,
			}, true
		default:
			return &plugins.ActionExecutor{
				Manager:  pluginManager,
				NodeType: binding.Type,
				Outputs:  append([]pluginapi.OutputHandle(nil), binding.Spec.Outputs...),
			}, true
		}
	})
}
