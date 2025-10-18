package domain

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	Id        uuid.UUID
	ChatId    uuid.UUID
	Content   string
	IsUser    bool
	CreatedAt time.Time
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
