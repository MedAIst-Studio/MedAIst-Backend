package chat

import (
	"context"
	"errors"
	"time"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type GetChatByIDRequest struct {
	ChatID uuid.UUID `params:"id"`
}

type GetChatByIDResponse []Message

type Message struct {
	IsUser          bool                    `json:"is_user"`
	Content         string                  `json:"content"`
	Timestamp       time.Time               `json:"timestamp"`
	ContextSnapshot *domain.ContextSnapshot `json:"context_snapshot,omitempty"`
	Attachments     []domain.Attachment     `json:"attachments,omitempty"`
}

type GetChatByIDHandler struct {
	repo Repository
}

func NewGetChatByIDHandler(repo Repository) *GetChatByIDHandler {
	return &GetChatByIDHandler{repo: repo}
}

// godoc
// Handle processes a GetChatByIDRequest and returns a GetChatByIDResponse.
//
//	@Summary		Retrieve messages for a specific chat
//	@Description	Retrieves all messages associated with a specific chat by its ID.
//	@Tags			Chats
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Chat ID"
//	@Success		200	{object}	GetChatByIDResponse
//	@Failure		404
//	@Failure		500
//	@Security		BearerAuth
//	@Router			/chats/{id} [get]
func (h *GetChatByIDHandler) Handle(ctx context.Context, req *GetChatByIDRequest) (*GetChatByIDResponse, int, error) {
	chat, err := h.repo.GetChatByID(ctx, req.ChatID)
	if err != nil {
		if errors.Is(err, domain.ErrChatNotFound) {
			return nil, 404, err
		}
		return nil, 500, err
	}
	return chat, 200, nil
}
