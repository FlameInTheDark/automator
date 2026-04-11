package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/FlameInTheDark/emerald/internal/node"
)

type SortNode struct{}

type sortConfig struct {
	InputPath  string `json:"inputPath"`
	OutputPath string `json:"outputPath"`
	FieldPath  string `json:"fieldPath"`
	Direction  string `json:"direction"`
	ValueType  string `json:"valueType"`
}

func (e *SortNode) Execute(_ context.Context, config json.RawMessage, input map[string]any) (*node.NodeResult, error) {
	cfg, err := parseSortConfig(config)
	if err != nil {
		return nil, err
	}

	payload, items, err := prepareTransformPayload(input, cfg.InputPath, cfg.OutputPath)
	if err != nil {
		return nil, err
	}

	sortedItems := cloneSlice(items)
	var compareErr error
	sort.SliceStable(sortedItems, func(left, right int) bool {
		comparison, err := compareTransformValues(resolveTransformField(sortedItems[left], cfg.FieldPath), resolveTransformField(sortedItems[right], cfg.FieldPath), cfg.ValueType)
		if err != nil {
			compareErr = err
			return false
		}
		if cfg.Direction == "desc" {
			return comparison > 0
		}
		return comparison < 0
	})
	if compareErr != nil {
		return nil, compareErr
	}

	if err := writeTransformOutput(payload, cfg.OutputPath, cfg.InputPath, sortedItems); err != nil {
		return nil, err
	}

	return marshalTransformPayload(payload)
}

func (e *SortNode) Validate(config json.RawMessage) error {
	_, err := parseSortConfig(config)
	return err
}

type LimitNode struct{}

type limitConfig struct {
	InputPath  string `json:"inputPath"`
	OutputPath string `json:"outputPath"`
	MaxItems   int    `json:"maxItems"`
}

func (e *LimitNode) Execute(_ context.Context, config json.RawMessage, input map[string]any) (*node.NodeResult, error) {
	cfg, err := parseLimitConfig(config)
	if err != nil {
		return nil, err
	}

	payload, items, err := prepareTransformPayload(input, cfg.InputPath, cfg.OutputPath)
	if err != nil {
		return nil, err
	}

	limited := cloneSlice(items)
	if cfg.MaxItems < len(limited) {
		limited = limited[:cfg.MaxItems]
	}

	if err := writeTransformOutput(payload, cfg.OutputPath, cfg.InputPath, limited); err != nil {
		return nil, err
	}

	return marshalTransformPayload(payload)
}

func (e *LimitNode) Validate(config json.RawMessage) error {
	_, err := parseLimitConfig(config)
	return err
}

type RemoveDuplicatesNode struct{}

type removeDuplicatesConfig struct {
	InputPath  string `json:"inputPath"`
	OutputPath string `json:"outputPath"`
	Strategy   string `json:"strategy"`
	FieldPath  string `json:"fieldPath"`
	Keep       string `json:"keep"`
}

func (e *RemoveDuplicatesNode) Execute(_ context.Context, config json.RawMessage, input map[string]any) (*node.NodeResult, error) {
	cfg, err := parseRemoveDuplicatesConfig(config)
	if err != nil {
		return nil, err
	}

	payload, items, err := prepareTransformPayload(input, cfg.InputPath, cfg.OutputPath)
	if err != nil {
		return nil, err
	}

	deduped, err := removeDuplicateItems(items, cfg)
	if err != nil {
		return nil, err
	}

	if err := writeTransformOutput(payload, cfg.OutputPath, cfg.InputPath, deduped); err != nil {
		return nil, err
	}

	return marshalTransformPayload(payload)
}

func (e *RemoveDuplicatesNode) Validate(config json.RawMessage) error {
	_, err := parseRemoveDuplicatesConfig(config)
	return err
}

type SummarizeNode struct{}

type summarizeConfig struct {
	InputPath   string            `json:"inputPath"`
	OutputPath  string            `json:"outputPath"`
	GroupByPath string            `json:"groupByPath"`
	Metrics     []summarizeMetric `json:"metrics"`
}

type summarizeMetric struct {
	Name      string `json:"name"`
	Op        string `json:"op"`
	FieldPath string `json:"fieldPath"`
}

