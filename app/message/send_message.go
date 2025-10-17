package message

import (
	"context"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type SendMessageRequest struct {
	PdfId   uuid.UUID `json:"pdf_id"`
	ChatId  uuid.UUID `json:"chat_id"`
	Content string    `json:"content"`
}

type SendMessageResponse struct {
	ChatId uuid.UUID `json:"chat_id"`
}

type SendMessageHandler struct {
	PastMessagesLimit int  `json:"past_messages_limit"`
	Stream            bool `json:"stream"`
	Repo              Repository
	Vb                VectorDB
	Ems               EmbeddingService
	ChatSvc           ChatService
}

func NewSendMessageHandler(repo Repository, vb VectorDB, ems EmbeddingService, chatSvc ChatService) *SendMessageHandler {
	return &SendMessageHandler{
		Repo:    repo,
		Vb:      vb,
		Ems:     ems,
		ChatSvc: chatSvc,
	}
}

// Handle godoc
// @Summary Send a message in a chat
// @Description Sends a message in a chat and gets a response from the AI
// @Tags Message
// @Accept json
// @Produce json
// @Param request body SendMessageRequest true "Send Message Request"
// @Success 200 {object} SendMessageResponse
// @Failure 400
// @Failure 500
// @Router /messages [post]
func (h *SendMessageHandler) Handle(ctx context.Context, req *SendMessageRequest) (*SendMessageResponse, int, error) {
	userId := domain.GetUserID(ctx)

	userMessage := domain.NewMessage(req.ChatId, req.Content, true)

	isSubsequentMessage := true
	if req.ChatId == uuid.Nil {
		isSubsequentMessage = false
		// ai request for chat title
		chat := domain.NewChat(userId, req.PdfId)
		err := h.Repo.CreateChat(ctx, chat)
		if err != nil {
			return nil, 500, err
		}
		userMessage.UpdateChatId(chat.Id)
	}

	err := h.Repo.SaveMessage(ctx, userMessage)
	if err != nil {
		return nil, 500, err
	}

	embedding, err := h.Ems.ConvertToEmbedding(ctx, req.Content)
	if err != nil {
		return nil, 500, err
	}

	relevantText, err := h.Vb.SearchTextByEmbedding(ctx, req.PdfId, embedding)
	if err != nil {
		return nil, 500, err
	}

	aiReq := domain.NewThirdPartyAIRequest(h.Stream, relevantText, req.Content)

	if isSubsequentMessage {
		pastMessages, err := h.Repo.GetPastMessages(ctx, userMessage.ChatId, h.PastMessagesLimit)
		if err != nil {
			return nil, 500, err
		}
		aiReq.AppendPastMessages(pastMessages)
	}

	return &SendMessageResponse{
		ChatId: userMessage.ChatId,
	}, 200, nil
}
