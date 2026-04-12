package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/FlameInTheDark/emerald/internal/llm"
	"github.com/FlameInTheDark/emerald/internal/node"
	"github.com/FlameInTheDark/emerald/internal/templating"
	"github.com/FlameInTheDark/emerald/pkg/pluginapi"
)

const pluginTriggerEventKey = "_plugin_trigger"

type ActionExecutor struct {
	Manager  *Manager
	NodeType string
	Outputs  []pluginapi.OutputHandle
}

type TriggerExecutor struct {
	Manager  *Manager
	NodeType string
}

func (e *ActionExecutor) Execute(ctx context.Context, config json.RawMessage, input map[string]any) (*node.NodeResult, error) {
	if e == nil || e.Manager == nil {
		return nil, fmt.Errorf("plugin manager is not configured")
	}

	renderedConfig, err := templating.RenderJSONWithContext(ctx, config, input)
	if err != nil {
		return nil, fmt.Errorf("render config: %w", err)
	}

	output, err := e.Manager.ExecuteAction(ctx, e.NodeType, renderedConfig, input)
	if err != nil {
		return nil, err
	}

	if err := validateActionOutput(output, e.Outputs); err != nil {
		return nil, err
	}

	payload, err := json.Marshal(output)
	if err != nil {
		return nil, fmt.Errorf("encode plugin action output: %w", err)
	}

	return &node.NodeResult{Output: payload}, nil
}

func (e *ActionExecutor) Validate(config json.RawMessage) error {
	if e == nil || e.Manager == nil {
		return fmt.Errorf("plugin manager is not configured")
	}
	return e.Manager.ValidateConfig(context.Background(), e.NodeType, config)
}

func (e *TriggerExecutor) Execute(_ context.Context, _ json.RawMessage, input map[string]any) (*node.NodeResult, error) {
	if e == nil || e.Manager == nil {
		return nil, fmt.Errorf("plugin manager is not configured")
	}

	event, err := pluginTriggerEventFromInput(input)
	if err != nil {
		return nil, err
	}

	output := map[string]any{
		"triggered_by":    "plugin",
		"subscription_id": event.SubscriptionID,
		"payload":         event.Payload,
	}

	if payloadMap, ok := event.Payload.(map[string]any); ok {
		for key, value := range payloadMap {
			if _, exists := output[key]; exists {
				continue
			}
			output[key] = value
		}
	}

	payload, err := json.Marshal(output)
	if err != nil {
		return nil, fmt.Errorf("encode plugin trigger output: %w", err)
	}

	return &node.NodeResult{Output: payload}, nil
}

func (e *TriggerExecutor) Validate(config json.RawMessage) error {
	if e == nil || e.Manager == nil {
		return fmt.Errorf("plugin manager is not configured")
	}
	return e.Manager.ValidateConfig(context.Background(), e.NodeType, config)
}

type ToolExecutor struct {
	Manager  *Manager
	NodeType string
}

func (e *ToolExecutor) Execute(ctx context.Context, config json.RawMessage, input map[string]any) (*node.NodeResult, error) {
	result, err := e.ExecuteTool(ctx, config, nil, input)
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("encode plugin tool output: %w", err)
	}

	return &node.NodeResult{Output: payload}, nil
}

func (e *ToolExecutor) Validate(config json.RawMessage) error {
	if e == nil || e.Manager == nil {
		return fmt.Errorf("plugin manager is not configured")
	}
	return e.Manager.ValidateConfig(context.Background(), e.NodeType, config)
}

func (e *ToolExecutor) ToolDefinition(ctx context.Context, meta node.ToolNodeMetadata, config json.RawMessage) (*llm.ToolDefinition, error) {
	if e == nil || e.Manager == nil {
		return nil, fmt.Errorf("plugin manager is not configured")
	}

	renderedConfig, err := templating.RenderJSONWithContext(ctx, config, nil)
	if err != nil {
		return nil, fmt.Errorf("render config: %w", err)
	}

	definition, err := e.Manager.ToolDefinition(ctx, e.NodeType, pluginapi.ToolNodeMetadata{
		NodeID: meta.NodeID,
		Label:  meta.Label,
	}, renderedConfig)
	if err != nil {
		return nil, err
	}
	if definition == nil {
		return nil, fmt.Errorf("tool definition is required")
	}

	return &llm.ToolDefinition{
		Type: definition.Type,
		Function: llm.ToolSpec{
			Name:        definition.Function.Name,
			Description: definition.Function.Description,
			Parameters:  definition.Function.Parameters,
		},
	}, nil
}

func (e *ToolExecutor) ExecuteTool(ctx context.Context, config json.RawMessage, args json.RawMessage, input map[string]any) (any, error) {
	if e == nil || e.Manager == nil {
		return nil, fmt.Errorf("plugin manager is not configured")
	}

	renderedConfig, err := templating.RenderJSONWithContext(ctx, config, input)
	if err != nil {
		return nil, fmt.Errorf("render config: %w", err)
	}

	return e.Manager.ExecuteTool(ctx, e.NodeType, renderedConfig, args, input)
}

func validateActionOutput(output any, handles []pluginapi.OutputHandle) error {
	if len(handles) == 0 {
		return nil
	}

	payload, ok := output.(map[string]any)
	if !ok {
		return fmt.Errorf("plugin action must return an object when custom outputs are declared")
	}

	rawMatches, ok := payload["matches"]
	if !ok {
		return fmt.Errorf("plugin action must return a matches object for custom outputs")
	}

	matches, ok := rawMatches.(map[string]any)
	if !ok {
		return fmt.Errorf("plugin action matches must be an object")
	}

	for _, handle := range handles {
		handleID := strings.TrimSpace(handle.ID)
		value, exists := matches[handleID]
		if !exists {
			return fmt.Errorf("plugin action matches is missing handle %q", handleID)
		}
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("plugin action matches[%q] must be a boolean", handleID)
		}
	}

	return nil
}

func PluginTriggerExecutionContext(subscriptionID string, payload any) map[string]any {
	return map[string]any{
		pluginTriggerEventKey: map[string]any{
			"subscription_id": strings.TrimSpace(subscriptionID),
			"payload":         payload,
		},
	}
}

func pluginTriggerEventFromInput(input map[string]any) (*pluginapi.TriggerEvent, error) {
	if len(input) == 0 {
		return nil, fmt.Errorf("plugin trigger event is missing")
	}

	rawEvent, ok := input[pluginTriggerEventKey]
	if !ok {
		return nil, fmt.Errorf("plugin trigger event is missing")
	}

	eventMap, ok := rawEvent.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("plugin trigger event has invalid shape")
	}

	subscriptionID, _ := eventMap["subscription_id"].(string)
	if strings.TrimSpace(subscriptionID) == "" {
		return nil, fmt.Errorf("plugin trigger subscription id is missing")
	}

	return &pluginapi.TriggerEvent{
		SubscriptionID: strings.TrimSpace(subscriptionID),
		Payload:        eventMap["payload"],
	}, nil
}
