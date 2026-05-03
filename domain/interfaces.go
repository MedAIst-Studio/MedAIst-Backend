package domain

import (
	"context"
)

type TokenPayload struct {
	UserID string `json:"userID"`
}

type TokenService interface {
	GenerateAuthAccessToken(userID string) (string, error)
	ValidateAuthAccessToken(token string) (*TokenPayload, error)
	GenerateAuthRefreshToken(userID string) (string, error)
	ValidateAuthRefreshToken(token string) (*TokenPayload, error)
	GenerateSecureEmailToken(email string) (string, error)
	ValidateSecureEmailToken(token string) (string, error)
}

type MailService interface {
	SendWelcomeEmail(ctx context.Context, ml *EmailClaims) error
	SendVerificationEmail(ctx context.Context, ml *EmailClaims) error
	SendPasswordResetEmail(ctx context.Context, ml *EmailClaims) error
}
