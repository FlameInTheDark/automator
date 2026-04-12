package triggers

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/FlameInTheDark/emerald/internal/db/models"
	triggernode "github.com/FlameInTheDark/emerald/internal/node/trigger"
	"github.com/FlameInTheDark/emerald/internal/pipeline"
	"github.com/FlameInTheDark/emerald/internal/plugins"
	"github.com/FlameInTheDark/emerald/pkg/pluginapi"
)

var (
	ErrWebhookNotFound      = errors.New("webhook endpoint not found")
	ErrWebhookUnauthorized  = errors.New("webhook token is invalid")
	ErrTriggerRunnerMissing = errors.New("trigger execution runner is not configured")
)

type ActivePipelineStore interface {
	ListActive(ctx context.Context) ([]models.Pipeline, error)
}

type CronReloader interface {
	Reload(ctx context.Context) error
}

type PluginTriggerReloader interface {
	Reload(ctx context.Context, subscriptions []pluginapi.TriggerSubscription) error
	Stop()
}

type Service struct {
	store    ActivePipelineStore
	schedule CronReloader
	runner   *pipeline.ExecutionRunner
	plugins  PluginTriggerReloader

	webhooksMu sync.RWMutex
	webhooks   map[string]webhookEntry

	pluginDispatchMu sync.RWMutex
	pluginDispatch   map[string]pluginDispatchEntry
}

type WebhookRequest struct {
	Method      string
	Path        string
	Token       string
	ContentType string
	Headers     map[string][]string
	Query       map[string][]string
	Body        []byte
	RemoteIP    string
	UserAgent   string
}

type webhookEntry struct {
	Method     string
	Path       string
	Token      string
	PipelineID string
	NodeID     string
	FlowData   pipeline.FlowData
}

type pluginDispatchEntry struct {
	PipelineID string
	NodeID     string
	FlowData   pipeline.FlowData
}

type activeSnapshot struct {
	webhooks       map[string]webhookEntry
	pluginDispatch map[string]pluginDispatchEntry
	subscriptions  []pluginapi.TriggerSubscription
}

func NewService(store ActivePipelineStore, schedule CronReloader, runner *pipeline.ExecutionRunner, pluginTriggers PluginTriggerReloader) *Service {
	return &Service{
		store:          store,
		schedule:       schedule,
		runner:         runner,
		plugins:        pluginTriggers,
		webhooks:       make(map[string]webhookEntry),
		pluginDispatch: make(map[string]pluginDispatchEntry),
	}
}

func (s *Service) ValidatePipeline(ctx context.Context, pipelineModel *models.Pipeline) error {
	if s == nil || s.store == nil || pipelineModel == nil {
		return nil
	}
	if strings.TrimSpace(pipelineModel.Status) != "active" {
		return nil
	}

	activePipelines, err := s.store.ListActive(ctx)
	if err != nil {
		return fmt.Errorf("list active pipelines: %w", err)
	}

	filtered := make([]models.Pipeline, 0, len(activePipelines)+1)
	for _, activePipeline := range activePipelines {
		if strings.TrimSpace(activePipeline.ID) == strings.TrimSpace(pipelineModel.ID) && strings.TrimSpace(pipelineModel.ID) != "" {
			continue
		}
		filtered = append(filtered, activePipeline)
	}
	filtered = append(filtered, *pipelineModel)

	_, err = buildActiveSnapshot(filtered)
	return err
}

func (s *Service) Reload(ctx context.Context) error {
	if s == nil {
		return nil
	}

	var errs []error
	if s.schedule != nil {
		if err := s.schedule.Reload(ctx); err != nil {
			errs = append(errs, fmt.Errorf("reload cron triggers: %w", err))
		}
	}

	if s.store == nil {
		return errors.Join(errs...)
	}

	activePipelines, err := s.store.ListActive(ctx)
	if err != nil {
		errs = append(errs, fmt.Errorf("list active pipelines: %w", err))
		return errors.Join(errs...)
	}

	snapshot, err := buildActiveSnapshot(activePipelines)
	if err != nil {
		errs = append(errs, err)
		return errors.Join(errs...)
	}

	s.webhooksMu.Lock()
	s.webhooks = snapshot.webhooks
	s.webhooksMu.Unlock()

	s.pluginDispatchMu.Lock()
	s.pluginDispatch = snapshot.pluginDispatch
	s.pluginDispatchMu.Unlock()

	if s.plugins != nil {
		if err := s.plugins.Reload(ctx, snapshot.subscriptions); err != nil {
			errs = append(errs, fmt.Errorf("reload plugin triggers: %w", err))
		}
	}

	return errors.Join(errs...)
}

