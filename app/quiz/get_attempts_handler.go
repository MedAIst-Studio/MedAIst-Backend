package quiz

import (
	"context"
	"time"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type GetQuizAttemptsRequest struct {
	// Query params for filtering/pagination
	Limit  int `query:"limit" validate:"omitempty,min=1,max=100"`
	Offset int `query:"offset" validate:"omitempty,min=0"`
}

type GetQuizAttemptsResponse struct {
	Attempts   []QuizAttemptDetail `json:"attempts"`
	TotalCount int                 `json:"total_count"`
}

type QuizAttemptDetail struct {
	ID             uuid.UUID `json:"id"`
	QuizID         uuid.UUID `json:"quiz_id"`
	QuizTitle      string    `json:"quiz_title"`
	Score          int       `json:"score"`
	TotalQuestions int       `json:"total_questions"`
	Percentage     float64   `json:"percentage"`
	TimeSpent      int       `json:"time_spent_seconds"`
	CompletedAt    time.Time `json:"completed_at"`
}

type GetQuizAttemptsHandler struct {
	repo Repository
}

func NewGetQuizAttemptsHandler(repo Repository) *GetQuizAttemptsHandler {
	return &GetQuizAttemptsHandler{repo: repo}
}

// @Summary		Get Quiz Attempts
// @Description	Get all quiz attempts for the current user
// @Tags			Quiz
// @Accept			json
// @Produce		json
// @Param			limit	query		int	false	"Limit"		default(10)
// @Param			offset	query		int	false	"Offset"	default(0)
// @Success		200		{object}	GetQuizAttemptsResponse
// @Failure		401
// @Failure		500
// @Router			/quizzes/attempts [get]
func (h *GetQuizAttemptsHandler) Handle(ctx context.Context, req *GetQuizAttemptsRequest) (*GetQuizAttemptsResponse, int, error) {
	userID := domain.GetUserID(ctx)

	// Set defaults
	if req.Limit == 0 {
		req.Limit = 10
	}

	attempts, totalCount, err := h.repo.GetQuizAttemptsByUserId(ctx, userID, req.Limit, req.Offset)
	if err != nil {
		return nil, 500, err
	}

	return &GetQuizAttemptsResponse{
		Attempts:   attempts,
		TotalCount: totalCount,
	}, 200, nil
}
