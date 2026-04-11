package templating

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/FlameInTheDark/emerald/internal/pipeline"
)

var placeholderPattern = regexp.MustCompile(`\{\{\s*([^{}]+?)\s*\}\}`)

const nodeSelectorContextKey = "__executed_node_outputs__"

type pathToken struct {
	key   string
	index *int
}

// RenderString resolves {{template}} expressions against the provided input.
// Nested map paths and array indexes such as {{input.nodes[0].status}} are supported.
func RenderString(value string, input map[string]any) (string, error) {
	return RenderStringWithContext(context.Background(), value, input)
}

// RenderStringWithContext resolves {{template}} expressions against the provided
// input plus any execution-runtime data available on ctx.
func RenderStringWithContext(ctx context.Context, value string, input map[string]any) (string, error) {
	matches := placeholderPattern.FindAllStringSubmatchIndex(value, -1)
	if len(matches) == 0 {
		return value, nil
	}

	renderContext := buildRenderContext(ctx, input)

	var builder strings.Builder
	lastIndex := 0

	for _, match := range matches {
		if len(match) < 4 {
			continue
		}

		builder.WriteString(value[lastIndex:match[0]])
		expression := strings.TrimSpace(value[match[2]:match[3]])

		resolved, err := resolveExpression(renderContext, expression)
		if err != nil {
			return "", err
		}

		rendered, err := stringifyValue(resolved)
		if err != nil {
			return "", err
		}

		builder.WriteString(rendered)
		lastIndex = match[1]
	}

	builder.WriteString(value[lastIndex:])
	return builder.String(), nil
}

// RenderStrings walks structs, slices, and string maps to render templates in string values.
func RenderStrings(target any, input map[string]any) error {
	return RenderStringsWithContext(context.Background(), target, input)
}

// RenderStringsWithContext walks structs, slices, and string maps to render
// templates in string values using the supplied execution context.
func RenderStringsWithContext(ctx context.Context, target any, input map[string]any) error {
	if target == nil {
		return nil
	}

	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Ptr || value.IsNil() {
		return fmt.Errorf("target must be a non-nil pointer")
	}

	return renderValue(ctx, value.Elem(), input)
}

// RenderJSON renders template placeholders across arbitrary JSON-compatible
// payloads and re-encodes the rendered value back into canonical JSON.
func RenderJSON(payload json.RawMessage, input map[string]any) (json.RawMessage, error) {
	return RenderJSONWithContext(context.Background(), payload, input)
}

// RenderJSONWithContext renders template placeholders across arbitrary
// JSON-compatible payloads with access to execution-runtime data on ctx.
func RenderJSONWithContext(ctx context.Context, payload json.RawMessage, input map[string]any) (json.RawMessage, error) {
	if len(payload) == 0 {
		return json.RawMessage("{}"), nil
	}

	var value any
	if err := json.Unmarshal(payload, &value); err != nil {
		return nil, fmt.Errorf("decode json payload: %w", err)
	}

	if err := RenderStringsWithContext(ctx, &value, input); err != nil {
		return nil, err
	}

	rendered, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode json payload: %w", err)
	}

	return rendered, nil
}

func renderValue(ctx context.Context, value reflect.Value, input map[string]any) error {
	if !value.IsValid() {
		return nil
	}

	switch value.Kind() {
	case reflect.Pointer:
		if value.IsNil() {
			return nil
		}
		return renderValue(ctx, value.Elem(), input)
	case reflect.Interface:
		if value.IsNil() {
			return nil
		}

		elem := value.Elem()
		copyValue := reflect.New(elem.Type()).Elem()
		copyValue.Set(elem)
		if err := renderValue(ctx, copyValue, input); err != nil {
			return err
		}
		value.Set(copyValue)
		return nil
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			field := value.Field(i)
			if !field.CanSet() {
				continue
			}
			if err := renderValue(ctx, field, input); err != nil {
				return err
			}
		}
		return nil
	case reflect.String:
		rendered, err := RenderStringWithContext(ctx, value.String(), input)
		if err != nil {
			return err
		}
		value.SetString(rendered)
		return nil
	case reflect.Slice:
		if value.Type().Elem().Kind() == reflect.Uint8 {
			return nil
		}
		for i := 0; i < value.Len(); i++ {
			if err := renderValue(ctx, value.Index(i), input); err != nil {
				return err
			}
		}
		return nil
	case reflect.Array:
		for i := 0; i < value.Len(); i++ {
			if err := renderValue(ctx, value.Index(i), input); err != nil {
				return err
			}
		}
		return nil
	case reflect.Map:
		if value.IsNil() || value.Type().Key().Kind() != reflect.String {
			return nil
		}

		for _, key := range value.MapKeys() {
			item := value.MapIndex(key)
			copyValue := reflect.New(value.Type().Elem()).Elem()
			copyValue.Set(item)
			if err := renderValue(ctx, copyValue, input); err != nil {
				return err
			}
			value.SetMapIndex(key, copyValue)
		}
		return nil
	default:
		return nil
	}
}

