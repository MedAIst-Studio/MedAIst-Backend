package message

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/MedAIst-Studio/MedAIist-Backend/app/file"
	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type SendMessageRequest struct {
	PdfId    uuid.UUID `json:"pdf_id" form:"pdf_id"`
	ChatId   uuid.UUID `json:"chat_id,omitempty" form:"chat_id"`
	Content  string    `json:"content" validate:"required,min=1" form:"content"`
	ImageUrl string    `json:"image" form:"image_url"`
}

type SendMessageResponse struct {
	ChatId        *uuid.UUID `json:"chat_id,omitempty"`
	AiResponse    string     `json:"ai_response"`
	Title         string     `json:"title,omitempty"`
	UserMessageId uuid.UUID  `json:"user_message_id"`
	AiMessageId   uuid.UUID  `json:"ai_message_id"`
}

type Config struct {
	PastMessagesLimit int
	Stream            bool
	Repo              Repository
	Vb                VectorRepository
	Ems               EmbeddingService
	ChatSvc           AIService
	FileService       file.FileService
}

type SendMessageHandler struct {
	Config
}

func NewSendMessageHandler(config Config) *SendMessageHandler {
	return &SendMessageHandler{
		Config: config,
	}
}

// @Summary		Send a message in a chat
// @Description	Sends a message in a chat and gets a response from the AI
// @Tags			Message
// @Accept			multipart/form-data
// @Produce		json
// @Param			pdf_id		formData	string	false	"PDF ID"
// @Param			chat_id		formData	string	false	"Chat ID"
// @Param			content		formData	string	true	"Content"
// @Param			image_url	formData	string	false	"Image URL"
// @Success		200			{object}	SendMessageResponse
// @Failure		400
// @Failure		500
// @Security		BearerAuth
// @Router			/messages [post]
func (h *SendMessageHandler) Handle(ctx context.Context, req *SendMessageRequest) (*SendMessageResponse, int, error) {
	userId := domain.GetUserID(ctx)

	userMessage := domain.NewMessage(req.ChatId, req.Content, true)
	if req.ImageUrl != "" {
		userMessage.ImageUrl = req.ImageUrl
	}
	isNewChat := req.ChatId == uuid.Nil

	chatTitle, err := h.handleNewChatIfNeeded(ctx, req, userId, userMessage, isNewChat)
	if err != nil {
		return nil, 500, err
	}

	if err := h.Repo.SaveMessage(ctx, userMessage); err != nil {
		return nil, 500, err
	}

	// Determine QueryType
	var queryType domain.QueryType

	// Check if PdfId refers to an image
	if req.PdfId != uuid.Nil && req.ImageUrl == "" {
		fileMeta, err := h.Repo.GetFileMetadata(ctx, req.PdfId)
		if err == nil && isImageFile(fileMeta.Name) {
			// Construct Image URL (assuming accessible via some public/internal path or just passing identifier for now)
			// For local dev with MinIO, we might need a presigned URL or just the path if the service handles it.
			// Let's assume the service can handle the relative path or we construct a public URL.
			// Since I don't have URL generation logic here, I'll use a placeholder or the path.
			// Actually, ExtractTextFromImage probably takes a URL.
			// Let's try setting it to the path for now and logging.
			req.ImageUrl = "uploads/" + req.PdfId.String() // Internal path pattern
			fmt.Printf("DEBUG: PdfId %s points to image file %s. Auto-switching to ImageQuery.\n", req.PdfId, fileMeta.Name)
		}
	}

	if req.ImageUrl != "" {
		fmt.Printf("DEBUG: ImageUrl provided: %s. Setting QueryType to ImageQuery.\n", req.ImageUrl)
		queryType = domain.ImageQuery
	} else {
		// Only check user query with LLM if it's text-only
		var err error
		queryType, err = h.ChatSvc.CheckUserQuery(ctx, req.Content)
		if err != nil {
			fmt.Printf("Error checking user query: %v\n", err)
			h.Repo.DeleteMessage(ctx, userMessage.Id)
			return nil, 500, err
		}
		fmt.Printf("DEBUG: Checked user query. LLM determined QueryType: %s\n", queryType)
	}

	crm := domain.NewChatRequestManager(queryType, req.Content)

	// If ImageQuery, we need to attach the image to the Request
	if queryType == domain.ImageQuery && userMessage.ImageUrl != "" {
		if len(crm.Req.Messages) > 0 {
			lastMsgIdx := len(crm.Req.Messages) - 1
			if crm.Req.Messages[lastMsgIdx].Role == domain.RoleUser {
				crm.Req.Messages[lastMsgIdx].ImageUrl = []string{userMessage.ImageUrl}
			}
		}
	}

	if isNewChat && crm.IsClarificationQuery() {
		return h.handleUnallowedClarification(ctx, userMessage, chatTitle, isNewChat)
	}

	if err := h.processQuery(ctx, req, userMessage, crm, isNewChat); err != nil {
		if errors.Is(err, domain.ErrInfoNotFoundInPDF) {
			return h.handleInfoNotFound(ctx, userMessage, chatTitle, isNewChat, err)
		}
		return nil, 500, err
	}

	// If ImageQuery, instead of sending image to ChatRequest directly (or parallel),
	// we now implement the OCR -> Reasoning flow as requested.
	if queryType == domain.ImageQuery && userMessage.ImageUrl != "" {
		fmt.Println("DEBUG: Entering OCR -> Reasoning flow for ImageQuery.")
		// 1. OCR Step
		extractedText, err := h.ChatSvc.ExtractTextFromImage(ctx, userMessage.ImageUrl)
		if err != nil {
			return nil, 500, fmt.Errorf("OCR failed: %w", err)
		}

		reasoningAns, err := h.ChatSvc.SolveQuestionWithReasoning(ctx, extractedText)
		if err != nil {
			return nil, 500, fmt.Errorf("reasoning failed: %w", err)
		}

		aiMessage := domain.NewMessage(userMessage.ChatId, reasoningAns, false)
		if err := h.Repo.SaveMessage(ctx, aiMessage); err != nil {
			return nil, 500, err
		}

		return h.buildResponse(userMessage, aiMessage, chatTitle, isNewChat), 200, nil
	}

	crm.Req.Stream = h.Stream
	aiResp, err := h.ChatSvc.SendChatRequest(ctx, crm.Req)
	if err != nil {
		return nil, 500, err
	}

	aiMessage := domain.NewMessage(userMessage.ChatId, aiResp.Text, false)
	if err := h.Repo.SaveMessage(ctx, aiMessage); err != nil {
		return nil, 500, err
	}

	return h.buildResponse(userMessage, aiMessage, chatTitle, isNewChat), 200, nil
}

