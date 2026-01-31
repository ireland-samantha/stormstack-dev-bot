// Package storage provides conversation storage interfaces and implementations.
package storage

import (
	"context"
	"time"
)

// Message represents a single message in a conversation.
type Message struct {
	Role      string    `json:"role"`       // "user" or "assistant"
	Content   string    `json:"content"`    // The message content
	Timestamp time.Time `json:"timestamp"`  // When the message was sent
}

// Conversation represents a conversation thread.
type Conversation struct {
	ID        string    `json:"id"`         // Unique identifier (thread_ts)
	ChannelID string    `json:"channel_id"` // Slack channel ID
	Messages  []Message `json:"messages"`   // Message history
	CreatedAt time.Time `json:"created_at"` // When the conversation started
	UpdatedAt time.Time `json:"updated_at"` // Last activity
}

// ConversationStore provides storage for conversation history.
type ConversationStore interface {
	// Get retrieves a conversation by ID. Returns nil if not found.
	Get(ctx context.Context, id string) (*Conversation, error)

	// Save stores or updates a conversation.
	Save(ctx context.Context, conv *Conversation) error

	// AddMessage appends a message to a conversation.
	// Creates the conversation if it doesn't exist.
	AddMessage(ctx context.Context, id, channelID string, msg Message) error

	// Delete removes a conversation.
	Delete(ctx context.Context, id string) error

	// Cleanup removes conversations older than the given duration.
	Cleanup(ctx context.Context, olderThan time.Duration) error
}
