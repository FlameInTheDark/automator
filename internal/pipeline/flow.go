package pipeline

import (
	"encoding/json"
	"fmt"
)

func ParseFlowData(nodesJSON string, edgesJSON string) (*FlowData, error) {
	var flowData FlowData
	if err := json.Unmarshal([]byte(nodesJSON), &flowData.Nodes); err != nil {
		return nil, fmt.Errorf("unmarshal nodes: %w", err)
	}
	if err := json.Unmarshal([]byte(edgesJSON), &flowData.Edges); err != nil {
		return nil, fmt.Errorf("unmarshal edges: %w", err)
	}
	return &flowData, nil
}
