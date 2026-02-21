package user

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

// DTOs

type GetCurrentUserRequest struct{}
type GetCurrentUserResponse struct {
	Id              string                  `json:"id"`
	Username        string                  `json:"username"`
	Email           string                  `json:"email"`
	IsEmailVerified bool                    `json:"isEmailVerified"`
	CreatedAt       time.Time               `json:"createdAt"`
	Preferences     *domain.UserPreferences `json:"preferences,omitempty"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required"`
}
type ChangePasswordResponse struct{}

type DeleteAccountRequest struct{}
type DeleteAccountResponse struct{}

type UpdateUsernameRequest struct {
	Username string `json:"username,omitempty" validate:"required"`
}
type UpdateUsernameResponse struct{}

type UpdateFCMTokenRequest struct {
	FCMToken string `json:"fcm_token" validate:"required"`
}
type UpdateFCMTokenResponse struct{}

type UpdatePreferencesRequest struct {
	Theme         string `json:"theme"`
	Notifications struct {
		Motivation bool `json:"motivation"`
		Reminders  bool `json:"reminders"`
		Updates    bool `json:"updates"`
	} `json:"notifications"`
}
type UpdatePreferencesResponse struct{}

type GetStreakRequest struct{}
type GetStreakResponse struct {
	CurrentStreak     int      `json:"current_streak"`
	MaxStreak         int      `json:"max_streak"`
	HasPlayedToday    bool     `json:"has_played_today"`
	RecentActiveDates []string `json:"recent_active_dates"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}
type ForgotPasswordResponse struct{}

type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"newPassword" validate:"required"`
}
type ResetPasswordResponse struct{}

type VerifyEmailRequest struct {
	Token string `json:"token" validate:"required"`
}
type VerifyEmailResponse struct{}

// Combined Handler

type UserHandler struct {
	userService   domain.UserService
	streakService domain.StreakService
	validator     domain.Validator
}

