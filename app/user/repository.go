package user

import (
	"context"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"

	"github.com/google/uuid"
)

type Repository interface {
	GetUserById(ctx context.Context, id uuid.UUID) (*GetCurrentUserResponse, error)
	CheckEmail(ctx context.Context, email string) error
	DeleteAccount(ctx context.Context, id uuid.UUID) error
	GetUserOnlyHavingPasswordById(ctx context.Context, id uuid.UUID) (*domain.User, error)
	EmailExists(ctx context.Context, email string) (bool, error)
	ResetPasswordByEmail(ctx context.Context, email, newPassword string) error
	ChangePassword(ctx context.Context, user *domain.User) error
	UpdateUsername(ctx context.Context, id uuid.UUID, username string) error
	VerifyEmail(ctx context.Context, email string) error
	GetUserNameAndEmailByIdForSendingVerificationEmail(ctx context.Context, id uuid.UUID) (string, string, error)
	GetUserByIdForAdmin(ctx context.Context, id uuid.UUID) (*GetUserResponse, error)
	GetUsers(ctx context.Context, page, limit int) (GetUsersResponse, error)
}
