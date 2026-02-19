package user

import (
	"context"
	"net/http"
	"time"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

type StreakHandler struct {
	streakRepo domain.StreakRepository
}

func NewStreakHandler(streakRepo domain.StreakRepository) *StreakHandler {
	return &StreakHandler{streakRepo: streakRepo}
}

// --- GET /users/usage-stats ---

type GetStreakRequest struct{}

type GetStreakResponse struct {
	CurrentStreak     int      `json:"current_streak"`
	MaxStreak         int      `json:"max_streak"`
	HasPlayedToday    bool     `json:"has_played_today"`
	RecentActiveDates []string `json:"recent_active_dates"`
}

type GetStreakHandler struct {
	streakRepo domain.StreakRepository
}

func NewGetStreakHandler(streakRepo domain.StreakRepository) *GetStreakHandler {
	return &GetStreakHandler{streakRepo: streakRepo}
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
func (h *GetStreakHandler) Handle(ctx context.Context, req *GetStreakRequest) (*GetStreakResponse, int, error) {
	userID := domain.GetUserID(ctx)

	if err := h.streakRepo.EnsureUsageStats(ctx, userID); err != nil {
		return nil, http.StatusInternalServerError, domain.ErrInternalServer
	}

	stats, err := h.streakRepo.GetUsageStats(ctx, userID)
	if err != nil {
		return nil, http.StatusInternalServerError, domain.ErrInternalServer
	}

	hasPlayedToday := false
	if stats.LastActivityDate != nil {
		today := time.Now().Truncate(24 * time.Hour)
		lastDate := stats.LastActivityDate.Truncate(24 * time.Hour)
		hasPlayedToday = lastDate.Equal(today)
	}

	return &GetStreakResponse{
		CurrentStreak:     stats.CurrentStreak,
		MaxStreak:         stats.MaxStreak,
		HasPlayedToday:    hasPlayedToday,
		RecentActiveDates: stats.RecentActiveDates,
	}, http.StatusOK, nil
}
