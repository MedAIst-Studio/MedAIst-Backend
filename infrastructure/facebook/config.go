package facebook

import (
	"github.com/MedAIst-Studio/MedAIist-Backend/app/auth"
	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
)

type FacebookOAuthDependenciesConfig struct {
	Repo         auth.Repository
	TokenService auth.TokenService
	Cache        auth.Cache
	Logger       domain.Logger
	ClientID     string
	ClientSecret string
	RedirectURL  string
	FrontendURL  string
}

type FacebookOAuthService struct {
	FacebookOAuthDependenciesConfig
	config *oauth2.Config
}

func NewFacebookOAuthService(cfg *FacebookOAuthDependenciesConfig) *FacebookOAuthService {
	return &FacebookOAuthService{
		FacebookOAuthDependenciesConfig: *cfg,
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Endpoint:     facebook.Endpoint,
			Scopes: []string{
				"public_profile",
				"email",
			},
		},
	}
}
