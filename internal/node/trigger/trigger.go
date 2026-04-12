package trigger

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/FlameInTheDark/emerald/internal/node"
)

type channelEventContextKey string

const channelEventKey channelEventContextKey = "channel_event"

type ChannelEvent struct {
	ChannelID      string         `json:"channel_id"`
	ChannelName    string         `json:"channel_name"`
	ChannelType    string         `json:"channel_type"`
	ContactID      string         `json:"contact_id"`
	ExternalUserID string         `json:"external_user_id"`
	ExternalChatID string         `json:"external_chat_id"`
	Username       string         `json:"username,omitempty"`
	DisplayName    string         `json:"display_name,omitempty"`
	Text           string         `json:"text"`
	Message        map[string]any `json:"message,omitempty"`
}

func WithChannelEvent(ctx context.Context, event ChannelEvent) context.Context {
	return context.WithValue(ctx, channelEventKey, event)
}

func ChannelEventFromContext(ctx context.Context) (ChannelEvent, bool) {
	if ctx == nil {
		return ChannelEvent{}, false
	}

	event, ok := ctx.Value(channelEventKey).(ChannelEvent)
	return event, ok
}

func IsTriggerType(nodeType node.NodeType) bool {
	return strings.HasPrefix(strings.TrimSpace(string(nodeType)), "trigger:")
}

func MatchesExecution(ctx context.Context, nodeType node.NodeType, config json.RawMessage, triggerType string) bool {
	switch triggerType {
	case "manual":
		return nodeType == node.TypeTriggerManual
	case "cron":
		return nodeType == node.TypeTriggerCron
	case "webhook":
		return nodeType == node.TypeTriggerWebhook
	case "channel":
		if nodeType != node.TypeTriggerChannel {
			return false
		}

		event, ok := ChannelEventFromContext(ctx)
		if !ok {
			return false
		}

		var cfg channelTriggerConfig
		if err := json.Unmarshal(config, &cfg); err != nil {
			return false
		}

		if strings.TrimSpace(cfg.ChannelID) == "" {
			return true
		}

		return strings.TrimSpace(cfg.ChannelID) == event.ChannelID
	default:
		return false
	}
}

type ManualTrigger struct{}

func (e *ManualTrigger) Execute(ctx context.Context, config json.RawMessage, input map[string]any) (*node.NodeResult, error) {
	output := map[string]any{
		"triggered_by": "manual",
		"input":        input,
	}
	data, _ := json.Marshal(output)
	return &node.NodeResult{Output: data}, nil
}

func (e *ManualTrigger) Validate(config json.RawMessage) error {
	return nil
}

type CronTrigger struct{}

type cronConfig struct {
	Schedule string `json:"schedule"`
	Timezone string `json:"timezone"`
}

func (e *CronTrigger) Execute(ctx context.Context, config json.RawMessage, input map[string]any) (*node.NodeResult, error) {
	var cfg cronConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.Timezone == "" {
		cfg.Timezone = "UTC"
	}

	output := map[string]any{
		"triggered_by": "cron",
		"schedule":     cfg.Schedule,
		"timezone":     cfg.Timezone,
		"input":        input,
	}
	data, _ := json.Marshal(output)
	return &node.NodeResult{Output: data}, nil
}

func (e *CronTrigger) Validate(config json.RawMessage) error {
	var cfg cronConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	if cfg.Schedule == "" {
		return fmt.Errorf("schedule is required")
	}
	return nil
}

type WebhookTrigger struct{}

type WebhookConfig struct {
	Path   string `json:"path"`
	Method string `json:"method"`
	Token  string `json:"token"`
}

func DecodeWebhookConfig(config json.RawMessage) (WebhookConfig, error) {
	cfg := WebhookConfig{Method: "POST"}
	if len(config) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(config, &cfg); err != nil {
		return WebhookConfig{}, fmt.Errorf("invalid config: %w", err)
	}

	return NormalizeWebhookConfig(cfg)
}

