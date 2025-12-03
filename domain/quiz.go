package domain

import (
	"github.com/google/uuid"
)

type Quiz struct {
	Id         uuid.UUID      `json:"id"`
	UserId     uuid.UUID      `json:"userId"`
	DocumentId uuid.UUID      `json:"documentId"`
	Title      string         `json:"title"`
	Questions  []QuizQuestion `json:"questions"`
}

type QuizQuestion struct {
	QuestionText       string   `json:"questionText"`
	Options            []Option `json:"options"`
	CorrectOptionLabel string   `json:"correctOptionLabel"`
	Explanation        string   `json:"explanation,omitempty"`
}

type Option struct {
	No        int    `json:"no"`
	Label     string `json:"label"`
	Text      string `json:"text"`
	IsCorrect bool   `json:"isCorrect"`
}

func NewQuiz(userId, documentId uuid.UUID, title string) *Quiz {
	return &Quiz{
		Id:         uuid.New(),
		UserId:     userId,
		DocumentId: documentId,
		Title:      title,
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
