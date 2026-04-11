package logic

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/FlameInTheDark/emerald/internal/node"
)

func decodeTransformResult(t *testing.T, result *node.NodeResult) map[string]any {
	t.Helper()

	var payload map[string]any
	if err := json.Unmarshal(result.Output, &payload); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}

	return payload
}

func executeTransformNode(t *testing.T, executor node.NodeExecutor, config string, input map[string]any) map[string]any {
	t.Helper()

	result, err := executor.Execute(context.Background(), json.RawMessage(config), input)
	if err != nil {
		t.Fatalf("execute node: %v", err)
	}

	return decodeTransformResult(t, result)
}

func TestSortNode(t *testing.T) {
	t.Run("sorts array into output path and preserves payload", func(t *testing.T) {
		output := executeTransformNode(t, &SortNode{}, `{
			"inputPath":"items",
			"outputPath":"results.sorted",
			"fieldPath":"score",
			"direction":"asc",
			"valueType":"number"
		}`, map[string]any{
			"meta": map[string]any{"source": "http"},
			"items": []any{
				map[string]any{"name": "three", "score": 3},
				map[string]any{"name": "one", "score": 1},
				map[string]any{"name": "two", "score": 2},
			},
		})

		if output["meta"].(map[string]any)["source"] != "http" {
			t.Fatalf("expected meta to be preserved, got %#v", output["meta"])
		}

		originalItems := output["items"].([]any)
		if originalItems[0].(map[string]any)["name"] != "three" {
			t.Fatalf("expected original items to remain unchanged, got %#v", originalItems)
		}

		results := output["results"].(map[string]any)["sorted"].([]any)
		got := []string{
			results[0].(map[string]any)["name"].(string),
			results[1].(map[string]any)["name"].(string),
			results[2].(map[string]any)["name"].(string),
		}
		want := []string{"one", "two", "three"}
		for index := range want {
			if got[index] != want[index] {
				t.Fatalf("sorted[%d] = %q, want %q", index, got[index], want[index])
			}
		}
	})

	t.Run("rejects invalid config", func(t *testing.T) {
		err := (&SortNode{}).Validate(json.RawMessage(`{"inputPath":"items","direction":"sideways"}`))
		if err == nil || !strings.Contains(err.Error(), "direction must be either asc or desc") {
			t.Fatalf("expected invalid direction error, got %v", err)
		}
	})

	t.Run("rejects missing and non-array input paths", func(t *testing.T) {
		err := (&SortNode{}).Validate(json.RawMessage(`{"direction":"asc"}`))
		if err == nil || !strings.Contains(err.Error(), "inputPath is required") {
			t.Fatalf("expected missing inputPath error, got %v", err)
		}

		_, err = (&SortNode{}).Execute(context.Background(), json.RawMessage(`{"inputPath":"items"}`), map[string]any{
			"items": map[string]any{"name": "not-an-array"},
		})
		if err == nil || !strings.Contains(err.Error(), "must resolve to an array") {
			t.Fatalf("expected non-array inputPath error, got %v", err)
		}
	})
}

func TestLimitNode(t *testing.T) {
	t.Run("limits array and preserves sibling fields", func(t *testing.T) {
		output := executeTransformNode(t, &LimitNode{}, `{
			"inputPath":"items",
			"outputPath":"items",
			"maxItems":2
		}`, map[string]any{
			"meta":  map[string]any{"source": "http"},
			"items": []any{"one", "two", "three"},
		})

		if output["meta"].(map[string]any)["source"] != "http" {
			t.Fatalf("expected meta to be preserved, got %#v", output["meta"])
		}

		items := output["items"].([]any)
		if len(items) != 2 || items[0] != "one" || items[1] != "two" {
			t.Fatalf("unexpected limited items: %#v", items)
		}
	})

	t.Run("rejects invalid config", func(t *testing.T) {
		err := (&LimitNode{}).Validate(json.RawMessage(`{"inputPath":"items","maxItems":-1}`))
		if err == nil || !strings.Contains(err.Error(), "maxItems must be 0 or greater") {
			t.Fatalf("expected invalid maxItems error, got %v", err)
		}
	})

	t.Run("rejects missing and non-array input paths", func(t *testing.T) {
		err := (&LimitNode{}).Validate(json.RawMessage(`{"maxItems":2}`))
		if err == nil || !strings.Contains(err.Error(), "inputPath is required") {
			t.Fatalf("expected missing inputPath error, got %v", err)
		}

		_, err = (&LimitNode{}).Execute(context.Background(), json.RawMessage(`{"inputPath":"items","maxItems":2}`), map[string]any{
			"items": "not-an-array",
		})
		if err == nil || !strings.Contains(err.Error(), "must resolve to an array") {
			t.Fatalf("expected non-array inputPath error, got %v", err)
		}
	})
}

