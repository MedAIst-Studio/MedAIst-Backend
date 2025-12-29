package domain

import (
	"time"

	"github.com/google/uuid"
)

type Chat struct {
	Id       uuid.UUID `json:"id"`
	UserId   uuid.UUID `json:"user_id"`
	Title    string    `json:"title"`
	CreateAt time.Time `json:"create_at"`
}

func NewChat(userId uuid.UUID, title string) *Chat {
	return &Chat{
		Id:       uuid.New(),
		UserId:   userId,
		Title:    title,
		CreateAt: time.Now(),
	}
}
