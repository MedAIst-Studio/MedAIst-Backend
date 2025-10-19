package postgres

import (
	"context"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

func (r *Repository) CreateChat(ctx context.Context, chat *domain.Chat) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO chats (id, user_id, file_id, title) VALUES ($1, $2, $3, $4)",
		chat.Id, chat.UserId, chat.PdfId, chat.Title)
	return err
}
