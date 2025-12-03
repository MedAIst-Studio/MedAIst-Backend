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
	AttemptID      uuid.UUID        `json:"attempt_id"`
	Score          int              `json:"score"`
	TotalQuestions int              `json:"total_questions"`
	Percentage     float64          `json:"percentage"`
	Answers        []AnswerFeedback `json:"answers"` // Detaylı cevap bilgisi
}

// Kullanıcıya her soru için feedback
type AnswerFeedback struct {
	QuestionNo     int    `json:"question_no"`
	SelectedOption string `json:"selected_option"`
	CorrectOption  string `json:"correct_option"`
	IsCorrect      bool   `json:"is_correct"`
	Explanation    string `json:"explanation"` // Doğru cevap açıklaması
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

// Kullanıcı cevabı kaydı
type QuizAttemptAnswer struct {
	ID             uuid.UUID
	AttemptID      uuid.UUID
	QuestionNo     int
	SelectedOption string
	CorrectOption  string
	IsCorrect      bool
}

type SolveQuizHandler struct {
	repo Repository
}

func NewSolveQuizHandler(repo Repository) *SolveQuizHandler {
	return &SolveQuizHandler{repo: repo}
}

// @Summary		Solve Quiz
// @Description	Submit answers for a quiz and get the score with detailed feedback
// @Tags			Quiz
// @Accept			json
// @Produce		json
// @Param			id		path		string				true	"Quiz ID"
// @Param			request	body		SolveQuizRequest	true	"Solve Quiz request"
// @Success		200		{object}	SolveQuizResponse
// @Failure		400
// @Failure		404
// @Failure		500
// @Router			/quizzes/{id}/solve [post]
func (h *SolveQuizHandler) Handle(ctx context.Context, req *SolveQuizRequest) (*SolveQuizResponse, int, error) {
	userID := domain.GetUserID(ctx)

	// Quiz cevaplarını ve açıklamalarını getir
	quizData, err := h.repo.GetQuizDataForSolving(ctx, req.ID)
	if err != nil {
		return nil, 404, err
	}

	// Cevapları kontrol et ve feedback oluştur
	correctCount := 0
	var answerFeedbacks []AnswerFeedback
	var attemptAnswers []QuizAttemptAnswer

	for _, userAnswer := range req.Answers {
		questionData, exists := quizData.Questions[userAnswer.QuestionNo]
		if !exists {
			continue
		}

		isCorrect := questionData.CorrectOption == userAnswer.SelectedOption
		if isCorrect {
			correctCount++
		}

		// Feedback için
		answerFeedbacks = append(answerFeedbacks, AnswerFeedback{
			QuestionNo:     userAnswer.QuestionNo,
			SelectedOption: userAnswer.SelectedOption,
			CorrectOption:  questionData.CorrectOption,
			IsCorrect:      isCorrect,
			Explanation:    questionData.Explanation,
		})

		// Veritabanına kaydetmek için
		attemptAnswers = append(attemptAnswers, QuizAttemptAnswer{
			ID:             uuid.New(),
			QuestionNo:     userAnswer.QuestionNo,
			SelectedOption: userAnswer.SelectedOption,
			CorrectOption:  questionData.CorrectOption,
			IsCorrect:      isCorrect,
		})
	}

	percentage := 0.0
	if quizData.TotalQuestions > 0 {
		percentage = (float64(correctCount) / float64(quizData.TotalQuestions)) * 100
	}

	// Attempt oluştur
	attemptID := uuid.New()
	attempt := &QuizAttempt{
		ID:             attemptID,
		QuizID:         req.ID,
		UserID:         userID,
		Score:          correctCount,
		TotalQuestions: quizData.TotalQuestions,
		CompletedAt:    time.Now(),
	}

	if err := h.repo.SaveQuizAttemptWithAnswers(ctx, attempt, attemptAnswers); err != nil {
		return nil, 500, err
	}

	return &SolveQuizResponse{
		AttemptID:      attemptID,
		Score:          correctCount,
		TotalQuestions: quizData.TotalQuestions,
		Percentage:     percentage,
		Answers:        answerFeedbacks,
	}, 200, nil
}
