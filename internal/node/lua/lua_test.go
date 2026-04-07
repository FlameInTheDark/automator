package lua

import (
	"context"
	"encoding/json"
	"testing"
)

func TestLuaNodeExecutePreservesPrimitiveTypes(t *testing.T) {
	executor := &LuaNode{}

	result, err := executor.Execute(
		context.Background(),
		json.RawMessage(`{"script":"return { status = \"ok\", data = input.status_code }"}`),
		map[string]any{
			"status_code": 200,
		},
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(result.Output, &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got := output["status"]; got != "ok" {
		t.Fatalf("status = %v, want ok", got)
	}

	if got := output["data"]; got != float64(200) {
		t.Fatalf("data = %#v, want 200", got)
	}
}

func TestLuaNodeExecutePreservesNestedInput(t *testing.T) {
	executor := &LuaNode{}

	result, err := executor.Execute(
		context.Background(),
		json.RawMessage(`{"script":"return { status = \"ok\", data = input }"}`),
		map[string]any{
			"method":      "GET",
			"response":    map[string]any{"status": "ok"},
			"status_code": 200,
			"url":         "http://127.0.0.1:8080/api/v1/health",
		},
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(result.Output, &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	data, ok := output["data"].(map[string]any)
	if !ok {
		t.Fatalf("data type = %T, want map[string]any", output["data"])
	}

	if got := data["status_code"]; got != float64(200) {
		t.Fatalf("status_code = %#v, want 200", got)
	}

	response, ok := data["response"].(map[string]any)
	if !ok {
		t.Fatalf("response type = %T, want map[string]any", data["response"])
	}

	if got := response["status"]; got != "ok" {
		t.Fatalf("response.status = %v, want ok", got)
	}
}

func TestLuaNodeExecutePreservesArrays(t *testing.T) {
	executor := &LuaNode{}

	result, err := executor.Execute(
		context.Background(),
		json.RawMessage(`{"script":"return { items = input.nodes }"}`),
		map[string]any{
			"nodes": []any{
				map[string]any{"name": "pve01", "status": "online"},
				map[string]any{"name": "pve02", "status": "offline"},
			},
		},
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(result.Output, &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	items, ok := output["items"].([]any)
	if !ok {
		t.Fatalf("items type = %T, want []any", output["items"])
	}

	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}

	first, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("items[0] type = %T, want map[string]any", items[0])
	}

	if got := first["status"]; got != "online" {
		t.Fatalf("items[0].status = %v, want online", got)
	}
}
