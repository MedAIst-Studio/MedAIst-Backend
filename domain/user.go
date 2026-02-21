package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	MaxUsernameLength = 100
	MinUsernameLength = 3
)

type User struct {
	Id              uuid.UUID        `json:"id" validate:"required"`
	Username        string           `json:"username" validate:"required"`
	Password        string           `json:"password" validate:"required"`
	IsEmailVerified bool             `json:"isEmailVerified" validate:"required"`
	CreatedAt       time.Time        `json:"createdAt" validate:"required"`
	Email           string           `json:"email" validate:"required"`
	Preferences     *UserPreferences `json:"preferences,omitempty"`
}

func NewUser(username, password, email string) (*User, error) {
	if err := ValidateUsername(username); err != nil {
		return nil, err
	}

	if IsPasswordTooShort(password) {
		return nil, ErrPasswordTooShort
	}

	hashedPassword, err := HashPassword(password)
	if err != nil {
		return nil, ErrInternalServer
	}

	return &User{
		Id:        uuid.New(),
		Username:  username,
		Password:  hashedPassword,
		Email:     email,
		CreatedAt: time.Now(),
	}, nil
}

func (u *User) ValidatePassword(password string) error {

	if IsPasswordTooShort(password) {
		return ErrPasswordTooShort
	}

	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrInvalidCredentials
		}

		return ErrInternalServer
	}
	return nil
}

func (u *User) HashPassword(password string) error {
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return ErrInternalServer
	}
	u.Password = hashedPassword
	return nil
}

func HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", ErrPasswordTooShort
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", ErrInternalServer
	}
	return string(hashedPassword), nil
}

func ValidateUsername(username string) error {

	if len(username) == 0 {
		return ErrEmptyUsername
	}

	if len(username) < MinUsernameLength {
		return ErrTooShortUsername
	}

	return nil
}

func IsPasswordTooShort(password string) bool {
	return len(password) < 8
}

type NotificationPreferences struct {
	Motivation bool `json:"motivation"`
	Reminders  bool `json:"reminders"`
	Updates    bool `json:"updates"`
}

type UserPreferences struct {
	Theme         string                  `json:"theme"`
	Notifications NotificationPreferences `json:"notifications"`
}

type UserRepository interface {
	GetUserById(ctx context.Context, id uuid.UUID) (*User, error)
	UpdateUserPreferences(ctx context.Context, id uuid.UUID, preferences *UserPreferences) error
	GetUserFCMTokenAndPreferences(ctx context.Context, id uuid.UUID) (string, *UserPreferences, error)
	UpdateFCMToken(ctx context.Context, id uuid.UUID, token string) error
	DeleteAccount(ctx context.Context, id uuid.UUID) ([]*File, error)
	UpdateUsername(ctx context.Context, id uuid.UUID, username string) error
	CheckEmail(ctx context.Context, email string) error
	UpdateAccount(ctx context.Context, id uuid.UUID, username string) error
	GetUserOnlyHavingPasswordById(ctx context.Context, id uuid.UUID) (*User, error)
	ChangePassword(ctx context.Context, user *User) error
	EmailExists(ctx context.Context, email string) (bool, error)
	ResetPasswordByEmail(ctx context.Context, email, newPassword string) error
	VerifyEmail(ctx context.Context, email string) error
	GetUserNameAndEmailByIdForSendingVerificationEmail(ctx context.Context, id uuid.UUID) (string, string, error)
	GetLazyUsersForEveningReminder(ctx context.Context) ([]uuid.UUID, error)
}

type UserService interface {
	GetUserById(ctx context.Context, id uuid.UUID) (*User, error)
	ChangePassword(ctx context.Context, id uuid.UUID, oldPassword, newPassword string) error
	DeleteAccount(ctx context.Context, id uuid.UUID) error
	UpdateUsername(ctx context.Context, id uuid.UUID, username string) error
	UpdatePreferences(ctx context.Context, id uuid.UUID, preferences *UserPreferences) error
	UpdateFCMToken(ctx context.Context, id uuid.UUID, token string) error
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, newPassword string) error
}
