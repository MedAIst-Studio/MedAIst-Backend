package postgres

import (
	"context"

	"github.com/MedAIst-Studio/MedAIist-Backend/app/chat"
	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

func (r *Repository) CreateChat(ctx context.Context, chat *domain.Chat) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO chats (id, user_id, file_id, title) VALUES ($1, $2, $3, $4)",
		chat.Id, chat.UserId, chat.PdfId, chat.Title)
	return err
}

func (r *Repository) GetChatsByUserID(ctx context.Context, userId uuid.UUID) ([]chat.Chat, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, title, created_at FROM chats WHERE user_id = $1", userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []chat.Chat
	for rows.Next() {
		var chat chat.Chat
		if err := rows.Scan(&chat.Id, &chat.Title, &chat.CreatedAt); err != nil {
			return nil, err
		}
		chats = append(chats, chat)
	}
	return chats, nil
}
