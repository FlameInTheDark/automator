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

type CustomProvider struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
	authHeader string
}

func NewCustomProvider(cfg Config) (*CustomProvider, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("base URL is required for custom provider")
	}

	authHeader := "Authorization"
	if cfg.ExtraConfig != nil {
		if h, ok := cfg.ExtraConfig["auth_header"].(string); ok {
			authHeader = h
		}
	}

	return &CustomProvider{
		apiKey:     cfg.APIKey,
		baseURL:    cfg.BaseURL,
		model:      cfg.Model,
		authHeader: authHeader,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}, nil
}

func (p *CustomProvider) Name() string {
	return "Custom"
}

func (p *CustomProvider) Type() ProviderType {
	return ProviderCustom
}

func (p *CustomProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	apiReq := openAIRequest{
		Model:       req.Model,
		Messages:    req.Messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
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

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if p.apiKey != "" {
		httpReq.Header.Set(p.authHeader, "Bearer "+p.apiKey)
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

	var apiResp openAIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := apiResp.Choices[0]
	return &ChatResponse{
		Content:   choice.Message.Content,
		ToolCalls: choice.Message.ToolCalls,
		Usage:     apiResp.Usage,
	}, nil
}
