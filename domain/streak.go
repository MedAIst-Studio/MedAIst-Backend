package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type UsageStats struct {
	ID                uuid.UUID  `json:"id"`
	UserID            uuid.UUID  `json:"userId"`
	CurrentStreak     int        `json:"currentStreak"`
	MaxStreak         int        `json:"maxStreak"`
	LastActivityDate  *time.Time `json:"lastActivityDate"`
	RecentActiveDates []string   `json:"recentActiveDates"`
}

type StreakResponse struct {
	CurrentStreak     int      `json:"currentStreak"`
	MaxStreak         int      `json:"maxStreak"`
	HasPlayedToday    bool     `json:"hasPlayedToday"`
	RecentActiveDates []string `json:"recentActiveDates"`
}

type StreakRepository interface {
	GetUsageStats(ctx context.Context, userID uuid.UUID) (*UsageStats, error)
	UpsertUsageStats(ctx context.Context, stats *UsageStats) error
	EnsureUsageStats(ctx context.Context, userID uuid.UUID) error
}

type StreakService interface {
	GetStreak(ctx context.Context, userID uuid.UUID) (*StreakResponse, error)
	CheckAndUpdateStreak(ctx context.Context, userID uuid.UUID) (*StreakResponse, error)
}
