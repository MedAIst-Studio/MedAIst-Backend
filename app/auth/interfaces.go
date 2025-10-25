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
	CreateSignupData(ctx context.Context, data *SignupData) error
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	SaveRefreshToken(ctx context.Context, record *domain.RefreshToken) error
	UpsertRefreshToken(ctx context.Context, record *domain.RefreshToken) error
	DeleteRefreshToken(ctx context.Context, token string) error
	RefreshTokenExists(ctx context.Context, token string) (bool, error)
}

type SignupData struct {
	User         *domain.User
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
