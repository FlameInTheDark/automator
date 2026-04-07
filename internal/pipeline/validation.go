package pipeline

import (
	"fmt"
	"strings"

	"github.com/FlameInTheDark/automator/internal/node"
)

func ValidateFlowData(flowData FlowData) error {
	if err := validateReturnNodes(flowData); err != nil {
		return err
	}

	return validateFlowEdges(flowData)
}

func validateReturnNodes(flowData FlowData) error {
	count := 0

	for _, flowNode := range flowData.Nodes {
		nodeType, _ := decodeNodeTypeAndConfig(flowNode)
		if nodeType == node.TypeLogicReturn {
			count++
		}
	}

	if count > 1 {
		return fmt.Errorf("only one Return node is allowed per pipeline")
	}

	return nil
}

func validateFlowEdges(flowData FlowData) error {
	nodeTypes := make(map[string]node.NodeType, len(flowData.Nodes))

	for _, flowNode := range flowData.Nodes {
		nodeID := strings.TrimSpace(flowNode.ID)
		if nodeID == "" {
			return fmt.Errorf("all nodes must have an id")
		}
		if _, exists := nodeTypes[nodeID]; exists {
			return fmt.Errorf("duplicate node id %q", nodeID)
		}

		nodeType, _ := decodeNodeTypeAndConfig(flowNode)
		nodeTypes[nodeID] = nodeType
	}

	for _, edge := range flowData.Edges {
		if err := validateFlowEdge(edge, nodeTypes); err != nil {
			return err
		}
	}

	return nil
}

func validateFlowEdge(edge FlowEdge, nodeTypes map[string]node.NodeType) error {
	edgeID := strings.TrimSpace(edge.ID)
	if edgeID == "" {
		edgeID = fmt.Sprintf("%s->%s", edge.Source, edge.Target)
	}

	sourceID := strings.TrimSpace(edge.Source)
	targetID := strings.TrimSpace(edge.Target)
	if sourceID == "" || targetID == "" {
		return fmt.Errorf("edge %q must include source and target", edgeID)
	}

	sourceType, ok := nodeTypes[sourceID]
	if !ok {
		return fmt.Errorf("edge %q references unknown source node %q", edgeID, sourceID)
	}
	targetType, ok := nodeTypes[targetID]
	if !ok {
		return fmt.Errorf("edge %q references unknown target node %q", edgeID, targetID)
	}

	if isToolEdge(edge) {
		if sourceType != node.TypeLLMAgent {
			return fmt.Errorf("edge %q uses the tool handle, but source node %q is %q instead of %s", edgeID, sourceID, sourceType, node.TypeLLMAgent)
		}
		if !isToolNodeType(targetType) {
			return fmt.Errorf("edge %q uses the tool handle, but target node %q is %q instead of a tool node", edgeID, targetID, targetType)
		}
		return nil
	}

	if isToolNodeType(sourceType) {
		return fmt.Errorf("tool node %q (%s) cannot be part of the main execution chain; connect it from an LLM Agent tool handle instead", sourceID, sourceType)
	}
	if isToolNodeType(targetType) {
		return fmt.Errorf("tool node %q (%s) cannot be part of the main execution chain; connect it from an LLM Agent tool handle instead", targetID, targetType)
	}
	if sourceType == node.TypeLogicReturn {
		return fmt.Errorf("Return node %q cannot have outgoing edges", sourceID)
	}
	if strings.HasPrefix(strings.TrimSpace(string(targetType)), "trigger:") {
		return fmt.Errorf("trigger node %q (%s) cannot have incoming edges", targetID, targetType)
	}

	return nil
}
