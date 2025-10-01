package auth

import (
	"context"
	"net/http"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

type LogoutRequest struct {
	Language     string `reqHeader:"X-Client-Language" validate:"required" swaggerignore:"true"`
	RefreshToken string `json:"refresh_token"`
}

type LogoutResponse struct {
}

type LogoutHandler struct {
	repo Repository
}

func NewLogoutHandler(repo Repository) *LogoutHandler {
	return &LogoutHandler{
		repo: repo,
	}
}

// Logout removes the refresh token from the database .
//
//	@Summary		Logout user
//	@Description	Removes the refresh token from the database.
//
//	@Param			X-Client-Language	header	string	true	"Client language"
//
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Success		204
//	@Failure		500
//	@Router			/auth/logout [post]
func (h *LogoutHandler) Handle(ctx context.Context, req *LogoutRequest) (*LogoutResponse, int, error) {
	if req.RefreshToken == "" {
		return nil, http.StatusBadRequest, domain.ErrInvalidRequest
	}

	if err := h.repo.DeleteRefreshToken(ctx, req.RefreshToken); err != nil {
		return nil, http.StatusInternalServerError, domain.ErrInternalServer
	}
	return nil, http.StatusNoContent, nil
}
