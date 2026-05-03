package oauth

import (
	"context"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type OAuthTokens struct {
	Refresh string `json:"refresh_token"`
	Access  string `json:"access_token"`
}

type OAuthService struct {
	Repo         domain.AuthRepository
	TokenService domain.TokenService
}

func NewOAuthService(repo domain.AuthRepository, tokenService domain.TokenService) *OAuthService {
	return &OAuthService{
		Repo:         repo,
		TokenService: tokenService,
	}
}

func (s *OAuthService) HandleOAuthUser(ctx context.Context, oauthCloudUser *domain.OAuthCloudUserInfo, provider string) (*OAuthTokens, error) {
	oauthUser, err := s.Repo.GetOAuthUser(ctx, oauthCloudUser.Email)
	if err != nil {
		return nil, err
	}

	newOAuthDomainUser := domain.NewOAuthUser(oauthCloudUser.Name, oauthCloudUser.Email, oauthCloudUser.ID, provider)
	if oauthUser.IsExistingUser() {
		newOAuthDomainUser.UpdateUserId(oauthUser.Id)
	}

	tokens, err := s.GenerateTokens(newOAuthDomainUser.Id)
	if err != nil {
		return nil, err
	}

	if oauthUser.IsNewUser() {
		if err := s.Repo.CreateUserByOAuth(ctx, &domain.OAuthUserRegistrationData{
			OAuthUser:    newOAuthDomainUser,
			RefreshToken: domain.NewRefreshToken(newOAuthDomainUser.Id, tokens.Refresh),
		}); err != nil {
			return nil, err
		}
	} else if oauthUser.IsDifferentProvider(provider) || oauthUser.HasNoProvider() {
		if err := s.Repo.AddOAuthUser(ctx, &domain.OAuthUserRegistrationData{
			OAuthUser:    newOAuthDomainUser,
			RefreshToken: domain.NewRefreshToken(newOAuthDomainUser.Id, tokens.Refresh),
		}); err != nil {
			return nil, err
		}
	} else {
		if err := s.Repo.UpsertRefreshToken(ctx, domain.NewRefreshToken(newOAuthDomainUser.Id, tokens.Refresh)); err != nil {
			return nil, err
		}
	}

	return tokens, nil
}

func (s *OAuthService) GenerateTokens(userId uuid.UUID) (*OAuthTokens, error) {
	accessToken, err := s.TokenService.GenerateAuthAccessToken(userId.String())
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.TokenService.GenerateAuthRefreshToken(userId.String())
	if err != nil {
		return nil, err
	}

	return &OAuthTokens{
		Refresh: refreshToken,
		Access:  accessToken,
	}, nil
}
