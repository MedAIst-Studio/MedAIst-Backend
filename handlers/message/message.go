package message

import (
	"context"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

type MessageConfig struct {
	MessageService domain.MessageService
}

type MessageHandler struct {
	messageService domain.MessageService
}

func NewMessageHandler(config *MessageConfig) *MessageHandler {
	return &MessageHandler{
		messageService: config.MessageService,
	}
}

// @Summary		Send a message in a chat
// @Description	Sends a message in a chat and gets a response from the AI
// @Tags			Message
// @Accept			json
// @Produce		json
// @Param			request	body		domain.SendMessageRequest	true	"Send Message Request"
// @Success		200		{object}	domain.SendMessageResponse
// @Failure		400
// @Failure		500
// @Security		BearerAuth
// @Router			/messages [post]
func (h *MessageHandler) HandleSendMessage(ctx context.Context, req *domain.SendMessageRequest) (*domain.SendMessageResponse, int, error) {
	userId := domain.GetUserID(ctx)

	resp, err := h.messageService.SendMessage(ctx, userId, req)
	if err != nil {
		if err == domain.ErrInfoNotFoundInPDF {
			// Based on previous code, handleInfoNotFound returns a 200 with AI stating info not found.
			// Currently this is handled by the service, so we don't need a special check here if it returns resp directly.
			return resp, 200, nil
		}
		// If the service just returns the standard error we might map it.
		// For now we map err to 500
		return nil, 500, err
	}

	return resp, 200, nil
}
