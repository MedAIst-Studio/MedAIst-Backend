package message

import (
	"context"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type Repository interface {
	SaveMessage(ctx context.Context, message *domain.Message) error
	CreateChat(ctx context.Context, chat *domain.Chat) error
	GetPastMessages(ctx context.Context, chatId uuid.UUID, limit int) ([]domain.ThirdPartyAIMessage, error)
}

type VectorDB interface {
	SearchTextByEmbedding(ctx context.Context, fileId uuid.UUID, embedding []float32) (string, error)
}

type EmbeddingService interface {
	ConvertToEmbedding(ctx context.Context, text string) ([]float32, error)
}

type ChatService interface {
	Request(ctx context.Context, req *domain.ThirdPartyAIRequest) (*domain.ThirdPartyAIResponse, error)
}
