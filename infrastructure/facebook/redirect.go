package facebook

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/oauth2"
)

func (h *FacebookOAuthService) RedirectHandler(ctx *fiber.Ctx) error {
	state := generateState()
	err := h.Cache.SaveOAuthState(ctx.Context(), state)
	if err != nil {
		h.Logger.Error("failed to save OAuth state", "error", err)
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "internal server error",
			"code":    fiber.StatusInternalServerError,
		})
	}

	url := h.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	return ctx.Redirect(url, fiber.StatusTemporaryRedirect)
}

func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
