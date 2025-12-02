package quiz

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type GetQuizByIdRequest struct {
	Id uuid.UUID `params:"id" validate:"required"`
}

type GetQuizByIdResponse struct {
	ID        uuid.UUID  `json:"id"`
	Title     string     `json:"title"`
	Questions []Question `json:"questions"`
}

type Question struct {
	No            int      `json:"no"`
	Text          string   `json:"text"`
	Options       []Option `json:"options"`
	CorrectOption string   `json:"correct_option"`
}

type Option struct {
	Text  string `json:"text"`
	Label string `json:"label"`
}

type GetQuizByIdHandler struct {
	repo Repository
}

func NewGetQuizByIdHandler(repo Repository) *GetQuizByIdHandler {
	return &GetQuizByIdHandler{repo: repo}
}

// @Summary		Get Quiz by ID
// @Description	Retrieve a quiz by its ID for the authenticated user
// @Tags			Quiz
// @Accept			json
// @Produce		json
// @Param			id	path		string	true	"Quiz ID"
// @Success		200	{object}	GetQuizByIdResponse
// @Failure		404
// @Router			/quizzes/{id} [get]
func (h *GetQuizByIdHandler) Handle(ctx context.Context, req *GetQuizByIdRequest) (*GetQuizByIdResponse, int, error) {
	fmt.Println("id", req.Id)
	quiz, err := h.repo.GetQuizById(ctx, req.Id)
	fmt.Print("QUIZ", quiz)
	if err != nil {
		return nil, 404, err
	}
	return quiz, 200, nil
}
