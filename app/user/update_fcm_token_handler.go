package user

import (
	"context"
	"net/http"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

type UpdateFCMTokenRequest struct {
	FCMToken string `json:"fcm_token" validate:"required"`
}

type UpdateFCMTokenResponse struct{}

type UpdateFCMTokenHandler struct {
	repo     Repository
	validate domain.Validator
}

func NewUpdateFCMTokenHandler(repo Repository, validate domain.Validator) *UpdateFCMTokenHandler {
	return &UpdateFCMTokenHandler{repo: repo, validate: validate}
}

// @Summary		Update FCM Token
// @Description	Register or update the user's FCM push notification token
// @Tags			User
// @Accept			json
// @Produce		json
// @Param			request	body	UpdateFCMTokenRequest	true	"FCM Token Request"
// @Security		BearerAuth
// @Success		204
// @Failure		400
// @Failure		401
// @Failure		500
// @Router			/users/fcm-token [patch]
func (h *UpdateFCMTokenHandler) Handle(ctx context.Context, req *UpdateFCMTokenRequest) (*UpdateFCMTokenResponse, int, error) {
	if err := h.validate.Validate(req); err != nil {
		return nil, http.StatusBadRequest, domain.ErrInvalidRequest
	}

	userId := domain.GetUserID(ctx)

	if err := h.repo.UpdateFCMToken(ctx, userId, req.FCMToken); err != nil {
		return nil, http.StatusInternalServerError, domain.ErrInternalServer
	}

	return nil, http.StatusNoContent, nil
}
