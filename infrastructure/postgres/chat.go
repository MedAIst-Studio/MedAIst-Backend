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
	rows, err := r.db.QueryContext(ctx, "SELECT id, title, file_id, created_at FROM chats WHERE user_id = $1", userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []chat.Chat
	for rows.Next() {
		var chat chat.Chat
		if err := rows.Scan(&chat.Id, &chat.Title, &chat.FileId, &chat.CreatedAt); err != nil {
			return nil, err
		}
		chats = append(chats, chat)
	}
	return chats, nil
}

func (r *Repository) GetChatByID(ctx context.Context, chatId uuid.UUID) (*chat.GetChatByIDResponse, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT is_user, content, created_at FROM messages WHERE chat_id = $1 ORDER BY created_at ASC", chatId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages chat.GetChatByIDResponse
	for rows.Next() {
		var msg chat.Message
		if err := rows.Scan(&msg.IsUser, &msg.Content, &msg.Timestamp); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	if len(messages) == 0 {
		return nil, domain.ErrChatNotFound
	}

	return &messages, nil
}

func (r *Repository) DeleteChat(ctx context.Context, chatId uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM chats WHERE id = $1", chatId)
	if err != nil {
		return err
	}
	if rowsAffected, _ := result.RowsAffected(); rowsAffected == 0 {
		return domain.ErrChatNotFound
	}
	return nil
}
