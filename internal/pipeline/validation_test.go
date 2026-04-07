package pipeline

import (
	"strings"
	"testing"
)

func TestValidateFlowDataRejectsToolNodeInMainExecutionChain(t *testing.T) {
	t.Parallel()

	flowData := FlowData{
		Nodes: []FlowNode{
			{ID: "trigger-1", Data: []byte(`{"type":"trigger:manual"}`)},
			{ID: "tool-1", Data: []byte(`{"type":"tool:http"}`)},
		},
		Edges: []FlowEdge{
			{ID: "edge-1", Source: "trigger-1", Target: "tool-1"},
		},
	}

	err := ValidateFlowData(flowData)
	if err == nil {
		t.Fatal("expected validation to fail")
	}
	if !strings.Contains(err.Error(), "cannot be part of the main execution chain") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateFlowDataRejectsReturnNodeOutgoingEdge(t *testing.T) {
	t.Parallel()

	flowData := FlowData{
		Nodes: []FlowNode{
			{ID: "return-1", Data: []byte(`{"type":"logic:return"}`)},
			{ID: "action-1", Data: []byte(`{"type":"action:http"}`)},
		},
		Edges: []FlowEdge{
			{ID: "edge-1", Source: "return-1", Target: "action-1"},
		},
	}

	err := ValidateFlowData(flowData)
	if err == nil {
		t.Fatal("expected validation to fail")
	}
	if !strings.Contains(err.Error(), "cannot have outgoing edges") {
		t.Fatalf("unexpected error: %v", err)
	}
}
