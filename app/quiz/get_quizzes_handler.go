package quiz

import (
	"context"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

type GetQuizzesRequest struct {
}

type GetQuizzesResponse []QuizSummary

type QuizSummary struct {
	QuizId    string `json:"quizId"`
	Title     string `json:"title"`
	CreatedAt string `json:"createdAt"`
}

type GetQuizzesHandler struct {
	repository Repository
}

func NewGetQuizzesHandler(repository Repository) *GetQuizzesHandler {
	return &GetQuizzesHandler{
		repository: repository,
	}
}

// @Summary		Get Quizzes
// @Description	Retrieve a list of quizzes for the authenticated user
// @Tags			Quiz
// @Accept			json
// @Produce		json
// @Success		200	{object}	GetQuizzesResponse
// @Failure		500
// @Router			/quizzes [get]
func (h *GetQuizzesHandler) Handle(ctx context.Context, req *GetQuizzesRequest) (*GetQuizzesResponse, int, error) {
	userId := domain.GetUserID(ctx)
	quizzes, err := h.repository.GetQuizzesByUserID(ctx, userId)
	if err != nil {
		return nil, 500, err
	}

	return quizzes, 200, nil
}
