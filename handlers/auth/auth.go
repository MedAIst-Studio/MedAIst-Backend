package auth

import (
	"context"
	"errors"
	"net/http"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

// ---------- DTOs ----------

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	RefreshToken string `json:"refreshToken"`
	AccessToken  string `json:"accessToken"`
}

type SignupRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
}

type SignupResponse struct {
	RefreshToken string `json:"refreshToken" swaggerignore:"true"`
	AccessToken  string `json:"accessToken" swaggerignore:"true"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type LogoutResponse struct{}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type RefreshTokenResponse struct {
	AccessToken string `json:"access_token"`
}

type ExchangeRequest struct {
	SessionKey string `json:"session"`
}

type ExchangeResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// ---------- AuthHandler AuthConfig ----------

type AuthConfig struct {
	AuthService domain.AuthService
	Validator   domain.Validator
	Logger      domain.Logger
}

type AuthHandler struct {
	authService domain.AuthService
	validator   domain.Validator
	logger      domain.Logger
}

func NewAuthHandler(config *AuthConfig) *AuthHandler {
	return &AuthHandler{
		authService: config.AuthService,
		validator:   config.Validator,
		logger:      config.Logger,
	}
}

// ---------- Handlers ----------

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
func (h *AuthHandler) HandleLogin(ctx context.Context, req *LoginRequest) (*LoginResponse, int, error) {
	if err := h.validator.Validate(req); err != nil {
		return nil, http.StatusBadRequest, domain.ErrInvalidRequest
	}

	accessToken, refreshToken, err := h.authService.Login(ctx, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrEmailNotFound) {
			return nil, http.StatusNotFound, domain.ErrEmailNotFound
		}
		if !errors.Is(err, domain.ErrInternalServer) {
			return nil, http.StatusBadRequest, err
		}
		h.logger.Error("error during login: ", err)
		return nil, http.StatusInternalServerError, domain.ErrInternalServer
	}

	return &LoginResponse{
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
	}, http.StatusOK, nil
}

// @Summary		Signup
// @Description	Signup a new user
// @Tags			Auth
// @Accept			json
// @Produce		json
// @Param			request	body		SignupRequest	true	"Signup request"
// @Success		200		{object}	SignupResponse
// @Failure		400
// @Failure		409
// @Failure		500
// @Router			/auth/signup [post]
func (h *AuthHandler) HandleSignup(ctx context.Context, req *SignupRequest) (*SignupResponse, int, error) {
	if err := h.validator.Validate(req); err != nil {
		return nil, http.StatusBadRequest, domain.ErrInvalidRequest
	}

	accessToken, refreshToken, err := h.authService.Signup(ctx, req.Username, req.Password, req.Email)
	if err != nil {
		if errors.Is(err, domain.ErrEmailAlreadyExists) {
			return nil, http.StatusConflict, domain.ErrEmailAlreadyExists
		}
		if !errors.Is(err, domain.ErrInternalServer) {
			return nil, http.StatusBadRequest, err
		}
		h.logger.Error("error during signup: ", err)
		return nil, http.StatusInternalServerError, domain.ErrInternalServer
	}

	return &SignupResponse{
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
	}, http.StatusCreated, nil
}

// Logout removes the refresh token from the database .
//
//	@Summary		Logout user
//	@Description	Removes the refresh token from the database.
//
//
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Success		204
//	@Failure		500
//	@Router			/auth/logout [post]
func (h *AuthHandler) HandleLogout(ctx context.Context, req *LogoutRequest) (*LogoutResponse, int, error) {
	if req.RefreshToken == "" {
		return nil, http.StatusBadRequest, domain.ErrInvalidRequest
	}

	if err := h.authService.Logout(ctx, req.RefreshToken); err != nil {
		return nil, http.StatusInternalServerError, domain.ErrInternalServer
	}
	return nil, http.StatusNoContent, nil
}

// @Summary		Refresh access token
// @Description	Generate a new access token using a valid refresh token.
// @Description	The API takes refresh token from req body the sends  a new access token.
// @Tags			Auth
// @Accept			json
// @Produce		json
// @Param			request	body		RefreshTokenRequest	true	"Refresh Token Request"
// @Success		200		{object}	RefreshTokenResponse
// @Failure		401
// @Failure		500
// @Router			/auth/refresh [post]
func (h *AuthHandler) HandleRefreshToken(ctx context.Context, req *RefreshTokenRequest) (*RefreshTokenResponse, int, error) {
	if req.RefreshToken == "" {
		return nil, http.StatusUnauthorized, domain.ErrMissingRefreshToken
	}

	accessToken, err := h.authService.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		if !errors.Is(err, domain.ErrInternalServer) {
			return nil, http.StatusUnauthorized, err
		}
		return nil, http.StatusInternalServerError, domain.ErrInternalServer
	}

	return &RefreshTokenResponse{AccessToken: accessToken}, http.StatusOK, nil
}

// godoc
//
//	@summary	Exchange OAuth2 tokens
//	@accept		json
//	@produce	json
//	@param		request	body		ExchangeRequest	true	"Exchange Request"
//	@success	200		{object}	ExchangeResponse
//	@failure	400
//	@failure	500
//	@router		/api/auth/exchange [post]
func (h *AuthHandler) HandleExchange(ctx context.Context, req *ExchangeRequest) (*ExchangeResponse, int, error) {
	if req.SessionKey == "" {
		return nil, 400, errors.New("session key is required")
	}

	accessToken, refreshToken, err := h.authService.ExchangeOAuthSession(ctx, req.SessionKey)
	if err != nil {
		if !errors.Is(err, domain.ErrInternalServer) {
			return nil, http.StatusBadRequest, err
		}
		return nil, http.StatusInternalServerError, err
	}

	return &ExchangeResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, 200, nil
}
