package postgres

import (
	"context"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

func (r *Repository) SaveMessage(ctx context.Context, message *domain.Message) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO messages (id, chat_id, content, is_user, created_at) VALUES ($1, $2, $3, $4, $5)",
		message.Id, message.ChatId, message.Content, message.IsUser, message.CreatedAt)
	return err
}

func (r *Repository) GetChatHistory(ctx context.Context, chatId uuid.UUID, limit int) ([]*domain.Message, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, chat_id, content, is_user FROM messages WHERE chat_id = $1 LIMIT $2 ORDER BY created_at ASC",
		chatId, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*domain.Message
	for rows.Next() {
		var msg domain.Message
		if err := rows.Scan(&msg.Id, &msg.ChatId, &msg.Content, &msg.IsUser); err != nil {
			return nil, err
		}
		messages = append(messages, &msg)
	}
	return messages, nil
}
