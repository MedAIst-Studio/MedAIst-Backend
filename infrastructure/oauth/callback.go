package oauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/oauth2"
)

func (h *OAuthProvider) Callback(ctx *fiber.Ctx) error {
	code := ctx.Query("code")
	state := ctx.Query("state")

	if code == "" || state == "" {
		return h.redirectToFrontendWithError(ctx, "missing_params")
	}

	valid, err := h.Cache.ValidateOAuthState(ctx.Context(), state)
	if err != nil || !valid {
		h.Logger.Error("invalid state, possible CSRF attack")
		return h.redirectToFrontendWithError(ctx, "invalid_state")
	}
	h.Cache.DeleteOAuthState(ctx.Context(), state)

	token, err := h.config.Exchange(context.Background(), code)
	if err != nil {
		h.Logger.Error("token exchange failed: ", err)
		return h.redirectToFrontendWithError(ctx, "token_exchange_failed")
	}

	cloudUserInfo, err := h.getCloudUserInfo(token)
	if err != nil {
		h.Logger.Error("failed to get user info: ", err)
		return h.redirectToFrontendWithError(ctx, "user_info_failed")
	}

	tokens, err := h.Service.HandleOAuthUser(ctx.Context(), cloudUserInfo, h.Provider)
	if err != nil {
		h.Logger.Error("failed to handle user: ", err)
		return h.redirectToFrontendWithError(ctx, "user_handling_failed")
	}

	sessionKey := generateSessionKey()
	session := &domain.OAuthSession{
		SessionKey:   sessionKey,
		AccessToken:  tokens.Access,
		RefreshToken: tokens.Refresh,
		CreatedAt:    time.Now(),
	}

	if err := h.Cache.SaveOAuthSession(ctx.Context(), session); err != nil {
		h.Logger.Error("failed to save session: ", err)
		return h.redirectToFrontendWithError(ctx, "session_failed")
	}
	return h.redirectToFrontendCallback(ctx, sessionKey)
}

func (h *OAuthProvider) getCloudUserInfo(token *oauth2.Token) (*domain.OAuthCloudUserInfo, error) {
	client := h.config.Client(context.Background(), token)
	resp, err := client.Get(h.GetUserInfoURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var info domain.OAuthCloudUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

func (h *OAuthProvider) redirectToFrontendWithError(ctx *fiber.Ctx, errorCode string) error {
	redirectURL := fmt.Sprintf("%s/login?error=%s", h.FrontendURL, errorCode)
	return ctx.Redirect(redirectURL, fiber.StatusTemporaryRedirect)
}

func (h *OAuthProvider) redirectToFrontendCallback(ctx *fiber.Ctx, sessionKey string) error {
	redirectURL := fmt.Sprintf("%s/auth/callback?session=%s", h.FrontendURL, sessionKey)
	return ctx.Redirect(redirectURL, fiber.StatusTemporaryRedirect)
}

func generateSessionKey() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
