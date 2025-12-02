package quiz

import (
	"context"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type Repository interface {
	SaveQuiz(ctx context.Context, quiz *domain.Quiz) error
	GetQuizzesByUserID(ctx context.Context, userId uuid.UUID) (*GetQuizzesResponse, error)
	GetQuizById(ctx context.Context, quizId uuid.UUID) (*GetQuizByIdResponse, error)
	GetQuizAnswersById(ctx context.Context, quizID uuid.UUID) (map[int]string, int, error)
	SaveQuizAttempt(ctx context.Context, attempt *QuizAttempt) error
}

type AIService interface {
	GenerateQuizQuestion(ctx context.Context, chunk string) (*domain.QuizQuestion, error)
}

type ChunkService interface {
	GetChunks(ctx context.Context, fileId uuid.UUID) ([]string, error)
	GetChunksByPages(ctx context.Context, fileId uuid.UUID, pageRange []int) ([]string, error)
}
