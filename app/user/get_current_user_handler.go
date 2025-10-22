package user

import (
	"context"
	"net/http"
	"time"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

type GetCurrentUserRequest struct{}

type GetCurrentUserResponse struct {
	Id              string    `json:"id"`
	Username        string    `json:"username"`
	Email           string    `json:"email"`
	IsEmailVerified bool      `json:"isEmailVerified"`
	CreatedAt       time.Time `json:"createdAt"`
}

type GetCurrentUserHandler struct {
	repo Repository
}

func NewGetCurrentUserHandler(repo Repository) *GetCurrentUserHandler {
	return &GetCurrentUserHandler{repo: repo}
}

// @Summary		Get Current User
// @Description	Get the current user. Requires Bearer token authentication.
// @Tags			User
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Success		200	{object}	GetCurrentUserResponse
// @Failure		400
// @Failure		401
// @Failure		500
// @Router			/users/profile [get]
func (h *GetCurrentUserHandler) Handle(ctx context.Context, req *GetCurrentUserRequest) (*GetCurrentUserResponse, int, error) {
	userId := domain.GetUserID(ctx)

	user, err := h.repo.GetUserById(ctx, userId)
	if err != nil {
		if err == domain.ErrUserNotFound {
			return nil, http.StatusNotFound, domain.ErrUserNotFound
		}
		return nil, http.StatusInternalServerError, err
	}
	return user, http.StatusOK, nil
}
