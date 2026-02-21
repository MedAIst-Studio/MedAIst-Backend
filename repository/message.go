package repository

import (
	"context"
	"encoding/json"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

func (r *Repository) SaveMessage(ctx context.Context, message *domain.Message) error {
	var contextVal, attachmentsVal interface{}
	var err error

	if message.ContextSnapshot != nil {
		contextVal, err = json.Marshal(message.ContextSnapshot)
		if err != nil {
			return err
		}
	}

	if len(message.Attachments) > 0 {
		attachmentsVal, err = json.Marshal(message.Attachments)
		if err != nil {
			return err
		}
	}

	_, err = r.db.ExecContext(ctx, "INSERT INTO messages (id, chat_id, content, is_user, created_at, context_snapshot, attachments) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		message.Id, message.ChatId, message.Content, message.IsUser, message.CreatedAt, contextVal, attachmentsVal)
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
		if isUser {
			msg.Role = domain.RoleUser
		} else {
			msg.Role = domain.RoleAssistant
		}
		messages = append(messages, msg)
	}

	return messages[:len(messages)-1], nil
}

func (r *Repository) DeleteMessage(ctx context.Context, messageId uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM messages WHERE id = $1", messageId)
	return err
}
