package quiz

import (
	"context"

	"github.com/google/uuid"
)

type GetAttemptByIdRequest struct {
	Id uuid.UUID `params:"id" validate:"required"`
}

type GetAttemptByIdResponse struct {
	AttemptDetailResponse
}

type GetAttemptByIdHandler struct {
	repo Repository
}

func NewGetAttemptByIdHandler(repo Repository) *GetAttemptByIdHandler {
	return &GetAttemptByIdHandler{repo: repo}
}

// @Summary		Get Quiz Attempt by ID
// @Description	Retrieve a specific quiz attempt by its ID for the authenticated user
// @Tags			Quiz
// @Accept			json
// @Produce		json
// @Param			id	path		string	true	"Attempt ID"
// @Success		200	{object}	GetAttemptByIdResponse
// @Failure		404
// @Router			/quizzes/attempts/{id} [get]
func (h *GetAttemptByIdHandler) Handle(ctx context.Context, req *GetAttemptByIdRequest) (*GetAttemptByIdResponse, int, error) {
	attempt, err := h.repo.GetQuizAttemptById(ctx, req.Id)
	if err != nil {
		return nil, 404, err
	}

	// Attempt'e ait cevapları al
	answers, err := h.repo.GetAttemptAnswers(ctx, req.Id)
	if err != nil {
		return nil, 500, err
	}

	return &GetAttemptByIdResponse{
		AttemptDetailResponse: AttemptDetailResponse{
			ID:             attempt.ID,
			QuizID:         attempt.QuizID,
			QuizTitle:      attempt.QuizTitle,
			Score:          attempt.Score,
			TotalQuestions: attempt.TotalQuestions,
			Percentage:     attempt.Percentage,
			CompletedAt:    attempt.CompletedAt,
			Answers:        answers,
		},
	}, 200, nil
}