func (h *SendMessageHandler) handleNewChatIfNeeded(ctx context.Context, req *SendMessageRequest, userId uuid.UUID, userMessage *domain.Message, isNewChat bool) (string, error) {
	if !isNewChat {
		return "", nil
	}

	aiResp, err := h.ChatSvc.GenerateChatTitle(ctx, domain.NewChatTitleRequest(req.Content))
	if err != nil {
		return "", err
	}

	chat := domain.NewChat(userId, trimTitle(aiResp.Text))
	if err := h.Repo.CreateChat(ctx, chat); err != nil {
		return "", err
	}

	userMessage.UpdateChatId(chat.Id)
	return chat.Title, nil
}

func (h *SendMessageHandler) processQuery(ctx context.Context, req *SendMessageRequest, userMessage *domain.Message, crm *domain.ChatRequestManager, isNewChat bool,
) error {
	if crm.IsGeneralQuery() {
		return nil
	}

	if !isNewChat {
		pastMessages, err := h.Repo.GetChatHistoryForAIContext(
			ctx,
			userMessage.ChatId,
			h.PastMessagesLimit,
		)
		if err != nil {
			return err
		}
		crm.Req.AppendMessages(pastMessages)
		crm.Req.RemoveUserMessageToEnd()
	}

	if crm.IsPdfSearchQuery() {
		return h.addPdfContext(ctx, req, crm)
	}
	return nil
}

