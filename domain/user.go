package domain

import (
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
	Id              uuid.UUID `json:"id" validate:"required"`
	Username        string    `json:"username" validate:"required"`
	Password        string    `json:"password" validate:"required"`
	IsEmailVerified bool      `json:"isEmailVerified" validate:"required"`
	CreatedAt       time.Time `json:"createdAt" validate:"required"`
	Email           string    `json:"email" validate:"required"`
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
