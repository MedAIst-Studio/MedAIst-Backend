package quiz

import (
	"context"
	"time"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type SolveQuizRequest struct {
	ID      uuid.UUID `param:"id" validate:"required,uuid"`
	Answers []Answer  `json:"answers" validate:"required,dive"`
}

type Answer struct {
	QuestionNo     int    `json:"question_no" validate:"required,min=1"`
	SelectedOption string `json:"selected_option" validate:"required,oneof=A B C D E"`
}

type SolveQuizResponse struct {
	AttemptID      uuid.UUID `json:"attempt_id"`
	Score          int       `json:"score"`
	TotalQuestions int       `json:"total_questions"`
	Percentage     float64   `json:"percentage"`
}

type SolveQuizHandler struct {
	repo Repository
}

func NewSolveQuizHandler(repo Repository) *SolveQuizHandler {
	return &SolveQuizHandler{repo: repo}
}

// @Summary     Solve Quiz
// @Description Submit answers for a quiz and get the score
// @Tags        Quiz
// @Accept      json
// @Produce     json
// @Param       id      path        string              true    "Quiz ID"
// @Param       request body        SolveQuizRequest    true    "Solve Quiz request"
// @Success     200     {object}    SolveQuizResponse
// @Failure     400
// @Failure     404
// @Failure     500
// @Router      /quizzes/{id}/solve [post]
func (h *SolveQuizHandler) Handle(ctx context.Context, req *SolveQuizRequest) (*SolveQuizResponse, int, error) {
	userID := domain.GetUserID(ctx)

	quizAnswers, totalQuestions, err := h.repo.GetQuizAnswersById(ctx, req.ID)
	if err != nil {
		return nil, 404, err
	}

	correctCount := 0
	for _, answer := range req.Answers {
		if correctOption, exists := quizAnswers[answer.QuestionNo]; exists && correctOption == answer.SelectedOption {
			correctCount++
		}
	}

	percentage := 0.0
	if totalQuestions > 0 {
		percentage = (float64(correctCount) / float64(totalQuestions)) * 100
	}

	attemptID := uuid.New()
	attempt := &QuizAttempt{
		ID:             attemptID,
		QuizID:         req.ID,
		UserID:         userID,
		Score:          correctCount,
		TotalQuestions: totalQuestions,
		CompletedAt:    time.Now(),
	}

	if err := h.repo.SaveQuizAttempt(ctx, attempt); err != nil {
		return nil, 500, err
	}

	return &SolveQuizResponse{
		AttemptID:      attemptID,
		Score:          correctCount,
		TotalQuestions: totalQuestions,
		Percentage:     percentage,
	}, 200, nil
}

type QuizAttempt struct {
	ID               uuid.UUID
	QuizID           uuid.UUID
	UserID           uuid.UUID
	Score            int
	TotalQuestions   int
	StartedAt        time.Time
	CompletedAt      time.Time
	TimeSpentSeconds int
}
