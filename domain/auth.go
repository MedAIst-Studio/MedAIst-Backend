package domain

import (
	"context"
)

type AuthService interface {
	Login(ctx context.Context, email, password string) (string, string, error) // access token, refresh token, error
	Signup(ctx context.Context, username, password, email string) (string, string, error)
	Logout(ctx context.Context, refreshToken string) error
	RefreshToken(ctx context.Context, refreshToken string) (string, error) // returns new access token
	ExchangeOAuthSession(ctx context.Context, sessionKey string) (string, string, error)
}

type AuthRepository interface {
	CreateUser(ctx context.Context, data *SignupData) error
	CreateUserByOAuth(ctx context.Context, data *OAuthUserRegistrationData) error
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	UpsertRefreshToken(ctx context.Context, record *RefreshToken) error
	DeleteRefreshToken(ctx context.Context, token string) error
	RefreshTokenExists(ctx context.Context, token string) (bool, error)
	GetOAuthUser(ctx context.Context, email string) (*OAuthUser, error)
	AddOAuthUser(ctx context.Context, data *OAuthUserRegistrationData) error
}

type SignupData struct {
	User         *User
	RefreshToken *RefreshToken
}

type OAuthUserRegistrationData struct {
	OAuthUser    *OAuthUser
	RefreshToken *RefreshToken
}

type AuthCache interface {
	SaveOAuthState(ctx context.Context, state string) error
	ValidateOAuthState(ctx context.Context, state string) (bool, error)
	DeleteOAuthState(ctx context.Context, state string) error
	SaveOAuthSession(ctx context.Context, session *OAuthSession) error
	GetOAuthSession(ctx context.Context, sessionKey string) (*OAuthSession, error)
	DeleteOAuthSession(ctx context.Context, sessionKey string) error
}
