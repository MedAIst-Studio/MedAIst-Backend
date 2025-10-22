package chat

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	GetChatsByUserID(ctx context.Context, userId uuid.UUID) ([]Chat, error)
}
