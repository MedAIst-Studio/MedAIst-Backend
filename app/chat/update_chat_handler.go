package chat

import (
	"context"
	"errors"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type UpdateChatHandlerRequest struct {
	Id    uuid.UUID `params:"id"`
	Title string    `json:"title"`
}

type UpdateChatHandler struct {
	repo Repository
}

func NewUpdateChatHandler(repo Repository) *UpdateChatHandler {
	return &UpdateChatHandler{repo: repo}
}

// godoc
//
//	@Summary		Update Chat Title
//	@Description	Updates the title of a chat by its ID.
//	@Tags			Chats
//	@Accept			json
//	@Produce		json
//	@Param			id		path	string						true	"Chat ID"
//	@Param			body	body	domain.UpdateChatRequest	true	"New Title"
//	@Success		204		"No Content"
//	@Failure		400
//	@Failure		404
//	@Failure		500
//	@Security		BearerAuth
//	@Router			/chats/{id} [patch]
func (h *UpdateChatHandler) Handle(ctx context.Context, req *UpdateChatHandlerRequest) (*domain.UpdateChatResponse, int, error) {
	if req.Title == "" {
		return nil, 400, errors.New("title cannot be empty")
	}

	err := h.repo.UpdateChat(ctx, req.Id, req.Title)
	if err != nil {
		if errors.Is(err, domain.ErrChatNotFound) {
			return nil, 404, err
		}
		return nil, 500, err
	}

	return nil, 204, nil
}
