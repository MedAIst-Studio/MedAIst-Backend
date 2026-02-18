package user

import (
	"context"
	"net/http"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

type UpdatePreferencesRequest struct {
	Theme         string `json:"theme"`
	Notifications struct {
		Motivation bool `json:"motivation"`
		Reminders  bool `json:"reminders"`
		Updates    bool `json:"updates"`
	} `json:"notifications"`
}

type UpdatePreferencesResponse struct{}

type UpdatePreferencesHandler struct {
	repo Repository
}

func NewUpdatePreferencesHandler(repo Repository) *UpdatePreferencesHandler {
	return &UpdatePreferencesHandler{repo: repo}
}

// @Summary		Update User Preferences
// @Description	Update the current user's preferences (theme and notification settings)
// @Tags			User
// @Accept			json
// @Produce		json
// @Param			request	body	UpdatePreferencesRequest	true	"Update Preferences Request"
// @Security		BearerAuth
// @Success		204
// @Failure		400
// @Failure		401
// @Failure		500
// @Router			/users/preferences [patch]
func (h *UpdatePreferencesHandler) Handle(ctx context.Context, req *UpdatePreferencesRequest) (*UpdatePreferencesResponse, int, error) {
	userId := domain.GetUserID(ctx)

	preferences := &domain.UserPreferences{
		Theme: req.Theme,
		Notifications: domain.NotificationPreferences{
			Motivation: req.Notifications.Motivation,
			Reminders:  req.Notifications.Reminders,
			Updates:    req.Notifications.Updates,
		},
	}

	if preferences.Theme == "" {
		preferences.Theme = "system"
	}

	if err := h.repo.UpdateUserPreferences(ctx, userId, preferences); err != nil {
		return nil, http.StatusInternalServerError, domain.ErrInternalServer
	}

	return nil, http.StatusNoContent, nil
}
