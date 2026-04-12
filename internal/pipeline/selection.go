package pipeline

import (
	"context"
	"slices"
	"strings"

	"github.com/FlameInTheDark/emerald/internal/node"
	"github.com/FlameInTheDark/emerald/internal/node/trigger"
)

type TriggerSelection struct {
	TriggerType string   `json:"trigger_type"`
	RootNodeIDs []string `json:"root_node_ids,omitempty"`
}

func ResolveTriggerSelection(ctx context.Context, flowData FlowData, selection TriggerSelection) TriggerSelection {
	normalized := normalizeTriggerSelection(selection)
	if len(normalized.RootNodeIDs) > 0 {
		return normalized
	}

	rootIDs, nodeMap := rootExecutableNodeIDs(flowData)
	if len(rootIDs) == 0 {
		return normalized
	}

	matchedRoots := make([]string, 0, len(rootIDs))
	hasRootTrigger := false

	for _, nodeID := range rootIDs {
		nodeType, config := decodeNodeTypeAndConfig(nodeMap[nodeID])
		if !trigger.IsTriggerType(nodeType) {
			continue
		}

		hasRootTrigger = true
		if trigger.MatchesExecution(ctx, nodeType, config, normalized.TriggerType) {
			matchedRoots = append(matchedRoots, nodeID)
		}
	}

	if hasRootTrigger {
		normalized.RootNodeIDs = matchedRoots
		return normalized
	}

	if normalized.TriggerType == "manual" {
		normalized.RootNodeIDs = append([]string(nil), rootIDs...)
	}

	return normalized
}

func HasMatchingRootTrigger(ctx context.Context, flowData FlowData, triggerType string) bool {
	return len(ResolveTriggerSelection(ctx, flowData, TriggerSelection{TriggerType: triggerType}).RootNodeIDs) > 0
}

func ReachableNodeIDs(flowData FlowData, rootNodeIDs []string) map[string]struct{} {
	if len(rootNodeIDs) == 0 {
		return map[string]struct{}{}
	}

	adjacency := make(map[string][]string)
	for _, edge := range flowData.Edges {
		if isToolEdge(edge) {
			continue
		}
		adjacency[edge.Source] = append(adjacency[edge.Source], edge.Target)
	}

	seen := make(map[string]struct{}, len(rootNodeIDs))
	queue := append([]string(nil), rootNodeIDs...)
	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]

		if _, ok := seen[currentID]; ok {
			continue
		}
		seen[currentID] = struct{}{}

		for _, targetID := range adjacency[currentID] {
			if _, ok := seen[targetID]; ok {
				continue
			}
			queue = append(queue, targetID)
		}
	}

	return seen
}

func rootExecutableNodeIDs(flowData FlowData) ([]string, map[string]FlowNode) {
	inDegree := make(map[string]int, len(flowData.Nodes))
	nodeMap := make(map[string]FlowNode, len(flowData.Nodes))
	toolTargets := collectToolTargets(flowData.Edges)

	for _, flowNode := range flowData.Nodes {
		inDegree[flowNode.ID] = 0
		nodeMap[flowNode.ID] = flowNode
	}

	for _, edge := range flowData.Edges {
		if isToolEdge(edge) {
			continue
		}
		inDegree[edge.Target]++
	}

	rootIDs := make([]string, 0, len(flowData.Nodes))
	for _, flowNode := range flowData.Nodes {
		if inDegree[flowNode.ID] != 0 {
			continue
		}
		if _, isToolNode := toolTargets[flowNode.ID]; isToolNode {
			continue
		}

		nodeType, _ := decodeNodeTypeAndConfig(flowNode)
		if isToolNodeType(nodeType) || isVisualNodeType(nodeType) {
			continue
		}

		rootIDs = append(rootIDs, flowNode.ID)
	}

	slices.Sort(rootIDs)
	return rootIDs, nodeMap
}

func normalizeTriggerSelection(selection TriggerSelection) TriggerSelection {
	normalized := TriggerSelection{
		TriggerType: strings.TrimSpace(selection.TriggerType),
	}

	if len(selection.RootNodeIDs) == 0 {
		return normalized
	}

	seen := make(map[string]struct{}, len(selection.RootNodeIDs))
	for _, nodeID := range selection.RootNodeIDs {
		trimmed := strings.TrimSpace(nodeID)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized.RootNodeIDs = append(normalized.RootNodeIDs, trimmed)
	}

	slices.Sort(normalized.RootNodeIDs)
	return normalized
}

func TriggerSelectionFromNodeIDs(triggerType string, nodeIDs []string) TriggerSelection {
	return normalizeTriggerSelection(TriggerSelection{
		TriggerType: triggerType,
		RootNodeIDs: nodeIDs,
	})
}

func IsSelectedRootNode(selection TriggerSelection, nodeID string) bool {
	if len(selection.RootNodeIDs) == 0 {
		return false
	}

	trimmed := strings.TrimSpace(nodeID)
	for _, selectedID := range selection.RootNodeIDs {
		if selectedID == trimmed {
			return true
		}
	}

	return false
}

func IsPluginTriggerNodeType(nodeType node.NodeType) bool {
	return strings.HasPrefix(strings.TrimSpace(string(nodeType)), "trigger:plugin/")
}
