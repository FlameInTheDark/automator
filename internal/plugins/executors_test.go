package plugins

import (
	"context"
	"encoding/json"
	"testing"
)

func TestTriggerExecutorFlattensPayloadWithoutOverwritingReservedFields(t *testing.T) {
	t.Parallel()

	executor := &TriggerExecutor{
		Manager:  &Manager{},
		NodeType: "trigger:plugin/acme/pulse",
	}

	result, err := executor.Execute(context.Background(), nil, PluginTriggerExecutionContext("subscription-1", map[string]any{
		"message":         "hello",
		"subscription_id": "payload-value",
		"count":           2,
	}))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(result.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}

	if got := output["triggered_by"]; got != "plugin" {
		t.Fatalf("triggered_by = %#v, want plugin", got)
	}
	if got := output["subscription_id"]; got != "subscription-1" {
		t.Fatalf("subscription_id = %#v, want subscription-1", got)
	}
	if got := output["message"]; got != "hello" {
		t.Fatalf("message = %#v, want hello", got)
	}
	if got := output["count"]; got != float64(2) {
		t.Fatalf("count = %#v, want 2", got)
	}

	payload, ok := output["payload"].(map[string]any)
	if !ok {
		t.Fatalf("payload type = %T, want map[string]any", output["payload"])
	}
	if got := payload["subscription_id"]; got != "payload-value" {
		t.Fatalf("payload.subscription_id = %#v, want payload-value", got)
	}
}