type summarizeGroup struct {
	Key     any
	Count   int
	Metrics map[string]*metricAccumulator
}

type metricAccumulator struct {
	Op    string
	Count int
	Sum   float64
	Min   *float64
	Max   *float64
}

func (e *SummarizeNode) Execute(_ context.Context, config json.RawMessage, input map[string]any) (*node.NodeResult, error) {
	cfg, err := parseSummarizeConfig(config)
	if err != nil {
		return nil, err
	}

	payload, items, err := prepareTransformPayload(input, cfg.InputPath, cfg.OutputPath)
	if err != nil {
		return nil, err
	}

	groups := make(map[string]*summarizeGroup)
	groupOrder := make([]string, 0)

	for _, item := range items {
		groupKeyID := "__all__"
		groupKeyValue := any("all")
		if strings.TrimSpace(cfg.GroupByPath) != "" {
			groupKeyValue, err = resolveTransformFieldValue(item, cfg.GroupByPath)
			if err != nil {
				return nil, fmt.Errorf("resolve summarize groupByPath: %w", err)
			}
			groupKeyID = canonicalTransformKey(groupKeyValue)
		}

		group := groups[groupKeyID]
		if group == nil {
			group = &summarizeGroup{
				Key:     groupKeyValue,
				Metrics: make(map[string]*metricAccumulator, len(cfg.Metrics)),
			}
			for _, metric := range cfg.Metrics {
				group.Metrics[metric.Name] = &metricAccumulator{Op: metric.Op}
			}
			groups[groupKeyID] = group
			groupOrder = append(groupOrder, groupKeyID)
		}

		group.Count++
		for _, metric := range cfg.Metrics {
			if err := applyMetric(group.Metrics[metric.Name], metric, item); err != nil {
				return nil, fmt.Errorf("apply summarize metric %q: %w", metric.Name, err)
			}
		}
	}

	summary := map[string]any{
		"count": len(items),
	}
	if strings.TrimSpace(cfg.GroupByPath) == "" {
		summary["metrics"] = finalizeMetricMap(groups["__all__"], cfg.Metrics)
	} else {
		groupEntries := make([]map[string]any, 0, len(groupOrder))
		byGroup := make(map[string]any, len(groupOrder))
		for _, groupKeyID := range groupOrder {
			group := groups[groupKeyID]
			entry := map[string]any{
				"key":     group.Key,
				"count":   group.Count,
				"metrics": finalizeMetricMap(group, cfg.Metrics),
			}
			groupEntries = append(groupEntries, entry)
			byGroup[summaryGroupKey(group.Key)] = entry
		}
		summary["groups"] = groupEntries
		summary["byGroup"] = byGroup
	}

	if err := writeTransformOutput(payload, cfg.OutputPath, cfg.InputPath, summary); err != nil {
		return nil, err
	}

	return marshalTransformPayload(payload)
}

func (e *SummarizeNode) Validate(config json.RawMessage) error {
	_, err := parseSummarizeConfig(config)
	return err
}

func parseSortConfig(config json.RawMessage) (sortConfig, error) {
	var cfg sortConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	cfg.InputPath = strings.TrimSpace(cfg.InputPath)
	cfg.OutputPath = strings.TrimSpace(cfg.OutputPath)
	cfg.FieldPath = strings.TrimSpace(cfg.FieldPath)
	cfg.Direction = strings.ToLower(strings.TrimSpace(cfg.Direction))
	cfg.ValueType = strings.ToLower(strings.TrimSpace(cfg.ValueType))
	if cfg.InputPath == "" {
		return cfg, fmt.Errorf("inputPath is required")
	}
	if cfg.Direction == "" {
		cfg.Direction = "asc"
	}
	if cfg.Direction != "asc" && cfg.Direction != "desc" {
		return cfg, fmt.Errorf("direction must be either asc or desc")
	}
	if cfg.ValueType == "" {
		cfg.ValueType = "auto"
	}
	if cfg.ValueType != "auto" && cfg.ValueType != "string" && cfg.ValueType != "number" && cfg.ValueType != "datetime" {
		return cfg, fmt.Errorf("valueType must be auto, string, number, or datetime")
	}
	return cfg, nil
}

