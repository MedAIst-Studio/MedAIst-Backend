package auth

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

type SignupRequest struct {
	FullName string `json:"fullName" validate:"required"`
	Password string `json:"password" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Phone    string `json:"phone" validate:"required"`
}

type SignupResponse struct {
	Role         string `json:"role"`
	RefreshToken string `json:"refreshToken" swaggerignore:"true"`
	AccessToken  string `json:"accessToken" swaggerignore:"true"`
}

type SignupConfig struct {
	Repo         Repository
	TokenService TokenService
	EmailService EmailService
	Validator    domain.Validator
	Logger       domain.Logger
}

type SignupHandler struct {
	secure    bool
	repo      Repository
	ts        TokenService
	es        EmailService
	validator domain.Validator
	logger    domain.Logger
}

func NewSignupHandler(config *SignupConfig) *SignupHandler {
	return &SignupHandler{
		repo:      config.Repo,
		ts:        config.TokenService,
		es:        config.EmailService,
		validator: config.Validator,
		logger:    config.Logger,
	}
}

//	@Summary		Signup
//	@Description	Signup a new user
//	@Tags			Auth
//	@Accept			json
//	@Produce		json

// @Param		request	body		SignupRequest	true	"Signup request"
// @Success	200		{object}	SignupResponse
// @Failure	400
// @Failure	409
// @Failure	500
// @Router		/auth/signup [post]
func (h *SignupHandler) Handle(ctx context.Context, req *SignupRequest) (*SignupResponse, int, error) {
	if err := h.validator.Validate(req); err != nil {
		return nil, http.StatusBadRequest, domain.ErrInvalidRequest
	}

	user, err := domain.NewUser(req.FullName, req.Password, req.Email)
	if err != nil {
		if !errors.Is(err, domain.ErrInternalServer) {
			return nil, http.StatusBadRequest, err
		}
		h.logger.Error("error while creating domain user: ", err)
		return nil, http.StatusInternalServerError, domain.ErrInternalServer
	}

	accessToken, err := h.ts.GenerateAuthAccessToken(user.Id.String(), user.Role)
	if err != nil {
		h.logger.Error("error while generating access token: ", err)
		return nil, http.StatusInternalServerError, domain.ErrInternalServer
	}

	refreshToken, err := h.ts.GenerateAuthRefreshToken(user.Id.String(), user.Role)
	if err != nil {
		h.logger.Error("error while generating refresh token: ", err)
		return nil, http.StatusInternalServerError, domain.ErrInternalServer
	}

	if err = h.repo.CreateSignupData(ctx, &SignupData{
		User:         user,
		RefreshToken: domain.NewRefreshToken(user.Id, refreshToken),
	}); err != nil {
		if errors.Is(err, domain.ErrEmailAlreadyExists) {
			return nil, http.StatusConflict, domain.ErrEmailAlreadyExists
		}
		h.logger.Error("error while creating user and cart in repository: ", err)
		return nil, http.StatusInternalServerError, domain.ErrInternalServer
	}

	go h.sendWelcomeEmail(&UserEmailInfo{
		username: user.FullName,
		email:    user.Email,
	})

	return &SignupResponse{
		Role:         user.Role,
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
	}, http.StatusCreated, nil
}

type UserEmailInfo struct {
	username string
	email    string
}

func (h *SignupHandler) sendWelcomeEmail(info *UserEmailInfo) {
	maxRetries := 3
	retryInterval := 30 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {

		err := h.es.SendWelcomeEmail(context.Background(), &domain.EmailClaims{
			Name:    info.username,
			To:      info.email,
			Subject: "Welcome to MedAIist!",
			HTML:    "We're excited to have you on board.",
		})

		if err == nil {
			return
		}

		if attempt < maxRetries {
			time.Sleep(retryInterval)
		}
	}
	h.logger.Error("all ", maxRetries, " attempts failed for welcome email to ", info.email)
}
