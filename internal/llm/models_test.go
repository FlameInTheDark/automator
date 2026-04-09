package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListModelsOpenRouter(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		fmt.Fprint(w, `{"data":[{"id":"openai/gpt-4o-mini","name":"GPT-4o mini","context_length":128000},{"id":"anthropic/claude-3.7-sonnet","name":"Claude 3.7 Sonnet","context_length":200000}]}`)
	}))
	defer server.Close()

	models, err := ListModels(context.Background(), Config{
		ProviderType: ProviderOpenRouter,
		APIKey:       "secret",
		BaseURL:      server.URL,
	})
	if err != nil {
		t.Fatalf("ListModels returned error: %v", err)
	}

	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}

	if models[0].ID != "anthropic/claude-3.7-sonnet" || models[1].ID != "openai/gpt-4o-mini" {
		t.Fatalf("unexpected model order: %#v", models)
	}
}

func TestListModelsOllama(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"models":[{"name":"llama3.2"},{"name":"qwen2.5"}]}`)
	}))
	defer server.Close()

	models, err := ListModels(context.Background(), Config{
		ProviderType: ProviderOllama,
		BaseURL:      server.URL,
	})
	if err != nil {
		t.Fatalf("ListModels returned error: %v", err)
	}

	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}

	if models[0].ID != "llama3.2" || models[1].ID != "qwen2.5" {
		t.Fatalf("unexpected models: %#v", models)
	}
}

func TestListModelsCustomProvider(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		fmt.Fprint(w, `{"data":[{"id":"custom/alpha","name":"Alpha"},{"id":"custom/beta","name":"Beta"}]}`)
	}))
	defer server.Close()

	models, err := ListModels(context.Background(), Config{
		ProviderType: ProviderCustom,
		APIKey:       "secret",
		BaseURL:      server.URL,
	})
	if err != nil {
		t.Fatalf("ListModels returned error: %v", err)
	}

	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}

	if models[0].ID != "custom/alpha" || models[1].ID != "custom/beta" {
		t.Fatalf("unexpected models: %#v", models)
	}
}
