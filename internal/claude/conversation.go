// Package claude provides conversation management for Claude interactions.
package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/ireland-samantha/stormstack-dev-bot/internal/storage"
)

// ToolExecutor executes a tool and returns the result.
type ToolExecutor func(ctx context.Context, name string, input json.RawMessage) (string, error)

// ConversationManager manages conversations with Claude.
type ConversationManager struct {
	client       *Client
	store        storage.ConversationStore
	systemPrompt string
	tools        []anthropic.ToolUnionParam
	executor     ToolExecutor
	logger       *slog.Logger
}

// NewConversationManager creates a new conversation manager.
func NewConversationManager(
	client *Client,
	store storage.ConversationStore,
	systemPrompt string,
	executor ToolExecutor,
	logger *slog.Logger,
) *ConversationManager {
	return &ConversationManager{
		client:       client,
		store:        store,
		systemPrompt: systemPrompt,
		tools:        GetAllTools(),
		executor:     executor,
		logger:       logger,
	}
}

// ProcessMessage processes a user message and returns the response.
func (m *ConversationManager) ProcessMessage(
	ctx context.Context,
	conversationID string,
	channelID string,
	userMessage string,
) (string, error) {
	// Get existing conversation or create new one
	conv, err := m.store.Get(ctx, conversationID)
	if err != nil {
		return "", fmt.Errorf("failed to get conversation: %w", err)
	}

	// Build message history
	messages := m.buildMessageHistory(conv)

	// Add user message
	messages = append(messages, BuildUserMessage(userMessage))

	// Store user message
	if err := m.store.AddMessage(ctx, conversationID, channelID, storage.Message{
		Role:    "user",
		Content: userMessage,
	}); err != nil {
		m.logger.Warn("failed to store user message", "error", err)
	}

	// Process with Claude (with tool use loop)
	response, err := m.processWithToolLoop(ctx, messages)
	if err != nil {
		return "", err
	}

	// Store assistant response
	if err := m.store.AddMessage(ctx, conversationID, channelID, storage.Message{
		Role:    "assistant",
		Content: response,
	}); err != nil {
		m.logger.Warn("failed to store assistant message", "error", err)
	}

	return response, nil
}

// buildMessageHistory builds message params from stored conversation.
func (m *ConversationManager) buildMessageHistory(conv *storage.Conversation) []anthropic.MessageParam {
	if conv == nil {
		return []anthropic.MessageParam{}
	}

	messages := make([]anthropic.MessageParam, 0, len(conv.Messages))
	for _, msg := range conv.Messages {
		switch msg.Role {
		case "user":
			messages = append(messages, BuildUserMessage(msg.Content))
		case "assistant":
			messages = append(messages, BuildAssistantMessage(msg.Content))
		}
	}
	return messages
}

// processWithToolLoop handles the Claude response including tool use.
func (m *ConversationManager) processWithToolLoop(
	ctx context.Context,
	messages []anthropic.MessageParam,
) (string, error) {
	const maxIterations = 20

	for i := 0; i < maxIterations; i++ {
		// Call Claude
		response, err := m.client.CreateMessageWithTools(ctx, m.systemPrompt, messages, m.tools)
		if err != nil {
			return "", fmt.Errorf("claude API error: %w", err)
		}

		// Check if we need to handle tool use
		if !HasToolUse(response) {
			// No tool use, return the text response
			return ExtractTextContent(response), nil
		}

		// Extract tool uses
		toolUses := ExtractToolUses(response)
		m.logger.Debug("processing tool uses", "count", len(toolUses))

		// Build assistant message with the full response (text + tool uses)
		assistantContent := make([]anthropic.ContentBlockParamUnion, 0, len(response.Content))
		for _, block := range response.Content {
			switch b := block.AsAny().(type) {
			case anthropic.TextBlock:
				if b.Text != "" {
					assistantContent = append(assistantContent, anthropic.NewTextBlock(b.Text))
				}
			case anthropic.ToolUseBlock:
				assistantContent = append(assistantContent, anthropic.ContentBlockParamOfRequestToolUseBlock(b.ID, b.Input, b.Name))
			}
		}
		messages = append(messages, anthropic.MessageParam{
			Role:    anthropic.MessageParamRoleAssistant,
			Content: assistantContent,
		})

		// Execute tools and collect results
		var results []ToolResult
		for _, toolUse := range toolUses {
			m.logger.Debug("executing tool", "name", toolUse.Name, "id", toolUse.ID)

			result, err := m.executor(ctx, toolUse.Name, toolUse.Input)
			isError := err != nil
			if isError {
				result = FormatError(err)
			}

			results = append(results, ToolResult{
				ToolUseID: toolUse.ID,
				Result:    result,
				IsError:   isError,
			})
		}

		// Add tool results as user message
		messages = append(messages, BuildToolResultsMessage(results))
	}

	return "", fmt.Errorf("exceeded maximum tool use iterations (%d)", maxIterations)
}

// SetSystemPrompt updates the system prompt.
func (m *ConversationManager) SetSystemPrompt(prompt string) {
	m.systemPrompt = prompt
}

// ClearConversation removes a conversation from storage.
func (m *ConversationManager) ClearConversation(ctx context.Context, conversationID string) error {
	return m.store.Delete(ctx, conversationID)
}
