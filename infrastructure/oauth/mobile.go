package oauth

import (
	"strings"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/gofiber/fiber/v2"
)

type MobileOAuthHandler struct {
	Service   *OAuthService
	Verifiers map[string]TokenVerifier
	Logger    domain.Logger
}

func NewMobileOAuthHandler(service *OAuthService, verifiers map[string]TokenVerifier, logger domain.Logger) *MobileOAuthHandler {
	return &MobileOAuthHandler{
		Service:   service,
		Verifiers: verifiers,
		Logger:    logger,
	}
}

type MobileLoginRequest struct {
	Provider    string `json:"provider"`
	IdToken     string `json:"idToken"`     // For Google
	AccessToken string `json:"accessToken"` // For Facebook
}

// Handle godoc
//
//	@Summary		Mobile OAuth Login
//	@Description	Authenticate user via Google or Facebook mobile tokens.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		MobileLoginRequest	true	"Mobile Login Request"
//	@Success		200		{object}	OAuthTokens
//	@Failure		400		{object}	map[string]string
//	@Router			/api/auth/oauth/mobile [post]
func (h *MobileOAuthHandler) Handle(ctx *fiber.Ctx) error {
	var req MobileLoginRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_request_body"})
	}

	provider := strings.ToLower(req.Provider)
	verifier, ok := h.Verifiers[provider]
	if !ok {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "unsupported_provider"})
	}

	var token string
	switch provider {
	case "google":
		token = req.IdToken
	case "facebook":
		token = req.AccessToken
	default:
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "unsupported_provider"})
	}

	if token == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing_token"})
	}

	cloudUser, err := verifier.Verify(token)
	if err != nil {
		h.Logger.Error("Token verification failed", "provider", req.Provider, "error", err)
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid_token"})
	}

	tokens, err := h.Service.HandleOAuthUser(ctx.Context(), cloudUser, strings.ToUpper(req.Provider))
	if err != nil {
		h.Logger.Error("Failed to handle OAuth user", "error", err)
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_server_error"})
	}

	return ctx.JSON(tokens)
}