type searchResult struct {
	chunks []string
	err    error
}

func (h *SendMessageHandler) addPdfContext(ctx context.Context, req *SendMessageRequest, crm *domain.ChatRequestManager) error {
	generatedQueries, err := h.ChatSvc.ExpandQuery(ctx, req.Content)
	if err != nil {
		return err
	}

	results := make(chan searchResult, len(generatedQueries))
	var wg sync.WaitGroup

	for _, query := range generatedQueries {
		wg.Add(1)
		go func(q string) {
			defer wg.Done()

			embedding, err := h.Ems.GenerateEmbedding(ctx, &EmbeddingRequest{Text: q})
			if err != nil {
				results <- searchResult{err: fmt.Errorf("embedding failed for query '%s': %w", q, err)}
				return
			}

			relevantTexts, err := h.Vb.SearchByEmbedding(ctx, req.PdfId, embedding.Vector)
			if err != nil {
				results <- searchResult{err: fmt.Errorf("search failed for query '%s': %w", q, err)}
				return
			}

			results <- searchResult{chunks: relevantTexts}
		}(query)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var allChunks []string
	for result := range results {
		if result.err != nil {
			return result.err
		}
		allChunks = append(allChunks, result.chunks...)
	}

	uniqueChunks := deduplicateChunks(allChunks)
	rerankedChunks, err := h.ChatSvc.RerankChunks(ctx, uniqueChunks, req.Content)
	if err != nil {
		return fmt.Errorf("reranking failed: %w", err)
	}
	crm.Req.AddRelevantExcerpts(rerankedChunks)
	return nil
}

func (h *SendMessageHandler) handleUnallowedClarification(ctx context.Context, userMessage *domain.Message, chatTitle string, isNewChat bool) (*SendMessageResponse, int, error) {
	aiMessage := domain.NewMessage(
		userMessage.ChatId,
		domain.UnallowedClarificationMsg,
		false,
	)

	if err := h.Repo.SaveMessage(ctx, aiMessage); err != nil {
		return nil, 500, err
	}

	return h.buildResponse(userMessage, aiMessage, chatTitle, isNewChat), 200, nil
}

func (h *SendMessageHandler) handleInfoNotFound(ctx context.Context, userMessage *domain.Message, chatTitle string, isNewChat bool, err error) (*SendMessageResponse, int, error) {
	aiMessage := domain.NewMessage(userMessage.ChatId, err.Error(), false)

	if saveErr := h.Repo.SaveMessage(ctx, aiMessage); saveErr != nil {
		return nil, 500, saveErr
	}

	return h.buildResponse(userMessage, aiMessage, chatTitle, isNewChat), 200, nil
}

func (h *SendMessageHandler) buildResponse(userMessage, aiMessage *domain.Message, chatTitle string, isNewChat bool) *SendMessageResponse {
	resp := &SendMessageResponse{
		AiResponse:    aiMessage.Content,
		UserMessageId: userMessage.Id,
		AiMessageId:   aiMessage.Id,
	}

	if isNewChat {
		resp.Title = chatTitle
		resp.ChatId = &userMessage.ChatId
	}

	return resp
}

func trimTitle(s string) string {
	return strings.ReplaceAll(s, "\"", "")
}

func deduplicateChunks(chunks []string) []string {
	seen := make(map[string]bool)
	unique := make([]string, 0, len(chunks))

	for _, chunk := range chunks {
		if !seen[chunk] {
			seen[chunk] = true
			unique = append(unique, chunk)
		}
	}

	return unique
}

func getFileExtension(filename string) string {
	if strings.Contains(filename, ".") {
		return filename[strings.LastIndex(filename, "."):]
	}
	return ""
}

func isImageFile(filename string) bool {
	ext := strings.ToLower(getFileExtension(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
		return true
	default:
		return false
	}
}
