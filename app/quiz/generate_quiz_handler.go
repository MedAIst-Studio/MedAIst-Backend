package quiz

import (
	"context"
	"fmt"
	"sync"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type GenerateQuizRequest struct {
	FileId    uuid.UUID `json:"fileId" validate:"required,uuid"`
	FileName  string    `json:"fileName" validate:"required"`
	StartPage int       `json:"startPage,omitempty"`
	EndPage   int       `json:"endPage,omitempty"`
}

type GenerateQuizResponse struct {
}

type GenerateQuizHandler struct {
	aiService    AIService
	repository   Repository
	chunkService ChunkService
}

func NewGenerateQuizHandler(aiService AIService, repository Repository, chunkService ChunkService) *GenerateQuizHandler {
	return &GenerateQuizHandler{
		aiService:    aiService,
		repository:   repository,
		chunkService: chunkService,
	}
}

type questionResult struct {
	question *domain.QuizQuestion
	index    int
	err      error
}

// @Summary     Generate Quiz
// @Description Generate a quiz based on the provided file
// @Tags        Quiz
// @Accept      json
// @Produce     json
// @Param       request body GenerateQuizRequest true "Generate Quiz request"
// @Success     200 {object} GenerateQuizResponse
// @Failure     400
// @Failure     500
// @Router      /quizes [post]
func (h *GenerateQuizHandler) Handle(ctx context.Context, req *GenerateQuizRequest) (*GenerateQuizResponse, int, error) {
	userId := domain.GetUserID(ctx)
	quiz := domain.NewQuiz(userId, req.FileId, req.FileName)
	fmt.Println(req)
	var chunks []string
	var err error

	pageRange := []int{
		req.StartPage,
		req.EndPage,
	}

	if len(pageRange) > 0 {
		fmt.Println("has page range")
		if len(pageRange) != 2 {
			return nil, 400, domain.ErrInvalidRequest
		}
		if chunks, err = h.chunkService.GetChunksByPages(ctx, req.FileId, pageRange); err != nil {
			return nil, 500, err
		}
	} else {
		if chunks, err = h.chunkService.GetChunks(ctx, req.FileId); err != nil {
			return nil, 500, err
		}
	}

	fmt.Println("Chunks:", chunks)

	var wg sync.WaitGroup
	resultChan := make(chan questionResult, len(chunks))

	for i, chunk := range chunks {
		wg.Add(1)
		go func(idx int, c string) {
			defer wg.Done()

			question, err := h.aiService.GenerateQuizQuestion(ctx, c)
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
			return nil, 500, result.err
		}
		quiz.AddQuestion(result.question)
	}
	fmt.Println(quiz)
	err = h.repository.SaveQuiz(ctx, quiz)
	if err != nil {
		return nil, 500, err
	}

	return nil, 204, nil
}
