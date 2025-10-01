package auth

import (
	"context"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

type EmailService interface {
	SendWelcomeEmail(ctx context.Context, ml *domain.EmailClaims) error
	SendVerificationEmail(ctx context.Context, ml *domain.EmailClaims) error
}
