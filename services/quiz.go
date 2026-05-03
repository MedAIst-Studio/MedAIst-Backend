package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type QuizService struct {
	aiService    domain.AIService
	repository   domain.QuizRepository
	chunkService domain.ChunkService
	streakSvc    domain.StreakService
}

func NewQuizService(
	aiService domain.AIService,
	repository domain.QuizRepository,
	chunkService domain.ChunkService,
	streakSvc domain.StreakService,
) *QuizService {
	return &QuizService{
		aiService:    aiService,
		repository:   repository,
		chunkService: chunkService,
		streakSvc:    streakSvc,
	}
}

type questionResult struct {
	question *domain.QuizQuestion
	index    int
	err      error
}

// mergeChunks combines adjacent chunks in pairs to provide richer context for question generation.
// For example, [A, B, C, D, E] becomes ["A\n\nB", "C\n\nD", "E"]
func mergeChunks(chunks []string, groupSize int) []string {
	if groupSize <= 1 || len(chunks) <= 1 {
		return chunks
	}

	var merged []string
	for i := 0; i < len(chunks); i += groupSize {
		end := i + groupSize
		if end > len(chunks) {
			end = len(chunks)
		}
		group := chunks[i:end]
		merged = append(merged, strings.Join(group, "\n\n"))
	}

	return merged
}

