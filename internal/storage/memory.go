// Package storage provides an in-memory conversation store implementation.
package storage

import (
	"context"
	"sync"
	"time"
)

// MemoryStore is an in-memory implementation of ConversationStore.
type MemoryStore struct {
	mu            sync.RWMutex
	conversations map[string]*Conversation
}

// NewMemoryStore creates a new in-memory conversation store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		conversations: make(map[string]*Conversation),
	}
}

// Get retrieves a conversation by ID.
func (s *MemoryStore) Get(ctx context.Context, id string) (*Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	conv, ok := s.conversations[id]
	if !ok {
		return nil, nil
	}

	// Return a copy to prevent external modification
	return s.copyConversation(conv), nil
}

// Save stores or updates a conversation.
func (s *MemoryStore) Save(ctx context.Context, conv *Conversation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.conversations[conv.ID] = s.copyConversation(conv)
	return nil
}

// AddMessage appends a message to a conversation.
func (s *MemoryStore) AddMessage(ctx context.Context, id, channelID string, msg Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv, ok := s.conversations[id]
	if !ok {
		conv = &Conversation{
			ID:        id,
			ChannelID: channelID,
			Messages:  make([]Message, 0),
			CreatedAt: time.Now(),
		}
		s.conversations[id] = conv
	}

	conv.Messages = append(conv.Messages, msg)
	conv.UpdatedAt = time.Now()

	return nil
}

// Delete removes a conversation.
func (s *MemoryStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.conversations, id)
	return nil
}

// Cleanup removes conversations older than the given duration.
func (s *MemoryStore) Cleanup(ctx context.Context, olderThan time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	for id, conv := range s.conversations {
		if conv.UpdatedAt.Before(cutoff) {
			delete(s.conversations, id)
		}
	}

	return nil
}

// copyConversation creates a deep copy of a conversation.
func (s *MemoryStore) copyConversation(conv *Conversation) *Conversation {
	copy := &Conversation{
		ID:        conv.ID,
		ChannelID: conv.ChannelID,
		Messages:  make([]Message, len(conv.Messages)),
		CreatedAt: conv.CreatedAt,
		UpdatedAt: conv.UpdatedAt,
	}
	for i, msg := range conv.Messages {
		copy.Messages[i] = msg
	}
	return copy
}

// Len returns the number of conversations in the store.
func (s *MemoryStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.conversations)
}
