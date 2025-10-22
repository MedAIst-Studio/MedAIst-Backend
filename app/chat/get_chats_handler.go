package chat

import (
	"context"
	"time"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

type GetChatsRequest struct {
}

type GetChatsResponse []Chat

type Chat struct {
	Id        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type GetChatsHandler struct {
	repo Repository
}

func NewGetChatsHandler(repo Repository) *GetChatsHandler {
	return &GetChatsHandler{repo: repo}
}

// godoc
// Handle processes a GetChatsRequest and returns a GetChatsResponse.
//
//	@Summary		Retrieve chats for the authenticated user
//	@Description	Retrieves a list of chats associated with the authenticated user.
//	@Tags			Chats
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	GetChatsResponse
//	@Failure		500
//	@Security		BearerAuth
//	@Router			/chats [get]
func (h *GetChatsHandler) Handle(ctx context.Context, req *GetChatsRequest) (*GetChatsResponse, int, error) {
	userId := domain.GetUserID(ctx)
	chats, err := h.repo.GetChatsByUserID(ctx, userId)
	if err != nil {
		return nil, 500, err
	}

	response := GetChatsResponse(chats)
	return &response, 200, nil
}
