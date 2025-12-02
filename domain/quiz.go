package domain

import (
	"crypto/rand"

	"github.com/google/uuid"
)

const (
	letters = "abcdefghijklmnopqrstuvwxyz"
)

type Quiz struct {
	UserId     uuid.UUID      `json:"userId"`
	DocumentId uuid.UUID      `json:"documentId"`
	Title      string         `json:"title"`
	Questions  []QuizQuestion `json:"questions"`
}

type QuizQuestion struct {
	QuestionText       string   `json:"questionText"`
	Options            []Option `json:"options"`
	CorrectOptionLabel string   `json:"correctOptionLabel"`
}

type Option struct {
	No        int    `json:"no"`
	Label     string `json:"label"`
	Text      string `json:"text"`
	IsCorrect bool   `json:"isCorrect"`
}

func NewQuiz(userId, documentId uuid.UUID, title string) *Quiz {
	return &Quiz{
		UserId:     userId,
		DocumentId: documentId,
		Title:      formatTitle(title),
	}
}

func (q *Quiz) AddQuestion(question *QuizQuestion) {
	q.Questions = append(q.Questions, *question)
}

func (qo *QuizQuestion) AddOption() *Option {
	for _, option := range qo.Options {
		if option.IsCorrect {
			return &option
		}
	}
	return nil
}

func formatTitle(title string) string {
	b := make([]byte, 6)

	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}

	for i := range b {
		b[i] = letters[int(b[i])%len(letters)]
	}

	return title + "-" + string(b)
}
