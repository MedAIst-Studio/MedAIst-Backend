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

func (r *Repository) GetChatHistoryForAIContext(ctx context.Context, chatId uuid.UUID, limit int) ([]domain.ChatMessage, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT content, is_user FROM messages WHERE chat_id = $1  ORDER BY created_at ASC LIMIT $2", chatId, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []domain.ChatMessage
	for rows.Next() {
		var msg domain.ChatMessage
		var isUser bool
		if err := rows.Scan(&msg.Content, &isUser); err != nil {
			return nil, err
		}
		msg.UpdateRole(isUser)
		messages = append(messages, msg)
	}

	return messages[:len(messages)-1], nil
}