func (s *QuizService) GenerateQuiz(ctx context.Context, userId uuid.UUID, req *domain.GenerateQuizRequest) (*domain.GenerateQuizResponse, error) {
	quiz := domain.NewQuiz(userId, req.FileId, req.QuizName)
	var chunks []string
	var err error

	var pageRange []int
	if req.StartPage != 0 && req.EndPage != 0 {
		pageRange = []int{
			req.StartPage,
			req.EndPage,
		}
	}

	if len(pageRange) > 0 {
		if len(pageRange) != 2 {
			return nil, domain.ErrInvalidRequest
		}
		if chunks, err = s.chunkService.GetChunksByPages(ctx, req.FileId, pageRange); err != nil {
			return nil, err
		}
	} else {
		if chunks, err = s.chunkService.GetChunks(ctx, req.FileId); err != nil {
			return nil, err
		}
	}

	if len(chunks) == 0 {
		return nil, domain.ErrNoContentToGenerateQuiz
	}

	// Merge adjacent chunks in pairs for richer context
	mergedChunks := mergeChunks(chunks, 2)
	log.Printf("[QuizService] Generating quiz: %d raw chunks → %d merged chunks", len(chunks), len(mergedChunks))

	var wg sync.WaitGroup
	resultChan := make(chan questionResult, len(mergedChunks))

	for i, chunk := range mergedChunks {
		wg.Add(1)
		go func(idx int, c string) {
			defer wg.Done()

			question, err := s.aiService.GenerateQuizQuestion(ctx, c)
			if err != nil {
				log.Printf("[QuizService] Question %d generation failed: %v", idx+1, err)
			}
			resultChan <- questionResult{
				question: question,
				index:    idx,
				err:      err,
			}
		}(i, chunk)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	results := make([]questionResult, len(mergedChunks))
	for result := range resultChan {
		results[result.index] = result
	}

	// Collect successful questions, skip failures (retry is handled in AI service)
	var failedCount int
	for _, result := range results {
		if result.err != nil {
			failedCount++
			continue
		}
		quiz.AddQuestion(result.question)
	}

	// If all questions failed, return error
	if len(quiz.Questions) == 0 {
		return nil, fmt.Errorf("all %d question generations failed", len(mergedChunks))
	}

	if failedCount > 0 {
		log.Printf("[QuizService] Quiz generated with %d/%d questions (skipped %d failures)",
			len(quiz.Questions), len(mergedChunks), failedCount)
	}

	err = s.repository.SaveQuiz(ctx, quiz)
	if err != nil {
		return nil, err
	}

	return &domain.GenerateQuizResponse{
		QuizID: quiz.Id,
	}, nil
}

func (s *QuizService) SolveQuiz(ctx context.Context, userId uuid.UUID, req *domain.SolveQuizRequest) (*domain.SolveQuizResponse, error) {
	quizData, err := s.repository.GetQuizDataForSolving(ctx, req.ID)
	if err != nil {
		return nil, err // Returning error instead of handling 404 map here
	}

	correctCount := 0
	var answerFeedbacks []domain.AnswerFeedback
	var attemptAnswers []domain.QuizAttemptAnswer

	for _, userAnswer := range req.Answers {
		questionData, exists := quizData.Questions[userAnswer.QuestionNo]
		if !exists {
			continue
		}

		isCorrect := questionData.CorrectOption == userAnswer.SelectedOption
		if isCorrect {
			correctCount++
		}

		answerFeedbacks = append(answerFeedbacks, domain.AnswerFeedback{
			QuestionNo:     userAnswer.QuestionNo,
			SelectedOption: userAnswer.SelectedOption,
			CorrectOption:  questionData.CorrectOption,
			IsCorrect:      isCorrect,
			Explanation:    questionData.Explanation,
		})

		attemptAnswers = append(attemptAnswers, domain.QuizAttemptAnswer{
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

	attemptID := uuid.New()
	attempt := &domain.QuizAttempt{
		ID:             attemptID,
		QuizID:         req.ID,
		UserID:         userId,
		Score:          correctCount,
		TotalQuestions: quizData.TotalQuestions,
		CompletedAt:    time.Now(),
	}

	if err := s.repository.SaveQuizAttemptWithAnswers(ctx, attempt, attemptAnswers); err != nil {
		return nil, err
	}

	s.streakSvc.CheckAndUpdateStreak(ctx, userId)

	return &domain.SolveQuizResponse{
		AttemptID:      attemptID,
		Score:          correctCount,
		TotalQuestions: quizData.TotalQuestions,
		Percentage:     percentage,
		Answers:        answerFeedbacks,
	}, nil
}

func (s *QuizService) GetQuizzes(ctx context.Context, userId uuid.UUID) (*domain.GetQuizzesResponse, error) {
	resp, err := s.repository.GetQuizzesByUserID(ctx, userId)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		empty := domain.GetQuizzesResponse([]domain.QuizResponseDTO{})
		return &empty, nil
	}
	return resp, nil
}

func (s *QuizService) GetQuizById(ctx context.Context, req *domain.GetQuizByIdRequest) (*domain.GetQuizByIdResponse, error) {
	return s.repository.GetQuizById(ctx, req.QuizID)
}

func (s *QuizService) GetAttempts(ctx context.Context, userId uuid.UUID, req *domain.GetAttemptsRequest) (*domain.GetAttemptsResponse, error) {
	attempts, count, err := s.repository.GetQuizAttemptsByUserId(ctx, userId, req.Limit, req.Offset)
	if err != nil {
		return nil, err
	}

	if attempts == nil {
		attempts = make([]domain.QuizAttemptDetail, 0)
	}

	return &domain.GetAttemptsResponse{
		Attempts: attempts,
		Count:    count,
	}, nil
}

func (s *QuizService) GetAttemptById(ctx context.Context, req *domain.GetAttemptByIdRequest) (*domain.AttemptDetailResponse, error) {
	attempt, err := s.repository.GetQuizAttemptById(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	answers, err := s.repository.GetAttemptAnswers(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	if answers == nil {
		answers = make([]domain.AttemptAnswerDetail, 0)
	}

	return &domain.AttemptDetailResponse{
		ID:             attempt.ID,
		QuizID:         attempt.QuizID,
		QuizTitle:      attempt.QuizTitle,
		Score:          attempt.Score,
		TotalQuestions: attempt.TotalQuestions,
		Percentage:     attempt.Percentage,
		CompletedAt:    attempt.CompletedAt,
		Answers:        answers,
	}, nil
}
