package auth

import (
	"context"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

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