func parseLimitConfig(config json.RawMessage) (limitConfig, error) {
	var cfg limitConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	cfg.InputPath = strings.TrimSpace(cfg.InputPath)
	cfg.OutputPath = strings.TrimSpace(cfg.OutputPath)
	if cfg.InputPath == "" {
		return cfg, fmt.Errorf("inputPath is required")
	}
	if cfg.MaxItems < 0 {
		return cfg, fmt.Errorf("maxItems must be 0 or greater")
	}
	return cfg, nil
}

func parseRemoveDuplicatesConfig(config json.RawMessage) (removeDuplicatesConfig, error) {
	var cfg removeDuplicatesConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	cfg.InputPath = strings.TrimSpace(cfg.InputPath)
	cfg.OutputPath = strings.TrimSpace(cfg.OutputPath)
	cfg.Strategy = strings.ToLower(strings.TrimSpace(cfg.Strategy))
	cfg.FieldPath = strings.TrimSpace(cfg.FieldPath)
	cfg.Keep = strings.ToLower(strings.TrimSpace(cfg.Keep))
	if cfg.InputPath == "" {
		return cfg, fmt.Errorf("inputPath is required")
	}
	if cfg.Strategy == "" {
		cfg.Strategy = "whole_item"
	}
	if cfg.Strategy != "whole_item" && cfg.Strategy != "field" {
		return cfg, fmt.Errorf("strategy must be whole_item or field")
	}
	if cfg.Strategy == "field" && cfg.FieldPath == "" {
		return cfg, fmt.Errorf("fieldPath is required when strategy is field")
	}
	if cfg.Keep == "" {
		cfg.Keep = "first"
	}
	if cfg.Keep != "first" && cfg.Keep != "last" {
		return cfg, fmt.Errorf("keep must be first or last")
	}
	return cfg, nil
}

func parseSummarizeConfig(config json.RawMessage) (summarizeConfig, error) {
	var cfg summarizeConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	cfg.InputPath = strings.TrimSpace(cfg.InputPath)
	cfg.OutputPath = strings.TrimSpace(cfg.OutputPath)
	cfg.GroupByPath = strings.TrimSpace(cfg.GroupByPath)
	if cfg.InputPath == "" {
		return cfg, fmt.Errorf("inputPath is required")
	}
	seenMetricNames := make(map[string]struct{}, len(cfg.Metrics))
	for index := range cfg.Metrics {
		cfg.Metrics[index].Name = strings.TrimSpace(cfg.Metrics[index].Name)
		cfg.Metrics[index].Op = strings.ToLower(strings.TrimSpace(cfg.Metrics[index].Op))
		cfg.Metrics[index].FieldPath = strings.TrimSpace(cfg.Metrics[index].FieldPath)
		if cfg.Metrics[index].Name == "" {
			return cfg, fmt.Errorf("metrics[%d].name is required", index)
		}
		if _, exists := seenMetricNames[cfg.Metrics[index].Name]; exists {
			return cfg, fmt.Errorf("metric name %q is duplicated", cfg.Metrics[index].Name)
		}
		seenMetricNames[cfg.Metrics[index].Name] = struct{}{}
		switch cfg.Metrics[index].Op {
		case "count":
		case "sum", "avg", "min", "max":
			if cfg.Metrics[index].FieldPath == "" {
				return cfg, fmt.Errorf("metrics[%d].fieldPath is required for %s", index, cfg.Metrics[index].Op)
			}
		default:
			return cfg, fmt.Errorf("metrics[%d].op must be count, sum, avg, min, or max", index)
		}
	}
	return cfg, nil
}

func prepareTransformPayload(input map[string]any, inputPath string, outputPath string) (map[string]any, []any, error) {
	payload := cloneInputMap(input)
	value, err := resolveTransformFieldValue(payload, inputPath)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve inputPath: %w", err)
	}
	items, ok := value.([]any)
	if !ok {
		return nil, nil, fmt.Errorf("inputPath %q must resolve to an array, got %T", inputPath, value)
	}
	return payload, items, nil
}

