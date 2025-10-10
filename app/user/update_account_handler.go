package user

import (
	"context"
	"fmt"
	"net/http"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

type UpdateUsernameRequest struct {
	Username string `json:"username,omitempty" validate:"required"`
	Address  string `json:"address,omitempty" validate:"required"`
}

type UpdateUsernameResponse struct{}

type UpdateUsernameService struct {
	repo     Repository
	validate domain.Validator
}

func NewUpdateUsernameHandler(repo Repository, validate domain.Validator) *UpdateUsernameService {
	return &UpdateUsernameService{repo: repo, validate: validate}
}

// Handle processes the request to update a user's account information.
//
//	@Summary		Update User Account
//	@Description	Update a user's account information
//	@Tags			User
//	@Accept			json
//	@Produce		json
//	@Param			request	body	UpdateUsernameRequest	true	"Update User Account Request"
//	@Security		BearerAuth
//	@Success		204
//	@Failure		400
//	@Failure		401
//	@Failure		500
//	@Router			/users/account [patch]
func (h *UpdateUsernameService) Handle(ctx context.Context, req *UpdateUsernameRequest) (*UpdateUsernameResponse, int, error) {
	if err := h.validate.Validate(req); err != nil {
		return nil, http.StatusBadRequest, domain.ErrInvalidRequest
	}
	userId := domain.GetUserID(ctx)
	if err := h.repo.UpdateUsername(ctx, userId, req.Username); err != nil {
		fmt.Println("Error updating account:", err)
		return nil, http.StatusInternalServerError, domain.ErrInternalServer
	}
	return nil, http.StatusNoContent, nil
}
