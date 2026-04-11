package lua

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

func TestLuaNodeExecutePreloadsModules(t *testing.T) {
	executor := &LuaNode{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html><head><title>Emerald</title></head><body>ok</body></html>"))
	}))
	defer server.Close()

	script := `
local strings = require("strings")
local template = require("template")
local url = require("url")
local re = require("re")
local http = require("http")
local scrape = require("scrape")

local rendered, tmplErr = template.dostring("Hello, {{.name}}!", { name = input.name })
if tmplErr ~= nil then
  error(tmplErr)
end

local parsed = url.parse(input.url)
local response, requestErr = http.get(input.url)
if requestErr ~= nil then
  error(requestErr)
end

local titles, scrapeErr = scrape.find_text_by_tag(response.body, "title")
if scrapeErr ~= nil then
  error(scrapeErr)
end

return {
  upper = strings.ToUpper(input.name),
  rendered = rendered,
  host = parsed.host,
  digits = re.match("order-42", "([0-9]+)"),
  title = titles[1],
  has_batch = http.request_batch ~= nil,
}
`

	result, err := executor.Execute(
		context.Background(),
		json.RawMessage(fmt.Sprintf(`{"script":%q}`, script)),
		map[string]any{
			"name": "emerald",
			"url":  server.URL,
		},
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(result.Output, &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got := output["upper"]; got != "EMERALD" {
		t.Fatalf("upper = %v, want EMERALD", got)
	}

	if got := output["rendered"]; got != "Hello, emerald!" {
		t.Fatalf("rendered = %v, want Hello, emerald!", got)
	}

	if got := output["host"]; got == nil || got == "" {
		t.Fatalf("host = %v, want non-empty host", got)
	}

	if got := output["digits"]; got != "42" {
		t.Fatalf("digits = %v, want 42", got)
	}

	if got := output["title"]; got != "Emerald" {
		t.Fatalf("title = %v, want Emerald", got)
	}

	if got := output["has_batch"]; got != false {
		t.Fatalf("has_batch = %v, want false", got)
	}
}

func TestLuaNodeExecutePropagatesContextToHTTPModule(t *testing.T) {
	executor := &LuaNode{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(250 * time.Millisecond):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	script := `
local http = require("http")
local response, requestErr = http.get(input.url)
assert(response ~= nil, requestErr)
return { status = response.status_code }
`

	_, err := executor.Execute(
		ctx,
		json.RawMessage(fmt.Sprintf(`{"script":%q}`, script)),
		map[string]any{
			"url": server.URL,
		},
	)
	if err == nil {
		t.Fatal("Execute() error = nil, want context cancellation error")
	}

	if !strings.Contains(err.Error(), "context deadline exceeded") && !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("Execute() error = %v, want context deadline exceeded or context canceled", err)
	}
}
