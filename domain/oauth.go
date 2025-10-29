package domain

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

type OAuthUser struct {
	User
	OAuthId   string   `json:"oauthId"`
	Providers []string `json:"providers"`
	Exists    bool     `json:"exists"`
}

func NewOAuthUser(username, email, oauthID, provider string) *OAuthUser {
	return &OAuthUser{
		User: User{
			Id:        uuid.New(),
			Username:  username,
			Password:  "",
			Email:     email,
			CreatedAt: time.Now(),
		},
		OAuthId:   oauthID,
		Providers: []string{provider},
	}
}

func (o *OAuthUser) IsNewUser() bool {
	return !o.Exists
}

func (o *OAuthUser) IsExistingUser() bool {
	return o.Exists
}

func (o *OAuthUser) IsDifferentProvider(provider string) bool {
	return !slices.Contains(o.Providers, provider)
}

func (o *OAuthUser) HasNoProvider() bool {
	return len(o.Providers) == 0
}

func (o *OAuthUser) UpdateOAuthId(newId uuid.UUID) {
	o.OAuthId = newId.String()
}

func (o *OAuthUser) UpdateUserId(newId uuid.UUID) {
	o.Id = newId
}

func (o *OAuthUser) ResetProviders(newProvider string) {
	o.Providers = []string{newProvider}
}

type OAuthSession struct {
	SessionKey   string `json:"session_key"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	CreatedAt    time.Time
}

type OAuthCloudUserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}
