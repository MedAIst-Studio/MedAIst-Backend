package subscription

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/gofiber/fiber/v2"
)

type SubscriptionConfig struct {
	SubscriptionService domain.SubscriptionService
}

type SubscriptionHandler struct {
	subscriptionService domain.SubscriptionService
}

func NewSubscriptionHandler(config *SubscriptionConfig) *SubscriptionHandler {
	return &SubscriptionHandler{
		subscriptionService: config.SubscriptionService,
	}
}

// @Summary		Get Usage Status
// @Description	Returns the current user's usage limits and remaining quota for messages, OCR, PDF chat, files, and quizzes
// @Tags			Subscription
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Success		200	{object}	domain.UsageStatusResponse
// @Failure		401
// @Failure		500
// @Router			/users/usage [get]
func (h *SubscriptionHandler) HandleGetUsageStatus(ctx context.Context, req *domain.GetUsageStatusRequest) (*domain.UsageStatusResponse, int, error) {
	userID := domain.GetUserID(ctx)

	status, err := h.subscriptionService.GetUsageStatus(ctx, userID)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	return status, http.StatusOK, nil
}

// @Summary		RevenueCat Webhook
// @Description	Handles incoming RevenueCat webhook events (INITIAL_PURCHASE, RENEWAL, CANCELLATION, EXPIRATION, etc.)
// @Tags			Webhook
// @Accept			json
// @Produce		json
// @Param			request	body	domain.RevenueCatWebhookEvent	true	"RevenueCat Webhook Event"
// @Success		200
// @Failure		400
// @Failure		401
// @Failure		500
// @Router			/webhooks/revenuecat [post]
func (h *SubscriptionHandler) HandleWebhook() fiber.Handler {
	return func(c *fiber.Ctx) error {
		expectedSecret := os.Getenv("REVENUECAT_WEBHOOK_SECRET")
		authHeader := c.Get("Authorization")

		if expectedSecret != "" && authHeader != "Bearer "+expectedSecret {
			return c.Status(fiber.StatusUnauthorized).JSON(domain.Error{
				Message: domain.ErrInvalidWebhookSecret.Error(),
				Code:    fiber.StatusUnauthorized,
			})
		}

		var event domain.RevenueCatWebhookEvent
		if err := json.Unmarshal(c.Body(), &event); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(domain.Error{
				Message: domain.ErrInvalidRequest.Error(),
				Code:    fiber.StatusBadRequest,
			})
		}

		if err := h.subscriptionService.HandleWebhookEvent(c.UserContext(), &event); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(domain.Error{
				Message: err.Error(),
				Code:    fiber.StatusInternalServerError,
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
	}
}
