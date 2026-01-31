// Package claude provides Anthropic Claude API integration.
package claude

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const (
	// ModelOpus is Claude Opus 4.5 model ID.
	ModelOpus = "claude-opus-4-5-20251101"
	// MaxTokens is the maximum number of tokens for responses.
	MaxTokens = 8192
)

// Client wraps the Anthropic SDK client.
type Client struct {
	client anthropic.Client
	model  string
}

// NewClient creates a new Claude API client.
func NewClient(apiKey string) *Client {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &Client{
		client: client,
		model:  ModelOpus,
	}
}

// CreateMessage sends a message to Claude and returns the response.
func (c *Client) CreateMessage(ctx context.Context, params anthropic.MessageNewParams) (*anthropic.Message, error) {
	// Ensure model is set
	if params.Model == "" {
		params.Model = anthropic.Model(c.model)
	}

	// Ensure max tokens is set
	if params.MaxTokens == 0 {
		params.MaxTokens = MaxTokens
	}

	return c.client.Messages.New(ctx, params)
}

// CreateMessageWithTools sends a message with tool definitions.
func (c *Client) CreateMessageWithTools(
	ctx context.Context,
	systemPrompt string,
	messages []anthropic.MessageParam,
	tools []anthropic.ToolUnionParam,
) (*anthropic.Message, error) {
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: MaxTokens,
		Messages:  messages,
		Tools:     tools,
	}

	if systemPrompt != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: systemPrompt},
		}
	}

	return c.client.Messages.New(ctx, params)
}

// BuildUserMessage creates a user message param.
func BuildUserMessage(content string) anthropic.MessageParam {
	return anthropic.MessageParam{
		Role: anthropic.MessageParamRoleUser,
		Content: []anthropic.ContentBlockParamUnion{
			anthropic.NewTextBlock(content),
		},
	}
}

// BuildAssistantMessage creates an assistant message param.
func BuildAssistantMessage(content string) anthropic.MessageParam {
	return anthropic.MessageParam{
		Role: anthropic.MessageParamRoleAssistant,
		Content: []anthropic.ContentBlockParamUnion{
			anthropic.NewTextBlock(content),
		},
	}
}

// BuildToolResultMessage creates a tool result message.
func BuildToolResultMessage(toolUseID, result string, isError bool) anthropic.MessageParam {
	return anthropic.MessageParam{
		Role: anthropic.MessageParamRoleUser,
		Content: []anthropic.ContentBlockParamUnion{
			anthropic.NewToolResultBlock(toolUseID, result, isError),
		},
	}
}

// BuildToolResultsMessage creates a message with multiple tool results.
func BuildToolResultsMessage(results []ToolResult) anthropic.MessageParam {
	blocks := make([]anthropic.ContentBlockParamUnion, len(results))
	for i, r := range results {
		blocks[i] = anthropic.NewToolResultBlock(r.ToolUseID, r.Result, r.IsError)
	}
	return anthropic.MessageParam{
		Role:    anthropic.MessageParamRoleUser,
		Content: blocks,
	}
}

// ToolResult represents a tool execution result.
type ToolResult struct {
	ToolUseID string
	Result    string
	IsError   bool
}

// ExtractTextContent extracts text content from a message.
func ExtractTextContent(msg *anthropic.Message) string {
	var text string
	for _, block := range msg.Content {
		switch b := block.AsAny().(type) {
		case anthropic.TextBlock:
			text += b.Text
		}
	}
	return text
}

// ExtractToolUses extracts tool use blocks from a message.
func ExtractToolUses(msg *anthropic.Message) []anthropic.ToolUseBlock {
	var toolUses []anthropic.ToolUseBlock
	for _, block := range msg.Content {
		switch b := block.AsAny().(type) {
		case anthropic.ToolUseBlock:
			toolUses = append(toolUses, b)
		}
	}
	return toolUses
}

// HasToolUse checks if a message contains tool use blocks.
func HasToolUse(msg *anthropic.Message) bool {
	return msg.StopReason == anthropic.MessageStopReasonToolUse
}

// FormatError formats an error for tool result.
func FormatError(err error) string {
	return fmt.Sprintf("Error: %v", err)
}
