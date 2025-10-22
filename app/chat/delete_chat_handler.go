package chat

import (
	"context"
	"errors"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type DeleteChatRequest struct {
	Id uuid.UUID `params:"id"`
}

type DeleteChatResponse struct {
}

type DeleteChatHandler struct {
	repo Repository
}

func NewDeleteChatHandler(repo Repository) *DeleteChatHandler {
	return &DeleteChatHandler{repo: repo}
}

// godoc
//
//	@Summary		Delete Chat
//	@Description	Deletes a chat by its ID.
//	@Tags			Chats
//	@Accept			json
//	@Produce		json
//	@Param			id	path	string	true	"Chat ID"
//	@Success		204	"No Content"
//	@Failure		404
//	@Failure		500
//	@Security		BearerAuth
//	@Router			/chats/{id} [delete]
func (h *DeleteChatHandler) Handle(ctx context.Context, req *DeleteChatRequest) (*DeleteChatResponse, int, error) {
	err := h.repo.DeleteChat(ctx, req.Id)
	if err != nil {
		if errors.Is(err, domain.ErrChatNotFound) {
			return nil, 404, err
		}
		return nil, 500, err
	}
	return nil, 204, nil
}
