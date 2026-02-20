package message

import (
	"context"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type Repository interface {
	SaveMessage(ctx context.Context, message *domain.Message) error
	CreateChat(ctx context.Context, chat *domain.Chat) error
	GetChatHistoryForAIContext(ctx context.Context, chatId uuid.UUID, limit int) ([]domain.ChatMessage, error)
	DeleteMessage(ctx context.Context, messageId uuid.UUID) error
	GetFileMetadata(ctx context.Context, fileId uuid.UUID) (*domain.File, error)
}

type VectorRepository interface {
	SearchByEmbedding(ctx context.Context, fileId uuid.UUID, embedding []float32) ([]string, error)
}

type EmbeddingRequest struct {
	Text string `json:"text"`
}

type EmbeddingResponse struct {
	Vector []float32 `json:"vector"`
}

type EmbeddingService interface {
	GenerateEmbedding(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error)
}

type AIService interface {
	SendChatRequest(ctx context.Context, req *domain.ChatRequest) (*domain.ChatResponse, error)
	GenerateChatTitle(ctx context.Context, req *domain.ChatRequest) (*domain.ChatResponse, error)
	ExpandQuery(ctx context.Context, content string) ([]string, error)
	RerankChunks(ctx context.Context, chunks []string, userQuestion string) ([]string, error)
	ExtractTextFromImage(ctx context.Context, imageUrl string) (string, error)
	SolveQuestionWithReasoning(ctx context.Context, questionText string) (string, error)
}