func marshalTransformPayload(payload map[string]any) (*node.NodeResult, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode output: %w", err)
	}
	return &node.NodeResult{Output: data}, nil
}

func writeTransformOutput(payload map[string]any, outputPath string, inputPath string, value any) error {
	targetPath := strings.TrimSpace(outputPath)
	if targetPath == "" {
		targetPath = strings.TrimSpace(inputPath)
	}
	if targetPath == "" {
		return fmt.Errorf("output path is required")
	}

	tokens, err := parseTransformPath(targetPath)
	if err != nil {
		return fmt.Errorf("parse output path: %w", err)
	}
	if len(tokens) == 0 {
		return fmt.Errorf("output path is invalid")
	}

	if err := setTransformPathValue(payload, tokens, value); err != nil {
		return fmt.Errorf("write output path %q: %w", targetPath, err)
	}
	return nil
}

func removeDuplicateItems(items []any, cfg removeDuplicatesConfig) ([]any, error) {
	seen := make(map[string]struct{}, len(items))
	result := make([]any, 0, len(items))

	appendItem := func(item any) error {
		key, err := duplicateKeyForItem(item, cfg)
		if err != nil {
			return err
		}
		if _, exists := seen[key]; exists {
			return nil
		}
		seen[key] = struct{}{}
		result = append(result, item)
		return nil
	}

	if cfg.Keep == "last" {
		for index := len(items) - 1; index >= 0; index-- {
			if err := appendItem(items[index]); err != nil {
				return nil, err
			}
		}
		reverseAnySlice(result)
		return result, nil
	}

	for _, item := range items {
		if err := appendItem(item); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func duplicateKeyForItem(item any, cfg removeDuplicatesConfig) (string, error) {
	if cfg.Strategy == "field" {
		value, err := resolveTransformFieldValue(item, cfg.FieldPath)
		if err != nil {
			return "", fmt.Errorf("resolve fieldPath: %w", err)
		}
		return canonicalTransformKey(value), nil
	}
	return canonicalTransformKey(item), nil
}

func resolveTransformField(item any, fieldPath string) any {
	if strings.TrimSpace(fieldPath) == "" {
		return item
	}

	value, err := resolveTransformFieldValue(item, fieldPath)
	if err != nil {
		return nil
	}
	return value
}

func resolveTransformFieldValue(value any, path string) (any, error) {
	tokens, err := parseTransformPath(path)
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return value, nil
	}

	current := value
	for _, token := range tokens {
		if token.index != nil {
			array, ok := current.([]any)
			if !ok {
				return nil, fmt.Errorf("cannot access index %d on %T", *token.index, current)
			}
			if *token.index < 0 || *token.index >= len(array) {
				return nil, fmt.Errorf("index %d out of range", *token.index)
			}
			current = array[*token.index]
			continue
		}

		key := token.key
		object, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cannot access key %q on %T", key, current)
		}
		next, exists := object[key]
		if !exists {
			return nil, fmt.Errorf("key %q not found", key)
		}
		current = next
	}

	return current, nil
}

func setTransformPathValue(root map[string]any, tokens []transformPathToken, value any) error {
	var current any = root
	for index := 0; index < len(tokens)-1; index++ {
		token := tokens[index]
		nextToken := tokens[index+1]

		if token.index != nil {
			array, ok := current.([]any)
			if !ok {
				return fmt.Errorf("cannot access index %d on %T", *token.index, current)
			}
			if *token.index < 0 || *token.index >= len(array) {
				return fmt.Errorf("index %d out of range", *token.index)
			}
			if array[*token.index] == nil {
				if nextToken.index != nil {
					return fmt.Errorf("cannot create nested array at index %d", *token.index)
				}
				array[*token.index] = make(map[string]any)
			}
			current = array[*token.index]
			continue
		}

		key := token.key
		object, ok := current.(map[string]any)
		if !ok {
			return fmt.Errorf("cannot access key %q on %T", key, current)
		}

		next, exists := object[key]
		if !exists || next == nil {
			if nextToken.index != nil {
				return fmt.Errorf("path %q requires an existing array", key)
			}
			next = make(map[string]any)
			object[key] = next
		}
		current = next
	}

	last := tokens[len(tokens)-1]
	if last.index != nil {
		array, ok := current.([]any)
		if !ok {
			return fmt.Errorf("cannot access index %d on %T", *last.index, current)
		}
		if *last.index < 0 || *last.index >= len(array) {
			return fmt.Errorf("index %d out of range", *last.index)
		}
		array[*last.index] = value
		return nil
	}

	object, ok := current.(map[string]any)
	if !ok {
		return fmt.Errorf("cannot access key %q on %T", last.key, current)
	}
	object[last.key] = value
	return nil
}

