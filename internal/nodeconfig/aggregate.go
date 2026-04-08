package nodeconfig

import (
	"encoding/json"
	"fmt"
	"strings"
)

type AggregateConfig struct {
	IDOverrides map[string]string `json:"idOverrides"`
}

func ParseAggregateConfig(config json.RawMessage) (AggregateConfig, error) {
	cfg := AggregateConfig{
		IDOverrides: make(map[string]string),
	}

	if len(config) == 0 {
		return cfg, nil
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(config, &payload); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}

	rawOverrides, ok := payload["idOverrides"]
	if !ok || len(rawOverrides) == 0 || string(rawOverrides) == "null" {
		return cfg, nil
	}

	var parsedOverrides map[string]any
	if err := json.Unmarshal(rawOverrides, &parsedOverrides); err != nil {
		return cfg, fmt.Errorf("idOverrides must be an object")
	}

	for rawSourceID, rawAlias := range parsedOverrides {
		sourceID := strings.TrimSpace(rawSourceID)
		if sourceID == "" {
			continue
		}

		alias, ok := rawAlias.(string)
		if !ok {
			return cfg, fmt.Errorf("idOverrides[%q] must be a string", sourceID)
		}

		alias = strings.TrimSpace(alias)
		if alias == "" {
			continue
		}

		cfg.IDOverrides[sourceID] = alias
	}

	return cfg, nil
}

func (c AggregateConfig) ResolveNodeID(sourceNodeID string) string {
	sourceNodeID = strings.TrimSpace(sourceNodeID)
	if sourceNodeID == "" {
		return ""
	}

	if alias, ok := c.IDOverrides[sourceNodeID]; ok {
		alias = strings.TrimSpace(alias)
		if alias != "" {
			return alias
		}
	}

	return sourceNodeID
}

func (c AggregateConfig) ValidateResolvedNodeIDs(sourceNodeIDs []string) error {
	resolvedIDs := make(map[string]string, len(sourceNodeIDs))

	for _, sourceNodeID := range sourceNodeIDs {
		sourceNodeID = strings.TrimSpace(sourceNodeID)
		if sourceNodeID == "" {
			continue
		}

		resolvedNodeID := c.ResolveNodeID(sourceNodeID)
		if previousSourceID, exists := resolvedIDs[resolvedNodeID]; exists && previousSourceID != sourceNodeID {
			return fmt.Errorf("aggregate output id %q is assigned to both %q and %q", resolvedNodeID, previousSourceID, sourceNodeID)
		}

		resolvedIDs[resolvedNodeID] = sourceNodeID
	}

	return nil
}
