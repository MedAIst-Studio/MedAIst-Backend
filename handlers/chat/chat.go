package chat

import (
	"context"
	"errors"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

type ChatConfig struct {
	ChatService domain.ChatService
}

type ChatHandler struct {
	chatService domain.ChatService
}

func NewChatHandler(config *ChatConfig) *ChatHandler {
	return &ChatHandler{
		chatService: config.ChatService,
	}
}

// godoc
// Handle processes a GetChatsRequest and returns a GetChatsResponse.
//
//	@Summary		Retrieve chats for the authenticated user
//	@Description	Retrieves a list of chats associated with the authenticated user.
//	@Tags			Chats
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	domain.GetChatsResponse
//	@Failure		500
//	@Security		BearerAuth
//	@Router			/chats [get]
func (h *ChatHandler) HandleGetChats(ctx context.Context, req *domain.GetChatsRequest) (*domain.GetChatsResponse, int, error) {
	userId := domain.GetUserID(ctx)
	chats, err := h.chatService.GetChatsByUserID(ctx, userId)
	if err != nil {
		return nil, 500, err
	}

	response := domain.GetChatsResponse(chats)
	return &response, 200, nil
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
//	@Success		200	{object}	domain.GetChatByIDResponse
//	@Failure		404
//	@Failure		500
//	@Security		BearerAuth
//	@Router			/chats/{id} [get]
func (h *ChatHandler) HandleGetChatByID(ctx context.Context, req *domain.GetChatByIDRequest) (*domain.GetChatByIDResponse, int, error) {
	chat, err := h.chatService.GetChatByID(ctx, req.ChatID)
	if err != nil {
		if errors.Is(err, domain.ErrChatNotFound) {
			return nil, 404, err
		}
		return nil, 500, err
	}
	return chat, 200, nil
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
//
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
func (h *ChatHandler) HandleUpdateChat(ctx context.Context, req *domain.UpdateChatDTO) (*domain.UpdateChatResponse, int, error) {
	if req.Title == "" {
		return nil, 400, errors.New("title cannot be empty")
	}

	err := h.chatService.UpdateChat(ctx, req.Id, req.Title)
	if err != nil {
		if errors.Is(err, domain.ErrChatNotFound) {
			return nil, 404, err
		}
		return nil, 500, err
	}

	return nil, 204, nil
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
func (h *ChatHandler) HandleDeleteChat(ctx context.Context, req *domain.DeleteChatRequest) (*domain.DeleteChatResponse, int, error) {
	err := h.chatService.DeleteChat(ctx, req.Id)
	if err != nil {
		if errors.Is(err, domain.ErrChatNotFound) {
			return nil, 404, err
		}
		return nil, 500, err
	}
	return nil, 204, nil
}