func NewUserHandler(userService domain.UserService, streakService domain.StreakService, validator domain.Validator) *UserHandler {
	return &UserHandler{
		userService:   userService,
		streakService: streakService,
		validator:     validator,
	}
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
func (h *UserHandler) HandleGetCurrentUser(ctx context.Context, req *GetCurrentUserRequest) (*GetCurrentUserResponse, int, error) {
	userId := domain.GetUserID(ctx)

	user, err := h.userService.GetUserById(ctx, userId)
	if err != nil {
		if err == domain.ErrUserNotFound {
			return nil, http.StatusNotFound, domain.ErrUserNotFound
		}
		return nil, http.StatusInternalServerError, err
	}

	return &GetCurrentUserResponse{
		Id:              user.Id.String(),
		Username:        user.Username,
		Email:           user.Email,
		IsEmailVerified: user.IsEmailVerified,
		CreatedAt:       user.CreatedAt,
		Preferences:     user.Preferences,
	}, http.StatusOK, nil
}

// @Summary		Change User Password
// @Description	Change the password of a user
// @Tags			User
// @Accept			json
// @Produce		json
// @Param			request	body	ChangePasswordRequest	true	"Change User Password Request"
// @Security		BearerAuth
// @Success		204
// @Failure		400
// @Failure		401
// @Failure		403
// @Failure		404
// @Failure		500
// @Router			/users/password [patch]
func (h *UserHandler) HandleChangePassword(ctx context.Context, req *ChangePasswordRequest) (*ChangePasswordResponse, int, error) {
	if err := h.validator.Validate(req); err != nil {
		return nil, http.StatusBadRequest, domain.ErrInvalidRequest
	}

	userId := domain.GetUserID(ctx)
	if err := h.userService.ChangePassword(ctx, userId, req.OldPassword, req.NewPassword); err != nil {
		if err == domain.ErrUserNotFound {
			return nil, http.StatusNotFound, domain.ErrUserNotFound
		} else if err == domain.ErrInvalidCredentials || err == domain.ErrPasswordTooShort {
			return nil, http.StatusBadRequest, err
		}
		return nil, http.StatusInternalServerError, err
	}

	return nil, http.StatusNoContent, nil
}

// @Summary		Delete User Account
// @Description	Delete a user's account
// @Tags			User
// @Security		BearerAuth
// @Success		204
// @Failure		400
// @Failure		401
// @Failure		500
// @Router			/users/account [delete]
func (h *UserHandler) HandleDeleteAccount(ctx context.Context, req *DeleteAccountRequest) (*DeleteAccountResponse, int, error) {
	userId := domain.GetUserID(ctx)
	if err := h.userService.DeleteAccount(ctx, userId); err != nil {
		return nil, http.StatusInternalServerError, domain.ErrInternalServer
	}
	return nil, http.StatusNoContent, nil
}

// @Summary		Update User Account
// @Description	Update a user's account information
// @Tags			User
// @Accept			json
// @Produce		json
// @Param			request	body	UpdateUsernameRequest	true	"Update User Account Request"
// @Security		BearerAuth
// @Success		204
// @Failure		400
// @Failure		401
// @Failure		500
// @Router			/users/account [patch]
func (h *UserHandler) HandleUpdateUsername(ctx context.Context, req *UpdateUsernameRequest) (*UpdateUsernameResponse, int, error) {
	if err := h.validator.Validate(req); err != nil {
		return nil, http.StatusBadRequest, domain.ErrInvalidRequest
	}
	userId := domain.GetUserID(ctx)
	if err := h.userService.UpdateUsername(ctx, userId, req.Username); err != nil {
		fmt.Println("Error updating account:", err)
		return nil, http.StatusInternalServerError, domain.ErrInternalServer
	}
	return nil, http.StatusNoContent, nil
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
func (h *UserHandler) HandleUpdateFCMToken(ctx context.Context, req *UpdateFCMTokenRequest) (*UpdateFCMTokenResponse, int, error) {
	if err := h.validator.Validate(req); err != nil {
		return nil, http.StatusBadRequest, domain.ErrInvalidRequest
	}

	userId := domain.GetUserID(ctx)
	if err := h.userService.UpdateFCMToken(ctx, userId, req.FCMToken); err != nil {
		return nil, http.StatusInternalServerError, domain.ErrInternalServer
	}

	return nil, http.StatusNoContent, nil
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
func (h *UserHandler) HandleUpdatePreferences(ctx context.Context, req *UpdatePreferencesRequest) (*UpdatePreferencesResponse, int, error) {
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

	if err := h.userService.UpdatePreferences(ctx, userId, preferences); err != nil {
		return nil, http.StatusInternalServerError, domain.ErrInternalServer
	}

	return nil, http.StatusNoContent, nil
}

// @Summary		Get Usage Stats (Streak)
// @Description	Get the current user's streak and usage statistics
// @Tags			User
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Success		200	{object}	GetStreakResponse
// @Failure		401
// @Failure		500
// @Router			/users/usage-stats [get]
func (h *UserHandler) HandleGetStreak(ctx context.Context, req *GetStreakRequest) (*GetStreakResponse, int, error) {
	userID := domain.GetUserID(ctx)

	stats, err := h.streakService.GetStreak(ctx, userID)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	return &GetStreakResponse{
		CurrentStreak:     stats.CurrentStreak,
		MaxStreak:         stats.MaxStreak,
		HasPlayedToday:    stats.HasPlayedToday,
		RecentActiveDates: stats.RecentActiveDates,
	}, http.StatusOK, nil
}

// @Summary		Forgot Password
// @Description	It sends  a password reset link to the user's email address.
// @Tags			User
// @Accept			json
// @Produce		json
// @Param			request	body	ForgotPasswordRequest	true	"Forgot Password Request"
// @Success		204
// @Failure		400
// @Failure		401
// @Failure		500
// @Router			/users/forgot-password [post]
func (h *UserHandler) HandleForgotPassword(ctx context.Context, req *ForgotPasswordRequest) (*ForgotPasswordResponse, int, error) {
	if err := h.validator.Validate(req); err != nil {
		return nil, http.StatusBadRequest, domain.ErrInvalidRequest
	}

	if err := h.userService.ForgotPassword(ctx, req.Email); err != nil {
		return nil, http.StatusInternalServerError, domain.ErrInternalServer
	}

	return nil, http.StatusNoContent, nil
}

// @Summary		Reset Password
// @Description	Resets the password completely for the specific user
// @Tags			User
// @Accept			json
// @Produce		json
// @Param			request	body	ResetPasswordRequest	true	"Reset Password Request"
// @Success		204
// @Failure		400
// @Failure		401
// @Failure		500
// @Router			/users/reset-password [post]
func (h *UserHandler) HandleResetPassword(ctx context.Context, req *ResetPasswordRequest) (*ResetPasswordResponse, int, error) {
	if err := h.validator.Validate(req); err != nil {
		return nil, http.StatusBadRequest, domain.ErrInvalidRequest
	}

	if err := h.userService.ResetPassword(ctx, req.Token, req.NewPassword); err != nil {
		return nil, http.StatusInternalServerError, domain.ErrInternalServer
	}

	return nil, http.StatusNoContent, nil
}
