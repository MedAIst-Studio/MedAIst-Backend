package auth

import (
	"context"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

type EmailService interface {
	SendWelcomeEmail(ctx context.Context, ml *domain.EmailClaims) error
	SendVerificationEmail(ctx context.Context, ml *domain.EmailClaims) error
}

type Repository interface {
	CreateUser(ctx context.Context, data *SignupData) error
	CreateUserByOAuth(ctx context.Context, data *OAuthUserRegistrationData) error
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	UpsertRefreshToken(ctx context.Context, record *domain.RefreshToken) error
	DeleteRefreshToken(ctx context.Context, token string) error
	RefreshTokenExists(ctx context.Context, token string) (bool, error)
	GetOAuthUser(ctx context.Context, email string) (*domain.OAuthUser, error)
	AddOAuthUser(ctx context.Context, data *OAuthUserRegistrationData) error
}

type SignupData struct {
	User         *domain.User
	RefreshToken *domain.RefreshToken
}

type OAuthUserRegistrationData struct {
	OAuthUser    *domain.OAuthUser
	RefreshToken *domain.RefreshToken
}

type TokenPayload struct {
	UserID string `json:"userID"`
}

type TokenService interface {
	GenerateAuthAccessToken(userID string) (string, error)
	ValidateAuthAccessToken(token string) (*TokenPayload, error)
	GenerateAuthRefreshToken(userID string) (string, error)
	ValidateAuthRefreshToken(token string) (*TokenPayload, error)
	GenerateSecureEmailToken(email string) (string, error)
}

type Cache interface {
	SaveOAuthState(ctx context.Context, state string) error
	ValidateOAuthState(ctx context.Context, state string) (bool, error)
	DeleteOAuthState(ctx context.Context, state string) error
	SaveOAuthSession(ctx context.Context, session *domain.OAuthSession) error
	GetOAuthSession(ctx context.Context, sessionKey string) (*domain.OAuthSession, error)
	DeleteOAuthSession(ctx context.Context, sessionKey string) error
}
