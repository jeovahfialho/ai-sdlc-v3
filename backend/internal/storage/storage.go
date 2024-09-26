package storage

import "backend-ai-sdlc/internal/models"

type Storage interface {
	GetOrCreateConversation(id string) (*models.Conversation, bool)
	UpdateConversation(conv *models.Conversation)
	GetConversation(id string) (*models.Conversation, bool)
}
