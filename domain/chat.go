package domain

import (
	"time"

	"github.com/google/uuid"
)

type Chat struct {
	Id       uuid.UUID `json:"id"`
	UserId   uuid.UUID `json:"user_id"`
	PdfId    uuid.UUID `json:"pdf_id"`
	Title    string    `json:"title"`
	CreateAt time.Time `json:"create_at"`
}

func NewChat(userId, pdfId uuid.UUID, title string) *Chat {
	return &Chat{
		Id:       uuid.New(),
		UserId:   userId,
		PdfId:    pdfId,
		Title:    title,
		CreateAt: time.Now(),
	}
}
