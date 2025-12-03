package message

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type SendMessageRequest struct {
	PdfId   uuid.UUID `json:"pdf_id" validate:"required"`
	ChatId  uuid.UUID `json:"chat_id,omitempty"`
	Content string    `json:"content" validate:"required,min=1"`
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
// @Accept			json
// @Produce		json
// @Param			request	body		SendMessageRequest	true	"Send Message Request"
// @Success		200		{object}	SendMessageResponse
// @Failure		400
// @Failure		500
// @Security		BearerAuth
// @Router			/messages [post]
func (h *SendMessageHandler) Handle(ctx context.Context, req *SendMessageRequest) (*SendMessageResponse, int, error) {
	userId := domain.GetUserID(ctx)

	userMessage := domain.NewMessage(req.ChatId, req.Content, true)
	isNewChat := req.ChatId == uuid.Nil

	chatTitle, err := h.handleNewChatIfNeeded(ctx, req, userId, userMessage, isNewChat)
	if err != nil {
		return nil, 500, err
	}

	if err := h.Repo.SaveMessage(ctx, userMessage); err != nil {
		return nil, 500, err
	}

	queryType, err := h.ChatSvc.CheckUserQuery(ctx, req.Content)
	if err != nil {
		fmt.Printf("Error checking user query: %v\n", err)
		h.Repo.DeleteMessage(ctx, userMessage.Id)
		return nil, 500, err
	}
	fmt.Printf("queryType: %v\n", queryType)

	crm := domain.NewChatRequestManager(queryType, req.Content)

	if isNewChat && crm.IsClarificationQuery() {
		return h.handleUnallowedClarification(ctx, userMessage, chatTitle, isNewChat)
	}

	if err := h.processQuery(ctx, req, userMessage, crm, isNewChat); err != nil {
		if errors.Is(err, domain.ErrInfoNotFoundInPDF) {
			return h.handleInfoNotFound(ctx, userMessage, chatTitle, isNewChat, err)
		}
		return nil, 500, err
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

	chat := domain.NewChat(userId, req.PdfId, trimTitle(aiResp.Text))
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
