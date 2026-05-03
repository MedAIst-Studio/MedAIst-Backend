package services

import (
	"context"
	"errors"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

type AuthService struct {
	repo         domain.AuthRepository
	tokenService domain.TokenService
	emailService domain.MailService
	cache        domain.AuthCache
}

func NewAuthService(repo domain.AuthRepository, tokenService domain.TokenService, emailService domain.MailService, cache domain.AuthCache) *AuthService {
	return &AuthService{
		repo:         repo,
		tokenService: tokenService,
		emailService: emailService,
		cache:        cache,
	}
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, string, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrEmailNotFound) {
			return "", "", domain.ErrEmailNotFound
		}
		return "", "", domain.ErrInternalServer
	}

	if err = user.ValidatePassword(password); err != nil {
		if !errors.Is(err, domain.ErrInternalServer) {
			return "", "", err
		}
		return "", "", domain.ErrInternalServer
	}

	accessToken, err := s.tokenService.GenerateAuthAccessToken(user.Id.String())
	if err != nil {
		return "", "", domain.ErrInternalServer
	}

	refreshToken, err := s.tokenService.GenerateAuthRefreshToken(user.Id.String())
	if err != nil {
		return "", "", domain.ErrInternalServer
	}

	if err := s.repo.UpsertRefreshToken(ctx, domain.NewRefreshToken(
		user.Id,
		refreshToken,
	)); err != nil {
		return "", "", domain.ErrInternalServer
	}

	return accessToken, refreshToken, nil
}

func (s *AuthService) Signup(ctx context.Context, username, password, email string) (string, string, error) {
	user, err := domain.NewUser(username, password, email)
	if err != nil {
		if !errors.Is(err, domain.ErrInternalServer) {
			return "", "", err
		}
		return "", "", domain.ErrInternalServer
	}

	accessToken, err := s.tokenService.GenerateAuthAccessToken(user.Id.String())
	if err != nil {
		return "", "", domain.ErrInternalServer
	}

	refreshToken, err := s.tokenService.GenerateAuthRefreshToken(user.Id.String())
	if err != nil {
		return "", "", domain.ErrInternalServer
	}

	if err = s.repo.CreateUser(ctx, &domain.SignupData{
		User:         user,
		RefreshToken: domain.NewRefreshToken(user.Id, refreshToken),
	}); err != nil {
		if errors.Is(err, domain.ErrEmailAlreadyExists) {
			return "", "", domain.ErrEmailAlreadyExists
		}
		return "", "", domain.ErrInternalServer
	}

	// Email sending should ideally be handled asynchronously, potentially moved out of the core service loop or managed by an event bus depending on architecture, but we'll keep it here for now
	go func() {
		_ = s.emailService.SendWelcomeEmail(context.Background(), &domain.EmailClaims{
			Name:    user.Username,
			To:      user.Email,
			Subject: "Welcome to MedAIist!",
			HTML:    "We're excited to have you on board.",
		})
	}()

	return accessToken, refreshToken, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	if err := s.repo.DeleteRefreshToken(ctx, refreshToken); err != nil {
		return domain.ErrInternalServer
	}
	return nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (string, error) {
	payload, err := s.tokenService.ValidateAuthRefreshToken(refreshToken)
	if err != nil {
		if !errors.Is(err, domain.ErrInternalServer) {
			return "", err
		}
		return "", domain.ErrInternalServer
	}

	exist, err := s.repo.RefreshTokenExists(ctx, refreshToken)
	if err != nil {
		return "", domain.ErrInternalServer
	}

	if !exist {
		return "", domain.ErrNotExistRefreshToken
	}

	accessToken, err := s.tokenService.GenerateAuthAccessToken(payload.UserID)
	if err != nil {
		return "", domain.ErrInternalServer
	}

	return accessToken, nil
}

func (s *AuthService) ExchangeOAuthSession(ctx context.Context, sessionKey string) (string, string, error) {
	session, err := s.cache.GetOAuthSession(ctx, sessionKey)
	if err != nil {
		if !errors.Is(err, domain.ErrInternalServer) {
			return "", "", err
		}
		return "", "", domain.ErrInternalServer
	}

	_ = s.cache.DeleteOAuthSession(ctx, sessionKey)

	return session.AccessToken, session.RefreshToken, nil
}
