package storage

import (
	"sync"

	"backend-ai-sdlc/internal/models"
)

type MemoryStorage struct {
	conversations map[string]*models.Conversation
	mu            sync.RWMutex
}

func NewMemoryStorage() Storage {
	return &MemoryStorage{
		conversations: make(map[string]*models.Conversation),
	}
}

func (m *MemoryStorage) GetConversation(id string) (*models.Conversation, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	conv, exists := m.conversations[id]
	return conv, exists
}

func (m *MemoryStorage) GetOrCreateConversation(id string) (*models.Conversation, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	conv, exists := m.conversations[id]
	if !exists {
		conv = &models.Conversation{ID: id}
		m.conversations[id] = conv
	}
	return conv, exists
}

func (m *MemoryStorage) UpdateConversation(conv *models.Conversation) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.conversations[conv.ID] = conv
}
