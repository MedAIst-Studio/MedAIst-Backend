package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	UnallowedClarificationMsg = `Bu konuyu biraz daha açmak isterim! Ancak, şu anda hangi meseleden bahsettiğinizi bilmiyorum. 🤔 
	Lütfen bana hangi konu, olay, fikir veya sorundan bahsettiğinizi söyleyin ki, o meseleyi daha detaylı bir şekilde açıklayabileyim.`
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
