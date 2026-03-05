package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

// GetSubscription returns the user's current subscription.
func (r *Repository) GetSubscription(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	query := `
		SELECT id, user_id, plan_type, revenuecat_id, product_id, entitlement_id,
		       expires_at, is_active, will_renew, created_at, updated_at
		FROM user_subscriptions
		WHERE user_id = $1
	`

	var sub domain.Subscription
	var revenueCatID, productID, entitlementID sql.NullString
	var expiresAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&sub.ID, &sub.UserID, &sub.PlanType, &revenueCatID, &productID,
		&entitlementID, &expiresAt, &sub.IsActive, &sub.WillRenew, &sub.CreatedAt, &sub.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if revenueCatID.Valid {
		sub.RevenueCatID = revenueCatID.String
	}
	if productID.Valid {
		sub.ProductID = productID.String
	}
	if entitlementID.Valid {
		sub.EntitlementID = entitlementID.String
	}
	if expiresAt.Valid {
		sub.ExpiresAt = &expiresAt.Time
	}

	return &sub, nil
}

// UpsertSubscription creates or updates a user's subscription.
func (r *Repository) UpsertSubscription(ctx context.Context, sub *domain.Subscription) error {
	query := `
		INSERT INTO user_subscriptions (user_id, plan_type, revenuecat_id, product_id, entitlement_id, expires_at, is_active, will_renew, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			plan_type = EXCLUDED.plan_type,
			revenuecat_id = EXCLUDED.revenuecat_id,
			product_id = EXCLUDED.product_id,
			entitlement_id = EXCLUDED.entitlement_id,
			expires_at = EXCLUDED.expires_at,
			is_active = EXCLUDED.is_active,
			will_renew = EXCLUDED.will_renew,
			updated_at = NOW()
	`

	_, err := r.db.ExecContext(ctx, query,
		sub.UserID, sub.PlanType, nullString(sub.RevenueCatID),
		nullString(sub.ProductID), nullString(sub.EntitlementID),
		nullTime(sub.ExpiresAt), sub.IsActive, sub.WillRenew,
	)
	return err
}

// EnsureFreeSubscription creates a free subscription if none exists.
func (r *Repository) EnsureFreeSubscription(ctx context.Context, userID uuid.UUID) error {
	query := `
		INSERT INTO user_subscriptions (user_id, plan_type, is_active, will_renew)
		VALUES ($1, 'free', TRUE, TRUE)
		ON CONFLICT (user_id) DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

// GetUsageCount returns the usage count for a given user, action type, and time window.
func (r *Repository) GetUsageCount(ctx context.Context, userID uuid.UUID, actionType domain.ActionType, since time.Time) (int, error) {
	query := `
		SELECT COALESCE(SUM(usage_count), 0)
		FROM usage_tracking
		WHERE user_id = $1 AND action_type = $2 AND usage_date >= $3
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, userID, string(actionType), since).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// IncrementUsage increments the usage counter for a given user and action type for today.
func (r *Repository) IncrementUsage(ctx context.Context, userID uuid.UUID, actionType domain.ActionType) error {
	query := `
		INSERT INTO usage_tracking (user_id, action_type, usage_date, usage_count, updated_at)
		VALUES ($1, $2, CURRENT_DATE, 1, NOW())
		ON CONFLICT (user_id, action_type, usage_date) DO UPDATE SET
			usage_count = usage_tracking.usage_count + 1,
			updated_at = NOW()
	`

	_, err := r.db.ExecContext(ctx, query, userID, string(actionType))
	return err
}

// -- Helpers --

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}
