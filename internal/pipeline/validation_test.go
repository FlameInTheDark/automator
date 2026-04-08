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

func TestValidateFlowDataAllowsVisualGroupWithoutEdges(t *testing.T) {
	t.Parallel()

	flowData := FlowData{
		Nodes: []FlowNode{
			{ID: "group-1", Data: []byte(`{"type":"visual:group"}`)},
			{ID: "action-1", Type: "action:http"},
		},
	}

	if err := ValidateFlowData(flowData); err != nil {
		t.Fatalf("expected validation to succeed, got %v", err)
	}
}

func TestValidateFlowDataRejectsVisualGroupEdges(t *testing.T) {
	t.Parallel()

	flowData := FlowData{
		Nodes: []FlowNode{
			{ID: "group-1", Data: []byte(`{"type":"visual:group"}`)},
			{ID: "action-1", Data: []byte(`{"type":"action:http"}`)},
		},
		Edges: []FlowEdge{
			{ID: "edge-1", Source: "group-1", Target: "action-1"},
		},
	}

	err := ValidateFlowData(flowData)
	if err == nil {
		t.Fatal("expected validation to fail")
	}
	if !strings.Contains(err.Error(), "visual group node") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateFlowDataRejectsUnsupportedContinueOnErrorPolicy(t *testing.T) {
	t.Parallel()

	flowData := FlowData{
		Nodes: []FlowNode{
			{ID: "condition-1", Data: []byte(`{"type":"logic:condition","config":{"expression":"true","errorPolicy":"continue"}}`)},
		},
	}

	err := ValidateFlowData(flowData)
	if err == nil {
		t.Fatal("expected validation to fail")
	}
	if !strings.Contains(err.Error(), "errorPolicy") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateFlowDataAllowsContinueOnErrorPolicyForActionNodes(t *testing.T) {
	t.Parallel()

	flowData := FlowData{
		Nodes: []FlowNode{
			{ID: "action-1", Data: []byte(`{"type":"action:http","config":{"url":"https://example.com","errorPolicy":"continue"}}`)},
		},
	}

	if err := ValidateFlowData(flowData); err != nil {
		t.Fatalf("expected validation to succeed, got %v", err)
	}
}

func TestValidateFlowDataRejectsAggregateDuplicateResolvedIDs(t *testing.T) {
	t.Parallel()

	flowData := FlowData{
		Nodes: []FlowNode{
			{ID: "left", Data: []byte(`{"type":"action:http"}`)},
			{ID: "right", Data: []byte(`{"type":"action:http"}`)},
			{ID: "aggregate", Data: []byte(`{"type":"logic:aggregate","config":{"idOverrides":{"left":"shared","right":"shared"}}}`)},
		},
		Edges: []FlowEdge{
			{ID: "edge-1", Source: "left", Target: "aggregate"},
			{ID: "edge-2", Source: "right", Target: "aggregate"},
		},
	}

	err := ValidateFlowData(flowData)
	if err == nil {
		t.Fatal("expected validation to fail")
	}
	if !strings.Contains(err.Error(), "aggregate output id") {
		t.Fatalf("unexpected error: %v", err)
	}
}
