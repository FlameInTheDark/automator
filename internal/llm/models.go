package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

type ModelInfo struct {
	ID            string `json:"id"`
	Name          string `json:"name,omitempty"`
	Description   string `json:"description,omitempty"`
	ContextLength int    `json:"context_length,omitempty"`
}

type openAIModelListResponse struct {
	Data []struct {
		ID          string `json:"id"`
		Name        string `json:"name,omitempty"`
		Description string `json:"description,omitempty"`
	} `json:"data"`
}

type openRouterModelListResponse struct {
	Data []struct {
		ID            string `json:"id"`
		Name          string `json:"name,omitempty"`
		Description   string `json:"description,omitempty"`
		ContextLength int    `json:"context_length,omitempty"`
	} `json:"data"`
}

type ollamaTagListResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

func ListModels(ctx context.Context, cfg Config) ([]ModelInfo, error) {
	switch cfg.ProviderType {
	case ProviderOpenAI:
		return listOpenAICompatibleModels(ctx, cfg, false)
	case ProviderOpenRouter:
		return listOpenAICompatibleModels(ctx, cfg, true)
	case ProviderOllama:
		return listOllamaModels(ctx, cfg)
	default:
		return nil, fmt.Errorf("model discovery is not supported for provider type %s", cfg.ProviderType)
	}
}

func listOpenAICompatibleModels(ctx context.Context, cfg Config, isOpenRouter bool) ([]ModelInfo, error) {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL(cfg.ProviderType)
	}
	if baseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("create models request: %w", err)
	}

	if cfg.APIKey != "" {
		request.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("load models: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read models response: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("models API error (status %d): %s", response.StatusCode, string(body))
	}

	var models []ModelInfo
	if isOpenRouter {
		var payload openRouterModelListResponse
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("parse models response: %w", err)
		}
		models = make([]ModelInfo, 0, len(payload.Data))
		for _, item := range payload.Data {
			models = append(models, ModelInfo{
				ID:            item.ID,
				Name:          item.Name,
				Description:   item.Description,
				ContextLength: item.ContextLength,
			})
		}
	} else {
		var payload openAIModelListResponse
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("parse models response: %w", err)
		}
		models = make([]ModelInfo, 0, len(payload.Data))
		for _, item := range payload.Data {
			models = append(models, ModelInfo{
				ID:          item.ID,
				Name:        item.Name,
				Description: item.Description,
			})
		}
	}

	sort.Slice(models, func(i, j int) bool {
		return strings.ToLower(models[i].ID) < strings.ToLower(models[j].ID)
	})

	return models, nil
}

func listOllamaModels(ctx context.Context, cfg Config) ([]ModelInfo, error) {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL(ProviderOllama)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("create tags request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("load models: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read models response: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("models API error (status %d): %s", response.StatusCode, string(body))
	}

	var payload ollamaTagListResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse models response: %w", err)
	}

	models := make([]ModelInfo, 0, len(payload.Models))
	for _, item := range payload.Models {
		models = append(models, ModelInfo{
			ID:   item.Name,
			Name: item.Name,
		})
	}

	sort.Slice(models, func(i, j int) bool {
		return strings.ToLower(models[i].ID) < strings.ToLower(models[j].ID)
	})

	return models, nil
}