func resolveExpression(context map[string]any, expression string) (any, error) {
	if nodeID, remainder, ok, err := parseNodeSelectorExpression(expression); ok || err != nil {
		if err != nil {
			return nil, fmt.Errorf("template %q: %w", expression, err)
		}

		current, err := resolveNodeSelector(context, nodeID)
		if err != nil {
			return nil, fmt.Errorf("template %q: %w", expression, err)
		}

		if strings.TrimSpace(remainder) == "" {
			return current, nil
		}

		tokens, err := parseExpression(remainder)
		if err != nil {
			return nil, fmt.Errorf("template %q: %w", expression, err)
		}

		for _, token := range tokens {
			if token.index != nil {
				current, err = resolveIndex(current, *token.index)
				if err != nil {
					return nil, fmt.Errorf("template %q: %w", expression, err)
				}
				continue
			}

			current, err = resolveKey(current, token.key)
			if err != nil {
				return nil, fmt.Errorf("template %q: %w", expression, err)
			}
		}

		return current, nil
	}

	tokens, err := parseExpression(expression)
	if err != nil {
		return nil, fmt.Errorf("template %q: %w", expression, err)
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf("template %q: empty expression", expression)
	}

	rootToken := tokens[0]
	current, ok := context[rootToken.key]
	if !ok {
		return nil, fmt.Errorf("template %q: %s not found", expression, rootToken.key)
	}

	if rootToken.index != nil {
		current, err = resolveIndex(current, *rootToken.index)
		if err != nil {
			return nil, fmt.Errorf("template %q: %w", expression, err)
		}
	}

	for _, token := range tokens[1:] {
		if token.index != nil {
			current, err = resolveIndex(current, *token.index)
			if err != nil {
				return nil, fmt.Errorf("template %q: %w", expression, err)
			}
			continue
		}

		current, err = resolveKey(current, token.key)
		if err != nil {
			return nil, fmt.Errorf("template %q: %w", expression, err)
		}
	}

	return current, nil
}

func buildRenderContext(ctx context.Context, input map[string]any) map[string]any {
	renderContext := make(map[string]any, len(input)+2)
	for key, item := range input {
		renderContext[key] = item
	}
	renderContext["input"] = input

	if executedNodeOutputs := executedNodeOutputsFromContext(ctx); len(executedNodeOutputs) > 0 {
		renderContext[nodeSelectorContextKey] = executedNodeOutputs
	}

	return renderContext
}

func executedNodeOutputsFromContext(ctx context.Context) map[string]any {
	runtime := pipeline.RuntimeFromContext(ctx)
	if runtime == nil || runtime.State == nil {
		return nil
	}

	outputs := make(map[string]any, len(runtime.State.NodeResults))
	for nodeID, result := range runtime.State.NodeResults {
		if result == nil {
			continue
		}
		outputs[nodeID] = decodeRuntimeNodeOutput(result.Output)
	}

	if len(outputs) == 0 {
		return nil
	}

	return outputs
}

func decodeRuntimeNodeOutput(raw json.RawMessage) any {
	if len(raw) == 0 {
		return map[string]any{}
	}

	var decoded any
	if err := json.Unmarshal(raw, &decoded); err == nil {
		return decoded
	}

	return string(raw)
}

func parseNodeSelectorExpression(expression string) (string, string, bool, error) {
	trimmed := strings.TrimSpace(expression)
	if !strings.HasPrefix(trimmed, "$(") {
		return "", "", false, nil
	}

	index := 2
	for index < len(trimmed) && trimmed[index] == ' ' {
		index++
	}
	if index >= len(trimmed) {
		return "", "", true, fmt.Errorf("invalid node selector")
	}

	quote := trimmed[index]
	if quote != '\'' && quote != '"' {
		return "", "", true, fmt.Errorf("node selector must use single or double quotes")
	}
	index++

	start := index
	for index < len(trimmed) && trimmed[index] != quote {
		index++
	}
	if index >= len(trimmed) {
		return "", "", true, fmt.Errorf("node selector is missing a closing quote")
	}

	nodeID := strings.TrimSpace(trimmed[start:index])
	index++

	for index < len(trimmed) && trimmed[index] == ' ' {
		index++
	}
	if index >= len(trimmed) || trimmed[index] != ')' {
		return "", "", true, fmt.Errorf("node selector is missing a closing parenthesis")
	}
	index++

	remainder := strings.TrimSpace(trimmed[index:])
	if remainder != "" && !strings.HasPrefix(remainder, ".") && !strings.HasPrefix(remainder, "[") {
		return "", "", true, fmt.Errorf("node selector must be followed by dot or index access")
	}
	if nodeID == "" {
		return "", "", true, fmt.Errorf("node selector id is required")
	}

	return nodeID, remainder, true, nil
}

