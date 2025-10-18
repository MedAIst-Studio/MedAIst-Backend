package message

import (
	"context"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type SendMessageRequest struct {
	PdfId   uuid.UUID `json:"pdf_id"`
	ChatId  uuid.UUID `json:"chat_id,omitempty"`
	Content string    `json:"content"`
}

type SendMessageResponse struct {
	ChatId        uuid.UUID `json:"chat_id,omitempty"`
	AiResponse    string    `json:"ai_response"`
	Title         string    `json:"title,omitempty"`
	UserMessageId uuid.UUID `json:"user_message_id"`
	AiMessageId   uuid.UUID `json:"ai_message_id"`
}

type Config struct {
	PastMessagesLimit int  `json:"past_messages_limit"`
	Stream            bool `json:"stream"`
	Repo              Repository
	Vb                VectorRepository
	Ems               EmbeddingService
	ChatSvc           AIService
}

type SendMessageHandler struct {
	Config `json:"config"`
}

func NewSendMessageHandler(config Config) *SendMessageHandler {
	return &SendMessageHandler{
		Config: config,
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

	var chatTitle string
	if req.ChatId == uuid.Nil {
		isSubsequentMessage = false

		aiResp, err := h.ChatSvc.GenerateChatTitle(ctx, domain.NewChatTitleRequest(req.Content))
		if err != nil {
			return nil, 500, err
		}

		chat := domain.NewChat(userId, req.PdfId, aiResp.Text)

		err = h.Repo.CreateChat(ctx, chat)
		if err != nil {
			return nil, 500, err
		}

		chatTitle = aiResp.Text
		userMessage.UpdateChatId(chat.Id)
	}

	err := h.Repo.SaveMessage(ctx, userMessage)
	if err != nil {
		return nil, 500, err
	}

	embedding, err := h.Ems.GenerateEmbedding(ctx, req.Content)
	if err != nil {
		return nil, 500, err
	}

	relevantText, err := h.Vb.SearchByEmbedding(ctx, req.PdfId, embedding)
	if err != nil {
		return nil, 500, err
	}

	aiReq := domain.NewChatRequest(h.Stream, relevantText, req.Content)

	if isSubsequentMessage {
		pastMessages, err := h.Repo.GetChatHistory(ctx, userMessage.ChatId, h.PastMessagesLimit)
		if err != nil {
			return nil, 500, err
		}
		aiReq.AppendMessages(pastMessages)
	}

	aiResp, err := h.ChatSvc.SendChatRequest(ctx, aiReq)
	if err != nil {
		return nil, 500, err
	}

	aiMessage := domain.NewMessage(userMessage.ChatId, aiResp.Text, false)
	err = h.Repo.SaveMessage(ctx, aiMessage)
	if err != nil {
		return nil, 500, err
	}

	return &SendMessageResponse{
		ChatId:        userMessage.ChatId,
		AiResponse:    aiResp.Text,
		Title:         chatTitle,
		UserMessageId: userMessage.Id,
		AiMessageId:   aiMessage.Id,
	}, 200, nil

}
