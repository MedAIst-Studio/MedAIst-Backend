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

func CheckAndUpdateStreak(ctx context.Context, repo StreakRepository, userID uuid.UUID) (*StreakResponse, error) {
	if err := repo.EnsureUsageStats(ctx, userID); err != nil {
		return nil, err
	}

	stats, err := repo.GetUsageStats(ctx, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	today := now.Truncate(24 * time.Hour)

	if stats.LastActivityDate != nil {
		lastDate := stats.LastActivityDate.Truncate(24 * time.Hour)
		if lastDate.Equal(today) {
			return &StreakResponse{
				CurrentStreak:     stats.CurrentStreak,
				MaxStreak:         stats.MaxStreak,
				HasPlayedToday:    true,
				RecentActiveDates: stats.RecentActiveDates,
			}, nil
		}

		yesterday := today.AddDate(0, 0, -1)
		if lastDate.Equal(yesterday) {
			stats.CurrentStreak++
		} else {
			stats.CurrentStreak = 1
		}
	} else {
		stats.CurrentStreak = 1
	}

	if stats.CurrentStreak > stats.MaxStreak {
		stats.MaxStreak = stats.CurrentStreak
	}

	stats.LastActivityDate = &today

	todayStr := today.Format("2006-01-02")
	stats.RecentActiveDates = appendRecentDate(stats.RecentActiveDates, todayStr, 30)

	if err := repo.UpsertUsageStats(ctx, stats); err != nil {
		return nil, err
	}

	return &StreakResponse{
		CurrentStreak:     stats.CurrentStreak,
		MaxStreak:         stats.MaxStreak,
		HasPlayedToday:    true,
		RecentActiveDates: stats.RecentActiveDates,
	}, nil
}

func appendRecentDate(dates []string, newDate string, maxLen int) []string {
	for _, d := range dates {
		if d == newDate {
			return dates
		}
	}

	dates = append(dates, newDate)

	if len(dates) > maxLen {
		dates = dates[len(dates)-maxLen:]
	}

	return dates
}