func (s *Service) Stop() {
	if s == nil || s.plugins == nil {
		return
	}
	s.plugins.Stop()
}

func (s *Service) DispatchWebhook(ctx context.Context, request WebhookRequest) (*pipeline.ExecutionRunResult, error) {
	if s == nil || s.runner == nil {
		return nil, ErrTriggerRunnerMissing
	}

	entry, ok := s.lookupWebhook(request.Method, request.Path)
	if !ok {
		return nil, ErrWebhookNotFound
	}
	if entry.Token != "" && subtle.ConstantTimeCompare([]byte(entry.Token), []byte(strings.TrimSpace(request.Token))) != 1 {
		return nil, ErrWebhookUnauthorized
	}

	executionContext := buildWebhookExecutionContext(request, entry)
	return s.runner.Run(
		ctx,
		entry.PipelineID,
		entry.FlowData,
		pipeline.TriggerSelectionFromNodeIDs("webhook", []string{entry.NodeID}),
		executionContext,
	)
}

func (s *Service) HandlePluginEvent(ctx context.Context, subscription pluginapi.TriggerSubscription, event *pluginapi.TriggerEvent) error {
	if s == nil || s.runner == nil || event == nil {
		return nil
	}

	subscriptionID := strings.TrimSpace(event.SubscriptionID)
	if subscriptionID == "" {
		return nil
	}

	s.pluginDispatchMu.RLock()
	entry, ok := s.pluginDispatch[subscriptionID]
	s.pluginDispatchMu.RUnlock()
	if !ok {
		return nil
	}

	selection := pipeline.TriggerSelectionFromNodeIDs("plugin", []string{entry.NodeID})
	executionContext := plugins.PluginTriggerExecutionContext(subscriptionID, event.Payload)

	_, err := s.runner.Run(ctx, entry.PipelineID, entry.FlowData, selection, executionContext)
	return err
}

func (s *Service) lookupWebhook(method string, path string) (webhookEntry, bool) {
	if s == nil {
		return webhookEntry{}, false
	}

	key := webhookKey(method, path)

	s.webhooksMu.RLock()
	defer s.webhooksMu.RUnlock()

	entry, ok := s.webhooks[key]
	return entry, ok
}

func buildActiveSnapshot(activePipelines []models.Pipeline) (*activeSnapshot, error) {
	webhooks := make(map[string]webhookEntry)
	pluginDispatch := make(map[string]pluginDispatchEntry)
	subscriptions := make([]pluginapi.TriggerSubscription, 0)

	for _, pipelineModel := range activePipelines {
		flowData, err := pipeline.ParseFlowData(pipelineModel.Nodes, pipelineModel.Edges)
		if err != nil {
			return nil, fmt.Errorf("parse pipeline %s: %w", pipelineModel.ID, err)
		}

		for _, flowNode := range flowData.Nodes {
			nodeType, config := resolveNodeTypeAndConfig(flowNode)

			switch {
			case nodeType == "trigger:webhook":
				cfg, err := triggernode.DecodeWebhookConfig(config)
				if err != nil {
					return nil, fmt.Errorf("pipeline %s node %s: %w", pipelineModel.ID, flowNode.ID, err)
				}

				key := webhookKey(cfg.Method, cfg.Path)
				if existing, exists := webhooks[key]; exists {
					return nil, fmt.Errorf(
						"duplicate active webhook endpoint %s claimed by pipeline %s node %s and pipeline %s node %s",
						key,
						existing.PipelineID,
						existing.NodeID,
						pipelineModel.ID,
						flowNode.ID,
					)
				}

				webhooks[key] = webhookEntry{
					Method:     cfg.Method,
					Path:       cfg.Path,
					Token:      cfg.Token,
					PipelineID: pipelineModel.ID,
					NodeID:     flowNode.ID,
					FlowData:   *flowData,
				}

			case strings.HasPrefix(nodeType, "trigger:plugin/"):
				kind, _, nodeID, ok := plugins.ParseNodeType(nodeType)
				if !ok || kind != pluginapi.NodeKindTrigger {
					return nil, fmt.Errorf("pipeline %s node %s uses invalid trigger plugin type %q", pipelineModel.ID, flowNode.ID, nodeType)
				}

				subscriptionID := strings.TrimSpace(pipelineModel.ID) + ":" + strings.TrimSpace(flowNode.ID)
				subscription := pluginapi.TriggerSubscription{
					SubscriptionID: subscriptionID,
					PipelineID:     strings.TrimSpace(pipelineModel.ID),
					NodeType:       strings.TrimSpace(nodeType),
					NodeID:         nodeID,
					NodeInstanceID: strings.TrimSpace(flowNode.ID),
					Config:         append(json.RawMessage(nil), config...),
				}

				subscriptions = append(subscriptions, subscription)
				pluginDispatch[subscriptionID] = pluginDispatchEntry{
					PipelineID: pipelineModel.ID,
					NodeID:     flowNode.ID,
					FlowData:   *flowData,
				}
			}
		}
	}

	return &activeSnapshot{
		webhooks:       webhooks,
		pluginDispatch: pluginDispatch,
		subscriptions:  subscriptions,
	}, nil
}

