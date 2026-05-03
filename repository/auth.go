package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

func (r *Repository) CreateUser(ctx context.Context, signupData *domain.SignupData) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackTx(tx)

	if err := r.createUser(tx, signupData.User); err != nil {
		return err
	}

	if err := r.createRefreshToken(tx, signupData.RefreshToken); err != nil {
		return err
	}

	if err := r.initializeUserDefaults(tx, signupData.User.Id); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) createUser(tx *sql.Tx, user *domain.User) error {
	_, err := tx.ExecContext(context.Background(),
		"INSERT INTO users (id, username, email, password,  created_at) VALUES ($1, $2, $3, $4, $5)",
		user.Id, user.Username, user.Email, user.Password, user.CreatedAt)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return domain.ErrEmailAlreadyExists
		}
		return err
	}
	return nil
}

func (r *Repository) CreateUserByOAuth(ctx context.Context, signupData *domain.OAuthUserRegistrationData) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackTx(tx)

	if err := r.createUserByOAuth(tx, signupData.OAuthUser); err != nil {
		return err
	}

	if err := r.createRefreshToken(tx, signupData.RefreshToken); err != nil {
		return err
	}

	if err := r.initializeUserDefaults(tx, signupData.OAuthUser.Id); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) createUserByOAuth(tx *sql.Tx, user *domain.OAuthUser) error {
	_, err := tx.ExecContext(context.Background(),
		"INSERT INTO users (id, username, email, created_at) VALUES ($1, $2, $3, $4)",
		user.Id, user.Username, user.Email, user.CreatedAt)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return domain.ErrEmailAlreadyExists
		}
		return err
	}

	_, err = tx.ExecContext(context.Background(),
		"INSERT INTO oauth (id, user_id, oauth_provider, oauth_id, created_at) VALUES ($1, $2, $3, $4, $5)",
		uuid.New(), user.Id, user.Providers[0], user.OAuthId, user.CreatedAt)
	return err
}

func (r *Repository) createRefreshToken(tx *sql.Tx, record *domain.RefreshToken) error {
	_, err := tx.ExecContext(context.Background(),
		"INSERT INTO refresh_tokens (id, user_id, token, expires_at, created_at) VALUES ($1, $2, $3, $4, $5)",
		record.Id, record.UserID, record.Token, record.ExpiresAt, record.CreatedAt)
	return err
}

// initializeUserDefaults creates the free subscription and initial usage tracking rows for a new user.
func (r *Repository) initializeUserDefaults(tx *sql.Tx, userID uuid.UUID) error {
	// Create free subscription
	_, err := tx.ExecContext(context.Background(),
		`INSERT INTO user_subscriptions (user_id, plan_type, is_active, will_renew)
		 VALUES ($1, 'free', TRUE, TRUE)
		 ON CONFLICT (user_id) DO NOTHING`,
		userID)
	if err != nil {
		return fmt.Errorf("failed to create free subscription: %w", err)
	}

	// Create initial usage tracking rows for all action types
	actions := []domain.ActionType{
		domain.ActionMessage,
		domain.ActionOCR,
		domain.ActionPDFChat,
		domain.ActionFileUpload,
		domain.ActionQuiz,
	}

	for _, action := range actions {
		_, err := tx.ExecContext(context.Background(),
			`INSERT INTO usage_tracking (user_id, action_type, usage_date, usage_count, updated_at)
			 VALUES ($1, $2, CURRENT_DATE, 0, NOW())
			 ON CONFLICT (user_id, action_type, usage_date) DO NOTHING`,
			userID, string(action))
		if err != nil {
			return fmt.Errorf("failed to create usage tracking for %s: %w", action, err)
		}
	}

	return nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	row := r.db.QueryRowContext(ctx, "SELECT id, email,  password FROM users WHERE email = $1", email)

	if err := row.Err(); err != nil {
		return nil, err
	}

	var user domain.User
	err := row.Scan(&user.Id, &user.Email, &user.Password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrEmailNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *Repository) UpsertRefreshToken(ctx context.Context, record *domain.RefreshToken) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO refresh_tokens (id, user_id, token, expires_at, created_at) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (user_id) DO UPDATE SET token = EXCLUDED.token, expires_at = EXCLUDED.expires_at, created_at = EXCLUDED.created_at",
		record.Id, record.UserID, record.Token, record.ExpiresAt, record.CreatedAt)
	return err
}

func (r *Repository) DeleteRefreshToken(ctx context.Context, token string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM refresh_tokens WHERE token = $1", token)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) RefreshTokenExists(ctx context.Context, token string) (bool, error) {
	row := r.db.QueryRowContext(ctx, "SELECT 1 FROM refresh_tokens WHERE token = $1", token)
	var exists int
	err := row.Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *Repository) GetOAuthUser(ctx context.Context, email string) (*domain.OAuthUser, error) {
	query := `
			SELECT 
			u.id, 
			COALESCE(array_agg(o.oauth_provider) FILTER (WHERE o.oauth_provider IS NOT NULL), '{}') AS providers
			FROM users u
			LEFT JOIN oauth o ON u.id = o.user_id
			WHERE u.email = $1
			GROUP BY u.id;
	`

	var oauthUser domain.OAuthUser

	row := r.db.QueryRowContext(ctx, query, email)
	err := row.Scan(&oauthUser.Id, pq.Array(&oauthUser.Providers))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &domain.OAuthUser{Exists: false}, nil
		}
		return nil, err
	}

	oauthUser.Exists = true
	return &oauthUser, nil
}

func (r *Repository) AddOAuthUser(ctx context.Context, data *domain.OAuthUserRegistrationData) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackTx(tx)

	_, err = tx.ExecContext(context.Background(),
		"INSERT INTO oauth (id, user_id, oauth_provider, oauth_id) VALUES ($1, $2, $3, $4)",
		uuid.New(), data.OAuthUser.Id, data.OAuthUser.Providers[0], data.OAuthUser.OAuthId)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx,
		"INSERT INTO refresh_tokens (id, user_id, token, expires_at, created_at) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (user_id) DO UPDATE SET token = EXCLUDED.token, expires_at = EXCLUDED.expires_at, created_at = EXCLUDED.created_at",
		data.RefreshToken.Id, data.RefreshToken.UserID, data.RefreshToken.Token, data.RefreshToken.ExpiresAt, data.RefreshToken.CreatedAt)

	if err != nil {
		return err
	}

	return tx.Commit()
}
