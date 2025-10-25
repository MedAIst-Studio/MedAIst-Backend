package google

import (
	"github.com/MedAIst-Studio/MedAIist-Backend/app/auth"
	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleOAuthDependenciesConfig struct {
	Repo         auth.Repository
	TokenService auth.TokenService
	Cache        auth.Cache
	Logger       domain.Logger
	ClientID     string
	ClientSecret string
	RedirectURL  string
	FrontendURL  string
}

type GoogleOAuthService struct {
	GoogleOAuthDependenciesConfig
	config *oauth2.Config
}

func NewGoogleOAuthService(cfg *GoogleOAuthDependenciesConfig) *GoogleOAuthService {
	return &GoogleOAuthService{
		GoogleOAuthDependenciesConfig: *cfg,
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Endpoint:     google.Endpoint,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
		},
	}
}
