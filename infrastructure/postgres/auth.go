package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/MedAIst-Studio/MedAIist-Backend/app/auth"
	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/lib/pq"
)

func (r *Repository) CreateSignupData(ctx context.Context, signupData *auth.SignupData) error {
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

	return tx.Commit()
}

func (r *Repository) createUser(tx *sql.Tx, user *domain.User) error {
	_, err := tx.ExecContext(context.Background(),
		"INSERT INTO users (id, username, email, password,  created_at) VALUES ($1, $2, $3, $4, $5, $6)",
		user.Id, user.Username, user.Email, user.Password, user.CreatedAt)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return domain.ErrEmailAlreadyExists
		}
		return err
	}
	return nil
}

func (r *Repository) createRefreshToken(tx *sql.Tx, record *domain.RefreshToken) error {
	_, err := tx.ExecContext(context.Background(),
		"INSERT INTO refresh_tokens (id, user_id, token, expires_at, created_at) VALUES ($1, $2, $3, $4, $5)",
		record.Id, record.UserID, record.Token, record.ExpiresAt, record.CreatedAt)
	return err
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

func (r *Repository) SaveRefreshToken(ctx context.Context, record *domain.RefreshToken) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO refresh_tokens (id, user_id, token, expires_at, created_at) VALUES ($1, $2, $3, $4, $5)",
		record.Id, record.UserID, record.Token, record.ExpiresAt, record.CreatedAt)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) UpsertRefreshToken(ctx context.Context, record *domain.RefreshToken) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO refresh_tokens (id, user_id, token, expires_at, created_at) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (user_id) DO UPDATE SET token = EXCLUDED.token, expires_at = EXCLUDED.expires_at, created_at = EXCLUDED.created_at",
		record.Id, record.UserID, record.Token, record.ExpiresAt, record.CreatedAt)
	if err != nil {
		return err
	}
	return nil
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
