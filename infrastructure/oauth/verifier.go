package oauth

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

type TokenVerifier interface {
	Verify(token string) (*domain.OAuthCloudUserInfo, error)
}

type GoogleVerifier struct{}

func NewGoogleVerifier() *GoogleVerifier {
	return &GoogleVerifier{}
}

func (v *GoogleVerifier) Verify(idToken string) (*domain.OAuthCloudUserInfo, error) {
	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + idToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("invalid google token")
	}

	var claims struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&claims); err != nil {
		return nil, err
	}

	return &domain.OAuthCloudUserInfo{
		ID:    claims.Sub,
		Email: claims.Email,
		Name:  claims.Name,
	}, nil
}

type FacebookVerifier struct{}

func NewFacebookVerifier() *FacebookVerifier {
	return &FacebookVerifier{}
}

func (v *FacebookVerifier) Verify(accessToken string) (*domain.OAuthCloudUserInfo, error) {
	resp, err := http.Get("https://graph.facebook.com/me?fields=id,name,email&access_token=" + accessToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("invalid facebook token")
	}

	var claims struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&claims); err != nil {
		return nil, err
	}

	return &domain.OAuthCloudUserInfo{
		ID:    claims.ID,
		Email: claims.Email,
		Name:  claims.Name,
	}, nil
}
