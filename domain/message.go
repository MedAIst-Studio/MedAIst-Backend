package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Message struct {
	Id              uuid.UUID
	ChatId          uuid.UUID
	Content         string
	IsUser          bool
	CreatedAt       time.Time
	ContextSnapshot *ContextSnapshot
	Attachments     []Attachment
}

type ContextSnapshot struct {
	Type        string `json:"type"` // "pdf", "book", etc.
	DocumentId  string `json:"document_id"`
	CachedTitle string `json:"cached_title"`
}

type Attachment struct {
	Type string `json:"type"` // "image"
	Url  string `json:"url"`
}

func NewMessage(chatId uuid.UUID, content string, isUser bool) *Message {
	return &Message{
		Id:        uuid.New(),
		ChatId:    chatId,
		Content:   content,
		IsUser:    isUser,
		CreatedAt: time.Now(),
	}
}

func (m *Message) UpdateChatId(chatId uuid.UUID) {
	m.ChatId = chatId
}

// ---------- Message DTOs ----------

type SendMessageRequest struct {
	PdfId    uuid.UUID `json:"pdf_id"`
	ChatId   uuid.UUID `json:"chat_id,omitempty"`
	Content  string    `json:"content" validate:"required,min=1"`
	ImageUrl string    `json:"image_url"`
}

type SendMessageResponse struct {
	ChatId        *uuid.UUID `json:"chat_id,omitempty"`
	AiResponse    string     `json:"ai_response"`
	Title         string     `json:"title,omitempty"`
	UserMessageId uuid.UUID  `json:"user_message_id"`
	AiMessageId   uuid.UUID  `json:"ai_message_id"`
}

// ---------- Message Interfaces ----------

type MessageRepository interface {
	SaveMessage(ctx context.Context, message *Message) error
	CreateChat(ctx context.Context, chat *Chat) error
	GetChatHistoryForAIContext(ctx context.Context, chatId uuid.UUID, limit int) ([]ChatMessage, error)
	DeleteMessage(ctx context.Context, messageId uuid.UUID) error
	GetFileMetadata(ctx context.Context, fileId uuid.UUID) (*File, error)
}

type MessageService interface {
	SendMessage(ctx context.Context, userId uuid.UUID, req *SendMessageRequest) (*SendMessageResponse, error)
}