func resolveNodeSelector(context map[string]any, nodeID string) (any, error) {
	raw, ok := context[nodeSelectorContextKey]
	if !ok {
		return nil, fmt.Errorf("node %q has not executed in this run", nodeID)
	}

	nodeOutputs, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("node execution context is unavailable")
	}

	value, exists := nodeOutputs[nodeID]
	if !exists {
		return nil, fmt.Errorf("node %q has not executed in this run", nodeID)
	}

	return value, nil
}

func parseExpression(expression string) ([]pathToken, error) {
	var tokens []pathToken

	for i := 0; i < len(expression); {
		switch expression[i] {
		case '.':
			i++
			continue
		case '[':
			end := strings.IndexByte(expression[i:], ']')
			if end == -1 {
				return nil, fmt.Errorf("missing closing bracket")
			}

			content := strings.TrimSpace(expression[i+1 : i+end])
			if content == "" {
				return nil, fmt.Errorf("empty array index")
			}

			if content[0] == '"' || content[0] == '\'' {
				if len(content) < 2 || content[len(content)-1] != content[0] {
					return nil, fmt.Errorf("invalid quoted key")
				}
				tokens = append(tokens, pathToken{key: content[1 : len(content)-1]})
			} else {
				index, err := strconv.Atoi(content)
				if err != nil {
					return nil, fmt.Errorf("invalid array index %q", content)
				}
				tokens = append(tokens, pathToken{index: &index})
			}

			i += end + 1
		default:
			start := i
			for i < len(expression) {
				ch := expression[i]
				if ch == '.' || ch == '[' {
					break
				}
				i++
			}

			segment := strings.TrimSpace(expression[start:i])
			if segment == "" {
				return nil, fmt.Errorf("invalid path")
			}

			tokens = append(tokens, pathToken{key: segment})
		}
	}

	return tokens, nil
}

func resolveKey(current any, key string) (any, error) {
	switch typed := current.(type) {
	case map[string]any:
		value, ok := typed[key]
		if !ok {
			return nil, fmt.Errorf("key %q not found", key)
		}
		return value, nil
	}

	value := reflect.ValueOf(current)
	if !value.IsValid() {
		return nil, fmt.Errorf("cannot access key %q on nil", key)
	}

	switch value.Kind() {
	case reflect.Map:
		if value.Type().Key().Kind() != reflect.String {
			return nil, fmt.Errorf("cannot access key %q on %T", key, current)
		}

		item := value.MapIndex(reflect.ValueOf(key))
		if !item.IsValid() {
			return nil, fmt.Errorf("key %q not found", key)
		}
		return item.Interface(), nil
	case reflect.Struct:
		field := value.FieldByNameFunc(func(name string) bool {
			return strings.EqualFold(name, key)
		})
		if !field.IsValid() {
			return nil, fmt.Errorf("field %q not found", key)
		}
		return field.Interface(), nil
	default:
		return nil, fmt.Errorf("cannot access key %q on %T", key, current)
	}
}

func resolveIndex(current any, index int) (any, error) {
	switch typed := current.(type) {
	case []any:
		if index < 0 || index >= len(typed) {
			return nil, fmt.Errorf("index %d out of range", index)
		}
		return typed[index], nil
	}

	value := reflect.ValueOf(current)
	if !value.IsValid() {
		return nil, fmt.Errorf("cannot access index %d on nil", index)
	}

	switch value.Kind() {
	case reflect.Slice, reflect.Array:
		if index < 0 || index >= value.Len() {
			return nil, fmt.Errorf("index %d out of range", index)
		}
		return value.Index(index).Interface(), nil
	default:
		return nil, fmt.Errorf("cannot access index %d on %T", index, current)
	}
}

func stringifyValue(value any) (string, error) {
	switch typed := value.(type) {
	case nil:
		return "", nil
	case string:
		return typed, nil
	case bool:
		return strconv.FormatBool(typed), nil
	case json.Number:
		return typed.String(), nil
	case fmt.Stringer:
		return typed.String(), nil
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return fmt.Sprintf("%v", typed), nil
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return "", fmt.Errorf("marshal template value: %w", err)
		}
		return string(data), nil
	}
}
