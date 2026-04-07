package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type OllamaProvider struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

type ollamaRequest struct {
	Model    string         `json:"model"`
	Messages []Message      `json:"messages"`
	Stream   bool           `json:"stream"`
	Tools    []any          `json:"tools,omitempty"`
	Options  map[string]any `json:"options,omitempty"`
}

type ollamaResponse struct {
	Message struct {
		Role      string     `json:"role"`
		Content   string     `json:"content"`
		ToolCalls []ToolCall `json:"tool_calls"`
	} `json:"message"`
	Done bool `json:"done"`
}

func NewOllamaProvider(cfg Config) (*OllamaProvider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL(ProviderOllama)
	}

	return &OllamaProvider{
		baseURL: baseURL,
		model:   cfg.Model,
		httpClient: &http.Client{
			Timeout: 300 * time.Second,
		},
	}, nil
}

func (p *OllamaProvider) Name() string {
	return "Ollama"
}

func (p *OllamaProvider) Type() ProviderType {
	return ProviderOllama
}

func (p *OllamaProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	apiReq := ollamaRequest{
		Model:    req.Model,
		Messages: req.Messages,
		Stream:   false,
	}

	if len(req.Tools) > 0 {
		tools := make([]any, len(req.Tools))
		for i, t := range req.Tools {
			tools[i] = t
		}
		apiReq.Tools = tools
	}

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var apiResp ollamaResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &ChatResponse{
		Content:   apiResp.Message.Content,
		ToolCalls: apiResp.Message.ToolCalls,
	}, nil
}
