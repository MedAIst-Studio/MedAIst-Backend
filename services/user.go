package services

import (
	"context"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type UserService struct {
	repo         domain.UserRepository
	fs           domain.FileStorageService
	vb           domain.VectorDB
	emailService domain.MailService
	tokenService domain.TokenService
	logger       domain.Logger
}

func NewUserService(repo domain.UserRepository, fs domain.FileStorageService, vb domain.VectorDB, emailService domain.MailService, tokenService domain.TokenService, logger domain.Logger) *UserService {
	return &UserService{
		repo:         repo,
		fs:           fs,
		vb:           vb,
		emailService: emailService,
		tokenService: tokenService,
		logger:       logger,
	}
}

func (s *UserService) GetUserById(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.repo.GetUserById(ctx, id)
}

func (s *UserService) ChangePassword(ctx context.Context, id uuid.UUID, oldPassword, newPassword string) error {
	if len(oldPassword) < 8 || len(newPassword) < 8 {
		return domain.ErrPasswordTooShort
	}

	user, err := s.repo.GetUserOnlyHavingPasswordById(ctx, id)
	if err != nil {
		return err
	}

	if err := user.ValidatePassword(oldPassword); err != nil {
		return domain.ErrInvalidCredentials
	}

	if err := user.HashPassword(newPassword); err != nil {
		return err
	}

	user.Id = id

	return s.repo.ChangePassword(ctx, user)
}

func (s *UserService) DeleteAccount(ctx context.Context, id uuid.UUID) error {
	files, err := s.repo.DeleteAccount(ctx, id)
	if err != nil {
		return err
	}

	var filePaths []string
	var fileIds []string

	for _, f := range files {
		filePaths = append(filePaths, f.Path)
		fileIds = append(fileIds, f.Id.String())
	}

	if err := s.fs.DeleteFiles(ctx, filePaths); err != nil {
		return err
	}

	if err := s.vb.DeleteEmbeddingsByFileIDs(ctx, fileIds); err != nil {
		return err
	}

	return nil
}

func (s *UserService) UpdateUsername(ctx context.Context, id uuid.UUID, username string) error {
	if err := domain.ValidateUsername(username); err != nil {
		return err
	}
	return s.repo.UpdateUsername(ctx, id, username)
}

func (s *UserService) UpdatePreferences(ctx context.Context, id uuid.UUID, preferences *domain.UserPreferences) error {
	return s.repo.UpdateUserPreferences(ctx, id, preferences)
}

func (s *UserService) UpdateFCMToken(ctx context.Context, id uuid.UUID, token string) error {
	return s.repo.UpdateFCMToken(ctx, id, token)
}

func (s *UserService) ForgotPassword(ctx context.Context, email string) error {
	exists, err := s.repo.EmailExists(ctx, email)
	if err != nil {
		return domain.ErrInternalServer
	}
	if !exists {
		return nil
	}

	token, err := s.tokenService.GenerateSecureEmailToken(email)
	if err != nil {
		s.logger.Error("failed to generate token for forgot password: ", err)
		return domain.ErrInternalServer
	}

	if err := s.emailService.SendPasswordResetEmail(context.Background(), &domain.EmailClaims{
		To:      email,
		Subject: "Reset Your Password" + token,
		HTML:    "deneme deneme",
	}); err != nil {
		s.logger.Error("failed to send forgot password email: ", err)
		return domain.ErrInternalServer
	}

	return nil
}

func (s *UserService) ResetPassword(ctx context.Context, token, newPassword string) error {
	email, err := s.tokenService.ValidateSecureEmailToken(token)
	if err != nil {
		s.logger.Error("failed to validate token for resetting password: ", err)
		return domain.ErrInternalServer
	}

	if len(newPassword) < 8 {
		return domain.ErrPasswordTooShort
	}

	hashedPassword, err := domain.HashPassword(newPassword)
	if err != nil {
		s.logger.Error("failed to hash password for resetting password: ", err)
		return domain.ErrInternalServer
	}

	return s.repo.ResetPasswordByEmail(ctx, email, hashedPassword)
}
