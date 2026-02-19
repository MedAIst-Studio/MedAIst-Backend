package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

func (r *Repository) GetUsageStats(ctx context.Context, userID uuid.UUID) (*domain.UsageStats, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT id, user_id, current_streak, max_streak, last_activity_date, recent_active_dates FROM usage_stats WHERE user_id = $1",
		userID,
	)

	var stats domain.UsageStats
	var lastActivityDate sql.NullTime
	var recentDatesJSON []byte

	err := row.Scan(
		&stats.ID,
		&stats.UserID,
		&stats.CurrentStreak,
		&stats.MaxStreak,
		&lastActivityDate,
		&recentDatesJSON,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	if lastActivityDate.Valid {
		stats.LastActivityDate = &lastActivityDate.Time
	}

	if recentDatesJSON != nil {
		if err := json.Unmarshal(recentDatesJSON, &stats.RecentActiveDates); err != nil {
			stats.RecentActiveDates = []string{}
		}
	} else {
		stats.RecentActiveDates = []string{}
	}

	return &stats, nil
}

func (r *Repository) UpsertUsageStats(ctx context.Context, stats *domain.UsageStats) error {
	recentDatesJSON, err := json.Marshal(stats.RecentActiveDates)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx,
		`INSERT INTO usage_stats (id, user_id, current_streak, max_streak, last_activity_date, recent_active_dates)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (user_id) DO UPDATE SET
		   current_streak = EXCLUDED.current_streak,
		   max_streak = EXCLUDED.max_streak,
		   last_activity_date = EXCLUDED.last_activity_date,
		   recent_active_dates = EXCLUDED.recent_active_dates`,
		stats.ID, stats.UserID, stats.CurrentStreak, stats.MaxStreak, stats.LastActivityDate, recentDatesJSON,
	)
	return err
}

func (r *Repository) EnsureUsageStats(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO usage_stats (id, user_id, current_streak, max_streak, recent_active_dates)
		 VALUES ($1, $2, 0, 0, '[]')
		 ON CONFLICT (user_id) DO NOTHING`,
		uuid.New(), userID,
	)
	return err
}
