package fiber

import (
	"encoding/json"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// UsageLimitMiddleware checks if the user has remaining usage for the given action type.
// Should be placed AFTER AuthMiddleware in the middleware chain.
func (m *MiddlewareManager) UsageLimitMiddleware(actionType domain.ActionType) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := c.Locals("userID").(string)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(domain.Error{
				Message: domain.ErrUnauthorized.Error(),
				Code:    fiber.StatusUnauthorized,
			})
		}

		uid := domain.GetUserIDFromString(userID)

		allowed, err := m.subscriptionService.CheckUsageLimit(c.UserContext(), uid, actionType)
		if err != nil {
			m.logger.Error("Kullanım limiti kontrol hatası",
				"error", err.Error(),
				"userID", userID,
				"action", string(actionType),
			)
			return c.Status(fiber.StatusInternalServerError).JSON(domain.Error{
				Message: domain.ErrInternalServer.Error(),
				Code:    fiber.StatusInternalServerError,
			})
		}

		if !allowed {
			errMsg := getUsageLimitError(actionType)
			return c.Status(fiber.StatusForbidden).JSON(domain.Error{
				Message: errMsg.Error(),
				Code:    fiber.StatusForbidden,
			})
		}

		// Proceed to handler
		err = c.Next()

		// Only increment usage for action-based limits (not file count).
		// File count is tracked by the files table itself.
		if err == nil && c.Response().StatusCode() >= 200 && c.Response().StatusCode() < 300 {
			if actionType != domain.ActionFileUpload {
				_ = m.subscriptionService.IncrementUsage(c.UserContext(), uid, actionType)
			}
		}

		return err
	}
}

// MessageUsageLimitMiddleware is a smart middleware for the /messages endpoint.
// It inspects the request body to determine the message type (normal, OCR, PDF chat)
// and enforces the corresponding usage limit.
func (m *MiddlewareManager) MessageUsageLimitMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := c.Locals("userID").(string)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(domain.Error{
				Message: domain.ErrUnauthorized.Error(),
				Code:    fiber.StatusUnauthorized,
			})
		}

		uid := domain.GetUserIDFromString(userID)

		// Detect message type from request body
		actionType := detectMessageActionType(c.Body())

		allowed, err := m.subscriptionService.CheckUsageLimit(c.UserContext(), uid, actionType)
		if err != nil {
			m.logger.Error("Kullanım limiti kontrol hatası",
				"error", err.Error(),
				"userID", userID,
				"action", string(actionType),
			)
			return c.Status(fiber.StatusInternalServerError).JSON(domain.Error{
				Message: domain.ErrInternalServer.Error(),
				Code:    fiber.StatusInternalServerError,
			})
		}

		if !allowed {
			errMsg := getUsageLimitError(actionType)
			return c.Status(fiber.StatusForbidden).JSON(domain.Error{
				Message: errMsg.Error(),
				Code:    fiber.StatusForbidden,
			})
		}

		// Proceed to handler
		err = c.Next()

		// Increment usage on success
		if err == nil && c.Response().StatusCode() >= 200 && c.Response().StatusCode() < 300 {
			_ = m.subscriptionService.IncrementUsage(c.UserContext(), uid, actionType)
		}

		return err
	}
}

// messageTypeProbe is a lightweight struct to peek at the request body
// without fully parsing it.
type messageTypeProbe struct {
	PdfId    uuid.UUID `json:"pdf_id"`
	ImageUrl string    `json:"image_url"`
}

// detectMessageActionType inspects the raw JSON body to determine message type.
func detectMessageActionType(body []byte) domain.ActionType {
	var probe messageTypeProbe
	if err := json.Unmarshal(body, &probe); err != nil {
		return domain.ActionMessage
	}

	if probe.PdfId != uuid.Nil {
		return domain.ActionPDFChat
	}
	if probe.ImageUrl != "" {
		return domain.ActionOCR
	}
	return domain.ActionMessage
}

// getUsageLimitError returns the appropriate Turkish error message for each action type.
func getUsageLimitError(actionType domain.ActionType) error {
	switch actionType {
	case domain.ActionMessage:
		return domain.ErrMessageLimitExceeded
	case domain.ActionOCR:
		return domain.ErrOCRLimitExceeded
	case domain.ActionPDFChat:
		return domain.ErrPDFChatLimitExceeded
	case domain.ActionFileUpload:
		return domain.ErrFileUploadLimitExceeded
	case domain.ActionQuiz:
		return domain.ErrQuizLimitExceeded
	default:
		return domain.ErrUsageLimitExceeded
	}
}
