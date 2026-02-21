package services

import (
	"context"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type ChatService struct {
	repo domain.ChatRepository
}

func NewChatService(repo domain.ChatRepository) *ChatService {
	return &ChatService{
		repo: repo,
	}
}

func (s *ChatService) GetChatsByUserID(ctx context.Context, userId uuid.UUID) ([]domain.ChatResponseDTO, error) {
	return s.repo.GetChatsByUserID(ctx, userId)
}

func (s *ChatService) GetChatByID(ctx context.Context, chatId uuid.UUID) (*domain.GetChatByIDResponse, error) {
	return s.repo.GetChatByID(ctx, chatId)
}

func (s *ChatService) UpdateChat(ctx context.Context, chatId uuid.UUID, title string) error {
	return s.repo.UpdateChat(ctx, chatId, title)
}

func (s *ChatService) DeleteChat(ctx context.Context, chatId uuid.UUID) error {
	return s.repo.DeleteChat(ctx, chatId)
}
