package user

import (
	"context"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type Repository interface {
	GetUserById(ctx context.Context, id uuid.UUID) (*GetCurrentUserResponse, error)
	CheckEmail(ctx context.Context, email string) error
	DeleteAccount(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error)
	GetUserOnlyHavingPasswordById(ctx context.Context, id uuid.UUID) (*domain.User, error)
	EmailExists(ctx context.Context, email string) (bool, error)
	ResetPasswordByEmail(ctx context.Context, email, newPassword string) error
	ChangePassword(ctx context.Context, user *domain.User) error
	UpdateUsername(ctx context.Context, id uuid.UUID, username string) error
	VerifyEmail(ctx context.Context, email string) error
	GetUserNameAndEmailByIdForSendingVerificationEmail(ctx context.Context, id uuid.UUID) (string, string, error)
}

type TokenService interface {
	GenerateSecureEmailToken(email string) (string, error)
	ValidateSecureEmailToken(tokenString string) (string, error)
}

type MailService interface {
	SendSuccessfullyDeletedEmail(ctx context.Context, claims *domain.EmailClaims) error
	SendPasswordResetEmail(ctx context.Context, claims *domain.EmailClaims) error
	SendVerificationEmail(ctx context.Context, claims *domain.EmailClaims) error
}

type FileService interface {
	DeleteFiles(ctx context.Context, ids []string) error
}

type VectorRepository interface {
	DeleteEmbeddingsByFileIDs(ctx context.Context, ids []string) error
}
