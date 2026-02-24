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

// HandleGetUsageStatus returns the current usage status for the authenticated user.
func (h *SubscriptionHandler) HandleGetUsageStatus(ctx context.Context, req *domain.GetUsageStatusRequest) (*domain.UsageStatusResponse, int, error) {
	userID := domain.GetUserID(ctx)

	status, err := h.subscriptionService.GetUsageStatus(ctx, userID)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	return status, http.StatusOK, nil
}

// HandleWebhook returns a Fiber handler for RevenueCat webhook events.
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
