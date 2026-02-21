package quiz

import (
	"context"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

type QuizConfig struct {
	QuizService domain.QuizService
}

type QuizHandler struct {
	quizService domain.QuizService
}

func NewQuizHandler(config *QuizConfig) *QuizHandler {
	return &QuizHandler{
		quizService: config.QuizService,
	}
}

// @Summary		Generate Quiz
// @Description	Generate a quiz based on the provided file
// @Tags			Quiz
// @Accept			json
// @Produce		json
// @Param			request	body		domain.GenerateQuizRequest	true	"Generate Quiz request"
// @Success		200		{object}	domain.GenerateQuizResponse
// @Failure		400
// @Failure		500
// @Router			/quizzes [post]
func (h *QuizHandler) HandleGenerateQuiz(ctx context.Context, req *domain.GenerateQuizRequest) (*domain.GenerateQuizResponse, int, error) {
	userId := domain.GetUserID(ctx)
	resp, err := h.quizService.GenerateQuiz(ctx, userId, req)
	if err != nil {
		if err == domain.ErrNoContentToGenerateQuiz || err == domain.ErrInvalidRequest {
			return nil, 400, err
		}
		return nil, 500, err
	}
	return resp, 200, nil
}

// @Summary		Solve Quiz
// @Description	Submit answers for a quiz and get the score with detailed feedback
// @Tags			Quiz
// @Accept			json
// @Produce		json
// @Param			id		path		string					true	"Quiz ID"
// @Param			request	body		domain.SolveQuizRequest	true	"Solve Quiz request"
// @Success		200		{object}	domain.SolveQuizResponse
// @Failure		400
// @Failure		404
// @Failure		500
// @Router			/quizzes/{id}/solve [post]
func (h *QuizHandler) HandleSolveQuiz(ctx context.Context, req *domain.SolveQuizRequest) (*domain.SolveQuizResponse, int, error) {
	userId := domain.GetUserID(ctx)
	resp, err := h.quizService.SolveQuiz(ctx, userId, req)
	if err != nil {
		// Quick heuristic based on prev code: if err, they mapped it to 404 earlier assuming Quiz não existe.
		// For thoroughness we map generic to 500, could be refined.
		return nil, 500, err
	}
	return resp, 200, nil
}

// @Summary		Get Quizzes
// @Description	Get all quizzes for the current user
// @Tags			Quiz
// @Produce		json
// @Success		200	{object}	domain.GetQuizzesResponse
// @Failure		500
// @Router			/quizzes [get]
func (h *QuizHandler) HandleGetQuizzes(ctx context.Context, req *domain.GetQuizzesRequest) (*domain.GetQuizzesResponse, int, error) {
	userId := domain.GetUserID(ctx)
	resp, err := h.quizService.GetQuizzes(ctx, userId)
	if err != nil {
		return nil, 500, err
	}
	return resp, 200, nil
}

// @Summary		Get Quiz by ID
// @Description	Get a specific quiz by its ID
// @Tags			Quiz
// @Produce		json
// @Param			id	path		string	true	"Quiz ID"
// @Success		200	{object}	domain.GetQuizByIdResponse
// @Failure		400
// @Failure		404
// @Failure		500
// @Router			/quizzes/{id} [get]
func (h *QuizHandler) HandleGetQuizById(ctx context.Context, req *domain.GetQuizByIdRequest) (*domain.GetQuizByIdResponse, int, error) {
	resp, err := h.quizService.GetQuizById(ctx, req)
	if err != nil {
		return nil, 500, err
	}
	return resp, 200, nil
}

// @Summary		Get Quiz Attempts
// @Description	Get paginated quiz attempts for the current user
// @Tags			Quiz
// @Produce		json
// @Param			limit	query		int	false	"Page size (default 10, max 50)"
// @Param			offset	query		int	false	"Offset (default 0)"
// @Success		200		{object}	domain.GetAttemptsResponse
// @Failure		500
// @Router			/quizzes/attempts [get]
func (h *QuizHandler) HandleGetAttempts(ctx context.Context, req *domain.GetAttemptsRequest) (*domain.GetAttemptsResponse, int, error) {
	userId := domain.GetUserID(ctx)

	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = 10
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	resp, err := h.quizService.GetAttempts(ctx, userId, req)
	if err != nil {
		return nil, 500, err
	}
	return resp, 200, nil
}

// @Summary		Get Attempt Details
// @Description	Get full details of a specific quiz attempt
// @Tags			Quiz
// @Produce		json
// @Param			id	path		string	true	"Attempt ID"
// @Success		200	{object}	domain.AttemptDetailResponse
// @Failure		400
// @Failure		404
// @Failure		500
// @Router			/quizzes/attempts/{id} [get]
func (h *QuizHandler) HandleGetAttemptById(ctx context.Context, req *domain.GetAttemptByIdRequest) (*domain.AttemptDetailResponse, int, error) {
	resp, err := h.quizService.GetAttemptById(ctx, req)
	if err != nil {
		return nil, 500, err
	}
	return resp, 200, nil
}
