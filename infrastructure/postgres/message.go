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
	// En son N mesajı al (DESC), yeni mesaj henüz kaydedilmediği için OFFSET gerekmez
	// limit parametresi direkt olarak istenen mesaj sayısıdır
	rows, err := r.db.QueryContext(ctx,
		"SELECT content, is_user FROM messages WHERE chat_id = $1 ORDER BY created_at DESC LIMIT $2",
		chatId, limit)
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

	// DESC ile aldık, tersine çevir ki kronolojik sırada olsun (eski → yeni)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}
