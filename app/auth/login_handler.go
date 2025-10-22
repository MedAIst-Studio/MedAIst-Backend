package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	RefreshToken string `json:"refreshToken"`
	AccessToken  string `json:"accessToken"`
}

type LoginConfig struct {
	Repo         Repository
	TokenService TokenService
	Validator    domain.Validator
	Logger       domain.Logger
}

type LoginHandler struct {
	secure    bool
	repo      Repository
	ts        TokenService
	validator domain.Validator
	logger    domain.Logger
}

func NewLoginHandler(config *LoginConfig) *LoginHandler {
	return &LoginHandler{
		repo:      config.Repo,
		ts:        config.TokenService,
		validator: config.Validator,
		logger:    config.Logger,
	}
}

// @Summary		Login
// @Description	Login a user or admin
// @Tags			Auth
// @Accept			json
// @Produce		json
// @Param			request	body		LoginRequest	true	"Login Request"
// @Success		200		{object}	LoginResponse
// @Failure		400
// @Failure		401
// @Failure		404
// @Failure		500
// @Router			/auth/login [post]
func (h *LoginHandler) Handle(ctx context.Context, req *LoginRequest) (*LoginResponse, int, error) {
	if err := h.validator.Validate(req); err != nil {
		return nil, http.StatusBadRequest, domain.ErrInvalidRequest
	}

	user, err := h.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {

		if errors.Is(err, domain.ErrEmailNotFound) {
			return nil, http.StatusNotFound, domain.ErrEmailNotFound
		}
		h.logger.Error("error while getting user by email: ", err)
		return nil, http.StatusInternalServerError, domain.ErrInternalServer
	}

	if err = user.ValidatePassword(req.Password); err != nil {
		if !errors.Is(err, domain.ErrInternalServer) {
			return nil, http.StatusBadRequest, err
		}
		h.logger.Error("error while validating password: ", err)
		return nil, http.StatusInternalServerError, domain.ErrInternalServer
	}

	accessToken, err := h.ts.GenerateAuthAccessToken(user.Id.String())
	if err != nil {
		fmt.Println("error while generating access token: ", err)
		return nil, http.StatusInternalServerError, domain.ErrInternalServer
	}

	refreshToken, err := h.ts.GenerateAuthRefreshToken(user.Id.String())
	if err != nil {
		fmt.Println("error while generating refresh token: ", err)
		return nil, http.StatusInternalServerError, domain.ErrInternalServer
	}

	if err := h.repo.UpsertRefreshToken(ctx, domain.NewRefreshToken(
		user.Id,
		refreshToken,
	)); err != nil {
		h.logger.Error("error while saving refresh token: ", err)
		return nil, http.StatusInternalServerError, domain.ErrInternalServer
	}

	return &LoginResponse{
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
	}, http.StatusOK, nil
}
