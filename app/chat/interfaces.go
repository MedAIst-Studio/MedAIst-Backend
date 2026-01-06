package chat

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	GetChatsByUserID(ctx context.Context, userId uuid.UUID) ([]Chat, error)
	GetChatByID(ctx context.Context, chatId uuid.UUID) (*GetChatByIDResponse, error)
	DeleteChat(ctx context.Context, chatId uuid.UUID) error
	UpdateChat(ctx context.Context, chatId uuid.UUID, title string) error
}
