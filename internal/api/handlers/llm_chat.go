package handlers

import (
	"context"

	"github.com/FlameInTheDark/automator/internal/llm"
)

const maxToolChatRounds = llm.DefaultMaxToolChatRounds

func runToolChat(
	ctx context.Context,
	provider llm.Provider,
	model string,
	messages []llm.Message,
	tools llm.ToolExecutor,
) (*llm.ChatResponse, []llm.ToolCall, []llm.ToolResult, error) {
	return llm.RunToolChat(ctx, provider, model, messages, tools, maxToolChatRounds)
}
