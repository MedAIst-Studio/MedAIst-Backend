package fiber

import (
	"context"
	"strings"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/gofiber/fiber/v2"
)

type MiddlewareManager struct {
	tokenService        domain.TokenService
	logger              domain.Logger
	subscriptionService domain.SubscriptionService
}

func NewMiddlewareManager(tokenService domain.TokenService, logger domain.Logger, subscriptionService domain.SubscriptionService) *MiddlewareManager {
	return &MiddlewareManager{
		tokenService:        tokenService,
		logger:              logger,
		subscriptionService: subscriptionService,
	}
}

func (m *MiddlewareManager) AuthMiddleware(c *fiber.Ctx) error {
	c.Locals("requireAuth", true)
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(domain.Error{
			Message: domain.ErrMissingAuthHeader.Error(),
			Code:    fiber.StatusUnauthorized,
		})
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return c.Status(fiber.StatusUnauthorized).JSON(domain.Error{
			Message: domain.ErrInvalidAuthHeader.Error(),
			Code:    fiber.StatusUnauthorized,
		})
	}

	token := parts[1]
	payload, err := m.tokenService.ValidateAuthAccessToken(token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(domain.Error{
			Message: err.Error(),
			Code:    fiber.StatusUnauthorized,
		})
	}
	c.Locals("userID", payload.UserID)
	return c.Next()
}

type FiberContextKey struct{}

func contextMiddleware(c *fiber.Ctx) error {
	ctx := context.WithValue(c.UserContext(), FiberContextKey{}, c)
	c.SetUserContext(ctx)
	return c.Next()
}
