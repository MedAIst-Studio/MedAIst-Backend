package oauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MedAIst-Studio/MedAIist-Backend/app/auth"
	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

func (h *OAuthProvider) Callback(ctx *fiber.Ctx) error {
	fmt.Println("Facebook OAuth callback invoked")
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

	userInfo, err := h.getFacebookUserInfo(token)
	fmt.Printf("user Facebook Info: %v\n", userInfo)
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

type FacebookUserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (h *OAuthProvider) getFacebookUserInfo(token *oauth2.Token) (*FacebookUserInfo, error) {
	client := h.config.Client(context.Background(), token)
	resp, err := client.Get("https://graph.facebook.com/me?fields=id,name,email")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var info FacebookUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

type Tokens struct {
	Refresh string
	Access  string
}

func (h *OAuthProvider) handleUser(ctx *fiber.Ctx, facebookUser *FacebookUserInfo) (*Tokens, error) {
	currentUserId, err := h.Repository.UserExistsByOAuthID(ctx.Context(), "FACEBOOK", facebookUser.ID)
	if err != nil {
		return nil, err
	}
	fmt.Printf("currentUserId: %v\n", currentUserId)
	isNewUser := currentUserId == uuid.Nil

	user := domain.NewOAuthUser(facebookUser.Name, facebookUser.Email, "FACEBOOK", facebookUser.ID)

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
		fmt.Printf("Creating new user: %v\n", user)
		if err := h.Repository.CreateUserByOAuth(ctx.Context(), &auth.SignupData{
			User:         user,
			RefreshToken: domain.NewRefreshToken(user.Id, refreshToken),
		}); err != nil {
			return nil, err
		}
	} else {
		if err := h.Repository.UpsertRefreshToken(ctx.Context(), domain.NewRefreshToken(currentUserId, refreshToken)); err != nil {
			return nil, err
		}
	}

	return &Tokens{
		Refresh: refreshToken,
		Access:  accessToken,
	}, nil
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
