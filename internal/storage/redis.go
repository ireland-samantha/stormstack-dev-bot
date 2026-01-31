// Package storage provides a Redis conversation store stub.
package storage

import (
	"context"
	"errors"
	"time"
)

// RedisStore is a Redis implementation of ConversationStore.
// This is a stub implementation for future use.
type RedisStore struct {
	address  string
	password string
}

// NewRedisStore creates a new Redis conversation store.
func NewRedisStore(address, password string) *RedisStore {
	return &RedisStore{
		address:  address,
		password: password,
	}
}

// Get retrieves a conversation by ID.
func (s *RedisStore) Get(ctx context.Context, id string) (*Conversation, error) {
	return nil, errors.New("redis store not implemented")
}

// Save stores or updates a conversation.
func (s *RedisStore) Save(ctx context.Context, conv *Conversation) error {
	return errors.New("redis store not implemented")
}

// AddMessage appends a message to a conversation.
func (s *RedisStore) AddMessage(ctx context.Context, id, channelID string, msg Message) error {
	return errors.New("redis store not implemented")
}

// Delete removes a conversation.
func (s *RedisStore) Delete(ctx context.Context, id string) error {
	return errors.New("redis store not implemented")
}

// Cleanup removes conversations older than the given duration.
func (s *RedisStore) Cleanup(ctx context.Context, olderThan time.Duration) error {
	return errors.New("redis store not implemented")
}
