package llm

import (
	"context"
	"encoding/json"
	"fmt"
)

const DefaultMaxToolChatRounds = 8

type ToolExecutor interface {
	GetAllTools() []ToolDefinition
	Execute(ctx context.Context, name string, arguments json.RawMessage) (any, error)
}

type ToolResult struct {
	Tool      string `json:"tool"`
	Arguments any    `json:"arguments,omitempty"`
	Result    any    `json:"result,omitempty"`
	Error     string `json:"error,omitempty"`
}

func RunToolChat(
	ctx context.Context,
	provider Provider,
	model string,
	messages []Message,
	tools ToolExecutor,
	maxRounds int,
) (*ChatResponse, []ToolCall, []ToolResult, error) {
	if maxRounds <= 0 {
		maxRounds = DefaultMaxToolChatRounds
	}

	allToolCalls := make([]ToolCall, 0)
	allToolResults := make([]ToolResult, 0)
	totalUsage := Usage{}

	for round := 0; round < maxRounds; round++ {
		resp, err := provider.Chat(ctx, ChatRequest{
			Model:    model,
			Messages: messages,
			Tools:    tools.GetAllTools(),
		})
		if err != nil {
			return nil, nil, nil, err
		}

		totalUsage.PromptTokens += resp.Usage.PromptTokens
		totalUsage.CompletionTokens += resp.Usage.CompletionTokens
		totalUsage.TotalTokens += resp.Usage.TotalTokens

		toolCalls := normalizeToolCalls(resp.ToolCalls)
		if len(toolCalls) == 0 {
			resp.Usage = totalUsage
			return resp, allToolCalls, allToolResults, nil
		}

		allToolCalls = append(allToolCalls, toolCalls...)
		messages = append(messages, Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: toolCalls,
		})

		for _, tc := range toolCalls {
			arguments := decodeJSONValue(tc.Function.Arguments)
			toolResult := ToolResult{
				Tool:      tc.Function.Name,
				Arguments: arguments,
			}

			result, err := tools.Execute(ctx, tc.Function.Name, json.RawMessage(tc.Function.Arguments))
			payload := map[string]any{}
			if err != nil {
				toolResult.Error = err.Error()
				payload["error"] = err.Error()
			} else {
				toolResult.Result = result
				payload["result"] = result
			}

			allToolResults = append(allToolResults, toolResult)
			messages = append(messages, Message{
				Role:       "tool",
				Name:       tc.Function.Name,
				ToolCallID: tc.ID,
				Content:    marshalToolPayload(payload),
			})
		}
	}

	return nil, allToolCalls, allToolResults, fmt.Errorf("tool execution exceeded %d rounds", maxRounds)
}

func normalizeToolCalls(toolCalls []ToolCall) []ToolCall {
	if len(toolCalls) == 0 {
		return nil
	}

	normalized := make([]ToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		if tc.ID == "" {
			tc.ID = fmt.Sprintf("tool-call-%d", i+1)
		}
		if tc.Type == "" {
			tc.Type = "function"
		}
		normalized[i] = tc
	}

	return normalized
}

func decodeJSONValue(raw string) any {
	if raw == "" {
		return nil
	}

	var value any
	if err := json.Unmarshal([]byte(raw), &value); err == nil {
		return value
	}

	return raw
}

func marshalToolPayload(payload map[string]any) string {
	encoded, err := json.Marshal(payload)
	if err != nil {
		if errMessage, ok := payload["error"].(string); ok && errMessage != "" {
			return errMessage
		}
		return "{}"
	}

	return string(encoded)
}
