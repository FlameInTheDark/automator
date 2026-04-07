package templating

import (
	"testing"
)

func TestRenderStringSupportsNestedPaths(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"nodes": []any{
			map[string]any{
				"status": "online",
				"node":   "pve1",
			},
		},
	}

	rendered, err := RenderString("Node {{input.nodes[0].node}} is {{input.nodes[0].status}}", input)
	if err != nil {
		t.Fatalf("RenderString returned error: %v", err)
	}

	if rendered != "Node pve1 is online" {
		t.Fatalf("unexpected rendered string: %q", rendered)
	}
}

func TestRenderStringSerializesObjectsAndArrays(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"nodes": []any{
			map[string]any{"status": "online"},
		},
	}

	rendered, err := RenderString("{{input.nodes}}", input)
	if err != nil {
		t.Fatalf("RenderString returned error: %v", err)
	}

	if rendered != `[{"status":"online"}]` {
		t.Fatalf("unexpected rendered string: %q", rendered)
	}
}

func TestRenderStringsWalksStructsAndMaps(t *testing.T) {
	t.Parallel()

	type sampleConfig struct {
		URL     string
		Headers map[string]string
	}

	cfg := sampleConfig{
		URL: "https://example.com/{{input.node}}",
		Headers: map[string]string{
			"X-Node": "{{node}}",
		},
	}

	input := map[string]any{
		"node": "pve1",
	}

	if err := RenderStrings(&cfg, input); err != nil {
		t.Fatalf("RenderStrings returned error: %v", err)
	}

	if cfg.URL != "https://example.com/pve1" {
		t.Fatalf("unexpected rendered URL: %q", cfg.URL)
	}

	if cfg.Headers["X-Node"] != "pve1" {
		t.Fatalf("unexpected rendered header: %q", cfg.Headers["X-Node"])
	}
}