func TestRemoveDuplicatesNode(t *testing.T) {
	t.Run("removes duplicates by field and keeps last occurrence", func(t *testing.T) {
		output := executeTransformNode(t, &RemoveDuplicatesNode{}, `{
			"inputPath":"items",
			"outputPath":"items",
			"strategy":"field",
			"fieldPath":"id",
			"keep":"last"
		}`, map[string]any{
			"meta": map[string]any{"source": "http"},
			"items": []any{
				map[string]any{"id": "alpha", "name": "first"},
				map[string]any{"id": "beta", "name": "middle"},
				map[string]any{"id": "alpha", "name": "last"},
			},
		})

		if output["meta"].(map[string]any)["source"] != "http" {
			t.Fatalf("expected meta to be preserved, got %#v", output["meta"])
		}

		items := output["items"].([]any)
		if len(items) != 2 {
			t.Fatalf("expected 2 items after dedupe, got %#v", items)
		}
		if items[0].(map[string]any)["id"] != "beta" || items[1].(map[string]any)["name"] != "last" {
			t.Fatalf("unexpected dedupe result: %#v", items)
		}
	})

	t.Run("rejects invalid config", func(t *testing.T) {
		err := (&RemoveDuplicatesNode{}).Validate(json.RawMessage(`{"inputPath":"items","strategy":"field","keep":"sometimes"}`))
		if err == nil || !strings.Contains(err.Error(), "fieldPath is required when strategy is field") {
			t.Fatalf("expected missing fieldPath validation error, got %v", err)
		}

		err = (&RemoveDuplicatesNode{}).Validate(json.RawMessage(`{"inputPath":"items","strategy":"whole_item","keep":"sometimes"}`))
		if err == nil || !strings.Contains(err.Error(), "keep must be first or last") {
			t.Fatalf("expected invalid keep validation error, got %v", err)
		}
	})

	t.Run("rejects missing and non-array input paths", func(t *testing.T) {
		err := (&RemoveDuplicatesNode{}).Validate(json.RawMessage(`{"strategy":"whole_item"}`))
		if err == nil || !strings.Contains(err.Error(), "inputPath is required") {
			t.Fatalf("expected missing inputPath error, got %v", err)
		}

		_, err = (&RemoveDuplicatesNode{}).Execute(context.Background(), json.RawMessage(`{"inputPath":"items","strategy":"whole_item"}`), map[string]any{
			"items": 42,
		})
		if err == nil || !strings.Contains(err.Error(), "must resolve to an array") {
			t.Fatalf("expected non-array inputPath error, got %v", err)
		}
	})
}

func TestSummarizeNode(t *testing.T) {
	t.Run("builds grouped summaries and preserves payload", func(t *testing.T) {
		output := executeTransformNode(t, &SummarizeNode{}, `{
			"inputPath":"items",
			"outputPath":"summary",
			"groupByPath":"region",
			"metrics":[
				{"name":"total","op":"count"},
				{"name":"amount","op":"sum","fieldPath":"amount"},
				{"name":"average","op":"avg","fieldPath":"amount"},
				{"name":"peak","op":"max","fieldPath":"amount"}
			]
		}`, map[string]any{
			"meta": map[string]any{"requestId": "req-1"},
			"items": []any{
				map[string]any{"region": "us", "amount": 3},
				map[string]any{"region": "eu", "amount": 5},
				map[string]any{"region": "us", "amount": 7},
			},
		})

		if output["meta"].(map[string]any)["requestId"] != "req-1" {
			t.Fatalf("expected meta to be preserved, got %#v", output["meta"])
		}

		summary := output["summary"].(map[string]any)
		if summary["count"] != float64(3) {
			t.Fatalf("summary count = %#v, want 3", summary["count"])
		}

		byGroup := summary["byGroup"].(map[string]any)
		us := byGroup["us"].(map[string]any)
		usMetrics := us["metrics"].(map[string]any)
		if us["count"] != float64(2) {
			t.Fatalf("us count = %#v, want 2", us["count"])
		}
		if usMetrics["total"] != float64(2) || usMetrics["amount"] != float64(10) || usMetrics["average"] != float64(5) || usMetrics["peak"] != float64(7) {
			t.Fatalf("unexpected us metrics: %#v", usMetrics)
		}

		items := output["items"].([]any)
		if len(items) != 3 {
			t.Fatalf("expected original items to remain available, got %#v", items)
		}
	})

	t.Run("rejects invalid config", func(t *testing.T) {
		err := (&SummarizeNode{}).Validate(json.RawMessage(`{
			"inputPath":"items",
			"metrics":[{"name":"bad","op":"median","fieldPath":"amount"}]
		}`))
		if err == nil || !strings.Contains(err.Error(), "must be count, sum, avg, min, or max") {
			t.Fatalf("expected invalid metric op error, got %v", err)
		}
	})

	t.Run("rejects missing and non-array input paths", func(t *testing.T) {
		err := (&SummarizeNode{}).Validate(json.RawMessage(`{"metrics":[{"name":"total","op":"count"}]}`))
		if err == nil || !strings.Contains(err.Error(), "inputPath is required") {
			t.Fatalf("expected missing inputPath error, got %v", err)
		}

		_, err = (&SummarizeNode{}).Execute(context.Background(), json.RawMessage(`{
			"inputPath":"items",
			"metrics":[{"name":"total","op":"count"}]
		}`), map[string]any{
			"items": map[string]any{"amount": 1},
		})
		if err == nil || !strings.Contains(err.Error(), "must resolve to an array") {
			t.Fatalf("expected non-array inputPath error, got %v", err)
		}
	})
}
