package services

import (
	"context"
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

	var wg sync.WaitGroup
	resultChan := make(chan questionResult, len(chunks))

	for i, chunk := range chunks {
		wg.Add(1)
		go func(idx int, c string) {
			defer wg.Done()

			question, err := s.aiService.GenerateQuizQuestion(ctx, c)
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

	results := make([]questionResult, len(chunks))
	for result := range resultChan {
		results[result.index] = result
	}

	for _, result := range results {
		if result.err != nil {
			return nil, result.err
		}
		quiz.AddQuestion(result.question)
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