type transformPathToken struct {
	key   string
	index *int
}

func parseTransformPath(path string) ([]transformPathToken, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return nil, nil
	}

	tokens := make([]transformPathToken, 0)
	for index := 0; index < len(trimmed); {
		switch trimmed[index] {
		case '.':
			index++
		case '[':
			end := strings.IndexByte(trimmed[index:], ']')
			if end == -1 {
				return nil, fmt.Errorf("missing closing bracket")
			}
			content := strings.TrimSpace(trimmed[index+1 : index+end])
			if content == "" {
				return nil, fmt.Errorf("empty path segment")
			}
			if content[0] == '\'' || content[0] == '"' {
				if len(content) < 2 || content[len(content)-1] != content[0] {
					return nil, fmt.Errorf("invalid quoted key")
				}
				tokens = append(tokens, transformPathToken{key: content[1 : len(content)-1]})
			} else {
				arrayIndex, err := strconv.Atoi(content)
				if err != nil {
					return nil, fmt.Errorf("invalid array index %q", content)
				}
				tokens = append(tokens, transformPathToken{index: &arrayIndex})
			}
			index += end + 1
		default:
			start := index
			for index < len(trimmed) && trimmed[index] != '.' && trimmed[index] != '[' {
				index++
			}
			segment := strings.TrimSpace(trimmed[start:index])
			if segment == "" {
				return nil, fmt.Errorf("invalid path")
			}
			tokens = append(tokens, transformPathToken{key: segment})
		}
	}

	return tokens, nil
}

func cloneInputMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return make(map[string]any)
	}

	cloned := make(map[string]any, len(input))
	for key, value := range input {
		cloned[key] = cloneTransformValue(value)
	}
	return cloned
}

func cloneTransformValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		cloned := make(map[string]any, len(typed))
		for key, child := range typed {
			cloned[key] = cloneTransformValue(child)
		}
		return cloned
	case []any:
		cloned := make([]any, len(typed))
		for index, child := range typed {
			cloned[index] = cloneTransformValue(child)
		}
		return cloned
	default:
		return typed
	}
}

func cloneSlice(values []any) []any {
	cloned := make([]any, len(values))
	for index, value := range values {
		cloned[index] = cloneTransformValue(value)
	}
	return cloned
}

func reverseAnySlice(values []any) {
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}
}

func compareTransformValues(left any, right any, valueType string) (int, error) {
	leftNil := left == nil
	rightNil := right == nil
	if leftNil || rightNil {
		switch {
		case leftNil && rightNil:
			return 0, nil
		case leftNil:
			return 1, nil
		default:
			return -1, nil
		}
	}

	switch valueType {
	case "string":
		return strings.Compare(fmt.Sprint(left), fmt.Sprint(right)), nil
	case "number":
		leftNumber, err := toTransformNumber(left)
		if err != nil {
			return 0, err
		}
		rightNumber, err := toTransformNumber(right)
		if err != nil {
			return 0, err
		}
		return compareFloat(leftNumber, rightNumber), nil
	case "datetime":
		leftTime, err := toTransformTime(left)
		if err != nil {
			return 0, err
		}
		rightTime, err := toTransformTime(right)
		if err != nil {
			return 0, err
		}
		if leftTime.Before(rightTime) {
			return -1, nil
		}
		if leftTime.After(rightTime) {
			return 1, nil
		}
		return 0, nil
	default:
		if leftNumber, err := toTransformNumber(left); err == nil {
			if rightNumber, rightErr := toTransformNumber(right); rightErr == nil {
				return compareFloat(leftNumber, rightNumber), nil
			}
		}
		if leftTime, err := toTransformTime(left); err == nil {
			if rightTime, rightErr := toTransformTime(right); rightErr == nil {
				if leftTime.Before(rightTime) {
					return -1, nil
				}
				if leftTime.After(rightTime) {
					return 1, nil
				}
				return 0, nil
			}
		}
		return strings.Compare(fmt.Sprint(left), fmt.Sprint(right)), nil
	}
}

