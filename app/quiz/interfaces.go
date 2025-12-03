package quiz

import (
	"context"
	"time"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type Repository interface {
	SaveQuiz(ctx context.Context, quiz *domain.Quiz) error
	GetQuizzesByUserID(ctx context.Context, userId uuid.UUID) (*GetQuizzesResponse, error)
	GetQuizById(ctx context.Context, quizId uuid.UUID) (*GetQuizByIdResponse, error)
	GetQuizAnswersById(ctx context.Context, quizID uuid.UUID) (map[int]string, int, error)
	SaveQuizAttempt(ctx context.Context, attempt *QuizAttempt) error
	GetQuizAttemptsByUserId(ctx context.Context, userID uuid.UUID, limit, offset int) ([]QuizAttemptDetail, int, error)
	GetQuizDataForSolving(ctx context.Context, quizID uuid.UUID) (*QuizSolvingData, error)
	SaveQuizAttemptWithAnswers(ctx context.Context, attempt *QuizAttempt, answers []QuizAttemptAnswer) error
}

type AIService interface {
	GenerateQuizQuestion(ctx context.Context, chunk string) (*domain.QuizQuestion, error)
}

type ChunkService interface {
	GetChunks(ctx context.Context, fileId uuid.UUID) ([]string, error)
	GetChunksByPages(ctx context.Context, fileId uuid.UUID, pageRange []int) ([]string, error)
}

type QuestionData struct {
	CorrectOption string
	Explanation   string
}

type QuizSolvingData struct {
	TotalQuestions int
	Questions      map[int]QuestionData
}

type QuizAttemptDetail struct {
	ID             uuid.UUID `json:"id"`
	QuizID         uuid.UUID `json:"quiz_id"`
	QuizTitle      string    `json:"quiz_title"`
	Score          int       `json:"score"`
	TotalQuestions int       `json:"total_questions"`
	Percentage     float64   `json:"percentage"`
	CompletedAt    time.Time `json:"completed_at"`
}

type AttemptDetailResponse struct {
	ID             uuid.UUID             `json:"id"`
	QuizID         uuid.UUID             `json:"quiz_id"`
	QuizTitle      string                `json:"quiz_title"`
	Score          int                   `json:"score"`
	TotalQuestions int                   `json:"total_questions"`
	Percentage     float64               `json:"percentage"`
	CompletedAt    time.Time             `json:"completed_at"`
	Answers        []AttemptAnswerDetail `json:"answers"`
}

type AttemptAnswerDetail struct {
	QuestionNo     int    `json:"question_no"`
	QuestionText   string `json:"question_text"`
	SelectedOption string `json:"selected_option"`
	CorrectOption  string `json:"correct_option"`
	IsCorrect      bool   `json:"is_correct"`
	Explanation    string `json:"explanation"`
}