func NormalizeWebhookConfig(cfg WebhookConfig) (WebhookConfig, error) {
	cfg.Path = NormalizeWebhookPath(cfg.Path)
	cfg.Method = NormalizeWebhookMethod(cfg.Method)
	cfg.Token = strings.TrimSpace(cfg.Token)

	if cfg.Path == "" || cfg.Path == "/webhook" {
		return WebhookConfig{}, fmt.Errorf("path is required")
	}

	switch cfg.Method {
	case "DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT":
		return cfg, nil
	default:
		return WebhookConfig{}, fmt.Errorf("method %q is not supported", cfg.Method)
	}
}

func NormalizeWebhookPath(rawPath string) string {
	trimmed := strings.TrimSpace(rawPath)
	if trimmed == "" {
		return ""
	}

	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}

	cleaned := path.Clean(trimmed)
	if cleaned == "." || cleaned == "/" {
		return ""
	}

	if strings.HasPrefix(cleaned, "/webhook/") || cleaned == "/webhook" {
		return cleaned
	}

	return "/webhook/" + strings.Trim(strings.TrimPrefix(cleaned, "/"), "/")
}

func NormalizeWebhookMethod(method string) string {
	normalized := strings.ToUpper(strings.TrimSpace(method))
	if normalized == "" {
		return "POST"
	}
	return normalized
}

func (e *WebhookTrigger) Execute(ctx context.Context, config json.RawMessage, input map[string]any) (*node.NodeResult, error) {
	cfg, err := DecodeWebhookConfig(config)
	if err != nil {
		return nil, err
	}

	output := map[string]any{
		"triggered_by": "webhook",
		"path":         cfg.Path,
		"method":       cfg.Method,
		"input":        input,
	}
	copyWebhookField(output, input, "body")
	copyWebhookField(output, input, "raw_body")
	copyWebhookField(output, input, "headers")
	copyWebhookField(output, input, "query")
	copyWebhookField(output, input, "content_type")
	copyWebhookField(output, input, "remote_ip")
	copyWebhookField(output, input, "user_agent")
	data, _ := json.Marshal(output)
	return &node.NodeResult{Output: data}, nil
}

func (e *WebhookTrigger) Validate(config json.RawMessage) error {
	_, err := DecodeWebhookConfig(config)
	return err
}

type ChannelMessageTrigger struct{}

type channelTriggerConfig struct {
	ChannelID string `json:"channelId"`
}

func (e *ChannelMessageTrigger) Execute(ctx context.Context, config json.RawMessage, input map[string]any) (*node.NodeResult, error) {
	var cfg channelTriggerConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	event, ok := ChannelEventFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("channel trigger event is missing")
	}

	if cfg.ChannelID != "" && cfg.ChannelID != event.ChannelID {
		return nil, fmt.Errorf("channel trigger does not match event channel")
	}

	output := map[string]any{
		"triggered_by":     "channel",
		"channel_id":       event.ChannelID,
		"channel_name":     event.ChannelName,
		"channel_type":     event.ChannelType,
		"contact_id":       event.ContactID,
		"external_user_id": event.ExternalUserID,
		"external_chat_id": event.ExternalChatID,
		"chat_id":          event.ExternalChatID,
		"user_id":          event.ExternalUserID,
		"username":         event.Username,
		"display_name":     event.DisplayName,
		"text":             event.Text,
		"message":          event.Message,
		"input":            input,
	}
	data, _ := json.Marshal(output)
	return &node.NodeResult{Output: data}, nil
}

func (e *ChannelMessageTrigger) Validate(config json.RawMessage) error {
	var cfg channelTriggerConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	return nil
}

func copyWebhookField(target map[string]any, input map[string]any, key string) {
	if target == nil || len(input) == 0 {
		return
	}

	if value, ok := input[key]; ok {
		target[key] = value
	}
}
