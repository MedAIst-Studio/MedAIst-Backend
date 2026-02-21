package services

import (
	"context"
	"time"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type StreakService struct {
	repo domain.StreakRepository
}

func NewStreakService(repo domain.StreakRepository) *StreakService {
	return &StreakService{repo: repo}
}

func (s *StreakService) GetStreak(ctx context.Context, userID uuid.UUID) (*domain.StreakResponse, error) {
	stats, err := s.repo.GetUsageStats(ctx, userID)
	if err != nil {
		return nil, err
	}

	hasPlayedToday := false
	if stats.LastActivityDate != nil {
		today := time.Now().Truncate(24 * time.Hour)
		lastDate := stats.LastActivityDate.Truncate(24 * time.Hour)
		hasPlayedToday = lastDate.Equal(today)
	}

	return &domain.StreakResponse{
		CurrentStreak:     stats.CurrentStreak,
		MaxStreak:         stats.MaxStreak,
		HasPlayedToday:    hasPlayedToday,
		RecentActiveDates: stats.RecentActiveDates,
	}, nil
}

func (s *StreakService) CheckAndUpdateStreak(ctx context.Context, userID uuid.UUID) (*domain.StreakResponse, error) {
	if err := s.repo.EnsureUsageStats(ctx, userID); err != nil {
		return nil, err
	}

	stats, err := s.repo.GetUsageStats(ctx, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	today := now.Truncate(24 * time.Hour)

	if stats.LastActivityDate != nil {
		lastDate := stats.LastActivityDate.Truncate(24 * time.Hour)
		if lastDate.Equal(today) {
			return &domain.StreakResponse{
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

	if err := s.repo.UpsertUsageStats(ctx, stats); err != nil {
		return nil, err
	}

	return &domain.StreakResponse{
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