func toTransformNumber(value any) (float64, error) {
	switch typed := value.(type) {
	case float64:
		return typed, nil
	case float32:
		return float64(typed), nil
	case int:
		return float64(typed), nil
	case int8:
		return float64(typed), nil
	case int16:
		return float64(typed), nil
	case int32:
		return float64(typed), nil
	case int64:
		return float64(typed), nil
	case uint:
		return float64(typed), nil
	case uint8:
		return float64(typed), nil
	case uint16:
		return float64(typed), nil
	case uint32:
		return float64(typed), nil
	case uint64:
		return float64(typed), nil
	case json.Number:
		return typed.Float64()
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return 0, fmt.Errorf("empty string is not a number")
		}
		return strconv.ParseFloat(trimmed, 64)
	default:
		return 0, fmt.Errorf("%T is not a number", value)
	}
}

func toTransformTime(value any) (time.Time, error) {
	switch typed := value.(type) {
	case time.Time:
		return typed, nil
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return time.Time{}, fmt.Errorf("empty string is not a datetime")
		}
		layouts := []string{
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02 15:04:05",
			"2006-01-02",
		}
		for _, layout := range layouts {
			parsed, err := time.Parse(layout, trimmed)
			if err == nil {
				return parsed, nil
			}
		}
		return time.Time{}, fmt.Errorf("unable to parse %q as datetime", trimmed)
	default:
		return time.Time{}, fmt.Errorf("%T is not a datetime", value)
	}
}

func compareFloat(left float64, right float64) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func canonicalTransformKey(value any) string {
	data, err := json.Marshal(value)
	if err == nil {
		return string(data)
	}
	return fmt.Sprintf("%#v", value)
}

func summaryGroupKey(value any) string {
	switch typed := value.(type) {
	case nil:
		return "null"
	case string:
		return typed
	case bool:
		return strconv.FormatBool(typed)
	case json.Number:
		return typed.String()
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return fmt.Sprintf("%v", typed)
	default:
		return canonicalTransformKey(value)
	}
}

func applyMetric(accumulator *metricAccumulator, metric summarizeMetric, item any) error {
	if accumulator == nil {
		return fmt.Errorf("metric accumulator is not configured")
	}

	if metric.Op == "count" {
		accumulator.Count++
		return nil
	}

	rawValue, err := resolveTransformFieldValue(item, metric.FieldPath)
	if err != nil {
		return err
	}

	number, err := toTransformNumber(rawValue)
	if err != nil {
		return err
	}

	accumulator.Count++
	accumulator.Sum += number
	if accumulator.Min == nil || number < *accumulator.Min {
		next := number
		accumulator.Min = &next
	}
	if accumulator.Max == nil || number > *accumulator.Max {
		next := number
		accumulator.Max = &next
	}
	return nil
}

func finalizeMetricMap(group *summarizeGroup, metrics []summarizeMetric) map[string]any {
	result := make(map[string]any, len(metrics))
	if group == nil {
		return result
	}

	for _, metric := range metrics {
		accumulator := group.Metrics[metric.Name]
		switch metric.Op {
		case "count":
			result[metric.Name] = group.Count
		case "sum":
			result[metric.Name] = accumulator.Sum
		case "avg":
			if accumulator.Count == 0 {
				result[metric.Name] = 0
			} else {
				result[metric.Name] = accumulator.Sum / float64(accumulator.Count)
			}
		case "min":
			if accumulator.Min == nil {
				result[metric.Name] = nil
			} else {
				result[metric.Name] = *accumulator.Min
			}
		case "max":
			if accumulator.Max == nil {
				result[metric.Name] = nil
			} else {
				result[metric.Name] = *accumulator.Max
			}
		}
	}

	return result
}
