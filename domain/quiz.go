package domain

import (
	"context"
	"time"

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
	Difficulty         string   `json:"difficulty,omitempty"`
	Topic              string   `json:"topic,omitempty"`
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

// ---------- Quiz Tracking & Solving Models ----------

type QuizSolvingData struct {
	TotalQuestions int
	Questions      map[int]QuestionData
}

type QuestionData struct {
	CorrectOption string
	Explanation   string
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

type QuizAttemptAnswer struct {
	ID             uuid.UUID
	AttemptID      uuid.UUID
	QuestionNo     int
	SelectedOption string
	CorrectOption  string
	IsCorrect      bool
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

// ---------- Quiz Request and Response DTOs ----------

type GenerateQuizRequest struct {
	FileId    uuid.UUID `json:"fileId" validate:"required,uuid"`
	QuizName  string    `json:"quizName" validate:"required"`
	StartPage int       `json:"startPage,omitempty"`
	EndPage   int       `json:"endPage,omitempty"`
}

type GenerateQuizResponse struct {
	QuizID uuid.UUID `json:"quizId"`
}

type GetQuizzesRequest struct{}

type GetQuizzesResponse []QuizResponseDTO

type QuizResponseDTO struct {
	Id             string    `json:"id"`
	Title          string    `json:"title"`
	TotalQuestions int       `json:"totalQuestions"`
	CreatedAt      time.Time `json:"createdAt"`
}

type GetQuizByIdRequest struct {
	QuizID uuid.UUID `params:"id"`
}

type GetQuizByIdResponse Quiz

type GetAttemptsRequest struct {
	Limit  int `query:"limit"`
	Offset int `query:"offset"`
}

type GetAttemptsResponse struct {
	Attempts []QuizAttemptDetail `json:"attempts"`
	Count    int                 `json:"count"`
}

type GetAttemptByIdRequest struct {
	Id uuid.UUID `params:"id" validate:"required"`
}

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
	Answers        []AnswerFeedback `json:"answers"`
}

type AnswerFeedback struct {
	QuestionNo     int    `json:"question_no"`
	SelectedOption string `json:"selected_option"`
	CorrectOption  string `json:"correct_option"`
	IsCorrect      bool   `json:"is_correct"`
	Explanation    string `json:"explanation"`
}

// ---------- Quiz Interfaces ----------

type QuizRepository interface {
	SaveQuiz(ctx context.Context, quiz *Quiz) error
	GetQuizzesByUserID(ctx context.Context, userId uuid.UUID) (*GetQuizzesResponse, error)
	GetQuizById(ctx context.Context, quizId uuid.UUID) (*GetQuizByIdResponse, error)
	GetQuizAnswersById(ctx context.Context, quizID uuid.UUID) (map[int]string, int, error)
	SaveQuizAttempt(ctx context.Context, attempt *QuizAttempt) error
	GetQuizAttemptsByUserId(ctx context.Context, userID uuid.UUID, limit, offset int) ([]QuizAttemptDetail, int, error)
	GetQuizDataForSolving(ctx context.Context, quizID uuid.UUID) (*QuizSolvingData, error)
	SaveQuizAttemptWithAnswers(ctx context.Context, attempt *QuizAttempt, answers []QuizAttemptAnswer) error
	GetQuizAttemptById(ctx context.Context, attemptId uuid.UUID) (*QuizAttemptDetail, error)
	GetAttemptAnswers(ctx context.Context, attemptId uuid.UUID) ([]AttemptAnswerDetail, error)
}

type ChunkService interface {
	GetChunks(ctx context.Context, fileId uuid.UUID) ([]string, error)
	GetChunksByPages(ctx context.Context, fileId uuid.UUID, pageRange []int) ([]string, error)
}

type QuizService interface {
	GenerateQuiz(ctx context.Context, userId uuid.UUID, req *GenerateQuizRequest) (*GenerateQuizResponse, error)
	SolveQuiz(ctx context.Context, userId uuid.UUID, req *SolveQuizRequest) (*SolveQuizResponse, error)
	GetQuizzes(ctx context.Context, userId uuid.UUID) (*GetQuizzesResponse, error)
	GetQuizById(ctx context.Context, req *GetQuizByIdRequest) (*GetQuizByIdResponse, error)
	GetAttempts(ctx context.Context, userId uuid.UUID, req *GetAttemptsRequest) (*GetAttemptsResponse, error)
	GetAttemptById(ctx context.Context, req *GetAttemptByIdRequest) (*AttemptDetailResponse, error)
}
