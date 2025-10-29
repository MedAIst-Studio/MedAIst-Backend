package fiber

import (
	"os"

	"github.com/MedAIst-Studio/MedAIist-Backend/infrastructure/oauth"
	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2/google"
)

func NewGoogleOAuthService(container *DependencyContainer) *oauth.OAuthProvider {
	return oauth.NewOAuthProvider(
		&oauth.OAuthDependenciesConfig{
			Provider:     "GOOGLE",
			Repository:   container.Repo,
			TokenService: container.jweTokenService,
			Cache:        container.inMemoryClient,
			FrontendURL:  "http://localhost:8000",
			ClientID:     os.Getenv("GOOGLE_OAUTH_CLIENT_ID"),
			ClientSecret: os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"),
			RedirectURL:  os.Getenv("GOOGLE_OAUTH_REDIRECT_URL"),
			Logger:       container.slogLogger,
		}, google.Endpoint, []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
	)
}

func NewFacebookOAuthService(container *DependencyContainer) *oauth.OAuthProvider {
	return oauth.NewOAuthProvider(
		&oauth.OAuthDependenciesConfig{
			Provider:     "FACEBOOK",
			Repository:   container.Repo,
			TokenService: container.jweTokenService,
			Cache:        container.inMemoryClient,
			FrontendURL:  "http://localhost:8000",
			ClientID:     os.Getenv("FACEBOOK_OAUTH_CLIENT_ID"),
			ClientSecret: os.Getenv("FACEBOOK_OAUTH_CLIENT_SECRET"),
			RedirectURL:  os.Getenv("FACEBOOK_OAUTH_REDIRECT_URL"),
			Logger:       container.slogLogger,
		}, facebook.Endpoint, []string{
			"email",
			"public_profile",
		},
	)
}
