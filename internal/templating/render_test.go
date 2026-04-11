package templating

import (
	"strings"
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

func TestRenderStringSupportsNodeSelectorsAlongsideInputAndSecrets(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"status": "ok",
		"secret": map[string]any{
			"api_key": "top-secret",
		},
		nodeSelectorContextKey: map[string]any{
			"action:http-1775583878229": map[string]any{
				"response": map[string]any{
					"status_code": 204,
				},
				"items": []any{
					map[string]any{"name": "alpha"},
				},
			},
		},
	}

	rendered, err := RenderString(
		`Status {{status}} / Secret {{secret.api_key}} / Code {{$('action:http-1775583878229').response.status_code}} / First {{$("action:http-1775583878229").items[0].name}}`,
		input,
	)
	if err != nil {
		t.Fatalf("RenderString returned error: %v", err)
	}

	if rendered != "Status ok / Secret top-secret / Code 204 / First alpha" {
		t.Fatalf("unexpected rendered string: %q", rendered)
	}
}

func TestRenderStringFailsForMissingNodeSelector(t *testing.T) {
	t.Parallel()

	_, err := RenderString(`{{$('missing-node').response.status_code}}`, map[string]any{})
	if err == nil {
		t.Fatal("expected missing node selector to fail")
	}
	if !strings.Contains(err.Error(), `node "missing-node" has not executed in this run`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRenderStringFailsForMissingNodeSelectorPath(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		nodeSelectorContextKey: map[string]any{
			"node-1": map[string]any{
				"response": map[string]any{
					"status_code": 200,
				},
			},
		},
	}

	_, err := RenderString(`{{$('node-1').response.body}}`, input)
	if err == nil {
		t.Fatal("expected missing node selector path to fail")
	}
	if !strings.Contains(err.Error(), `key "body" not found`) {
		t.Fatalf("unexpected error: %v", err)
	}
}
