package google

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/MedAIst-Studio/MedAIist-Backend/app/auth"
	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/oauth2"
)

func (h *GoogleOAuthService) Callback(ctx *fiber.Ctx) error {
	fmt.Println("Google OAuth callback invoked")
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

	userInfo, err := h.getGoogleUserInfo(token)
	fmt.Printf("userInfo: %v\n", userInfo)
	if err != nil {
		h.Logger.Error("failed to get user info: ", err)
		return h.redirectToFrontendWithError(ctx, "user_info_failed")
	}

	// User login/signup
	tokens, err := h.handleUser(ctx, userInfo)
	if err != nil {
		h.Logger.Error("failed to handle user: ", err)
		return h.redirectToFrontendWithError(ctx, "user_handling_failed")

	}

	// Session oluştur
	sessionKey := generateSessionKey()
	fmt.Printf("sessionKey: %v\n", sessionKey)
	session := &auth.OAuthSession{
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

type GoogleUserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (h *GoogleOAuthService) getGoogleUserInfo(token *oauth2.Token) (*GoogleUserInfo, error) {
	client := h.config.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var info GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

type Tokens struct {
	Refresh string
	Access  string
}

func (h *GoogleOAuthService) handleUser(ctx *fiber.Ctx, googleUser *GoogleUserInfo) (*Tokens, error) {
	currentUserId, err := h.Repo.UserExistsByOAuthID(ctx.Context(), "GOOGLE", googleUser.ID)
	if err != nil {
		return nil, err
	}

	isNewUser := currentUserId == uuid.Nil

	user := domain.NewOAuthUser(googleUser.Name, googleUser.Email, "GOOGLE", googleUser.ID)

	if isNewUser {
		currentUserId = user.Id
	}

	accessToken, err := h.TokenService.GenerateAuthAccessToken(currentUserId.String())
	if err != nil {
		return nil, err
	}

	refreshToken, err := h.TokenService.GenerateAuthRefreshToken(currentUserId.String())
	if err != nil {
		return nil, err
	}

	if isNewUser {
		if err := h.Repo.CreateUserByOAuth(ctx.Context(), &auth.SignupData{
			User:         user,
			RefreshToken: domain.NewRefreshToken(user.Id, refreshToken),
		}); err != nil {
			return nil, err
		}
	}

	return &Tokens{
		Refresh: refreshToken,
		Access:  accessToken,
	}, nil
}

func (h *GoogleOAuthService) redirectToFrontendWithError(ctx *fiber.Ctx, errorCode string) error {
	redirectURL := fmt.Sprintf("%s/login?error=%s", h.FrontendURL, errorCode)
	return ctx.Redirect(redirectURL, fiber.StatusTemporaryRedirect)
}

func (h *GoogleOAuthService) redirectToFrontendCallback(ctx *fiber.Ctx, sessionKey string) error {
	redirectURL := fmt.Sprintf("%s/auth/callback?session=%s", h.FrontendURL, sessionKey)
	return ctx.Redirect(redirectURL, fiber.StatusTemporaryRedirect)
}

func generateSessionKey() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
