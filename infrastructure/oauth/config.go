package oauth

import (
	"github.com/MedAIst-Studio/MedAIist-Backend/app/auth"
	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"golang.org/x/oauth2"
)

type OAuthDependenciesConfig struct {
	Provider     string
	Repository   auth.Repository
	TokenService auth.TokenService
	Cache        auth.Cache
	Logger       domain.Logger
	ClientID     string
	ClientSecret string
	RedirectURL  string
	FrontendURL  string
}

type OAuthProvider struct {
	OAuthDependenciesConfig
	config *oauth2.Config
}

func NewOAuthProvider(cfg *OAuthDependenciesConfig, endpoint oauth2.Endpoint, scopes []string) *OAuthProvider {
	return &OAuthProvider{
		OAuthDependenciesConfig: *cfg,
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Endpoint:     endpoint,
			Scopes:       scopes,
		},
	}
}