func webhookKey(method string, path string) string {
	normalizedMethod := triggernode.NormalizeWebhookMethod(method)
	normalizedPath := triggernode.NormalizeWebhookPath(path)
	return normalizedMethod + " " + normalizedPath
}

func resolveNodeTypeAndConfig(flowNode pipeline.FlowNode) (string, json.RawMessage) {
	nodeType := strings.TrimSpace(flowNode.Type)
	config := flowNode.Data

	if len(flowNode.Data) == 0 {
		return nodeType, config
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(flowNode.Data, &payload); err != nil {
		return nodeType, config
	}

	if rawType, ok := payload["type"]; ok {
		var decoded string
		if err := json.Unmarshal(rawType, &decoded); err == nil && strings.TrimSpace(decoded) != "" {
			nodeType = strings.TrimSpace(decoded)
		}
	}
	if rawConfig, ok := payload["config"]; ok {
		config = rawConfig
	}

	return nodeType, config
}

func buildWebhookExecutionContext(request WebhookRequest, entry webhookEntry) map[string]any {
	return map[string]any{
		"method":       entry.Method,
		"path":         entry.Path,
		"headers":      headerMapToAny(request.Headers),
		"query":        headerMapToAny(request.Query),
		"content_type": strings.TrimSpace(request.ContentType),
		"body":         decodeWebhookBody(request.Body, request.ContentType),
		"raw_body":     string(request.Body),
		"remote_ip":    strings.TrimSpace(request.RemoteIP),
		"user_agent":   strings.TrimSpace(request.UserAgent),
	}
}

func decodeWebhookBody(body []byte, contentType string) any {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return ""
	}

	if looksLikeJSONContent(contentType, trimmed) {
		var decoded any
		if err := json.Unmarshal([]byte(trimmed), &decoded); err == nil {
			return decoded
		}
	}

	return trimmed
}

func looksLikeJSONContent(contentType string, body string) bool {
	normalizedContentType := strings.ToLower(strings.TrimSpace(contentType))
	if strings.Contains(normalizedContentType, "application/json") || strings.Contains(normalizedContentType, "+json") {
		return true
	}

	return strings.HasPrefix(body, "{") || strings.HasPrefix(body, "[") || strings.HasPrefix(body, "\"")
}

func headerMapToAny(values map[string][]string) map[string]any {
	if len(values) == 0 {
		return map[string]any{}
	}

	result := make(map[string]any, len(values))
	for key, entries := range values {
		copied := append([]string(nil), entries...)
		if len(copied) == 1 {
			result[key] = copied[0]
			continue
		}
		result[key] = copied
	}

	return result
}

func ExtractWebhookToken(req *http.Request) string {
	if req == nil {
		return ""
	}

	if token := strings.TrimSpace(req.Header.Get("X-Emerald-Webhook-Token")); token != "" {
		return token
	}

	if authHeader := strings.TrimSpace(req.Header.Get("Authorization")); strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return strings.TrimSpace(authHeader[7:])
	}

	return strings.TrimSpace(req.URL.Query().Get("token"))
}
