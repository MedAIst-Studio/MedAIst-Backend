package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/MedAIst-Studio/MedAIist-Backend/app/user"
	"github.com/MedAIst-Studio/MedAIist-Backend/domain"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

func (r *Repository) GetUserById(ctx context.Context, id uuid.UUID) (*user.GetCurrentUserResponse, error) {
	row := r.db.QueryRowContext(ctx, "SELECT id, username, email, created_at, preferences FROM users WHERE id = $1", id)

	if err := row.Err(); err != nil {
		return nil, err
	}

	var u user.GetCurrentUserResponse
	var prefsJSON sql.NullString

	err := row.Scan(&u.Id, &u.Username, &u.Email, &u.CreatedAt, &prefsJSON)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	if prefsJSON.Valid {
		var prefs domain.UserPreferences
		if err := json.Unmarshal([]byte(prefsJSON.String), &prefs); err != nil {
			return nil, fmt.Errorf("failed to parse preferences: %w", err)
		}
		u.Preferences = &prefs
	}

	return &u, nil
}

func (r *Repository) UpdateUserPreferences(ctx context.Context, id uuid.UUID, preferences *domain.UserPreferences) error {
	prefsJSON, err := json.Marshal(preferences)
	if err != nil {
		return fmt.Errorf("failed to marshal preferences: %w", err)
	}

	_, err = r.db.ExecContext(ctx,
		"UPDATE users SET preferences = $1 WHERE id = $2",
		prefsJSON, id)
	return err
}

func (r *Repository) GetUserFCMTokenAndPreferences(ctx context.Context, id uuid.UUID) (string, *domain.UserPreferences, error) {
	row := r.db.QueryRowContext(ctx, "SELECT fcm_token, preferences FROM users WHERE id = $1", id)

	var fcmToken sql.NullString
	var prefsJSON sql.NullString

	err := row.Scan(&fcmToken, &prefsJSON)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil, domain.ErrUserNotFound
		}
		return "", nil, err
	}

	var prefs domain.UserPreferences
	if prefsJSON.Valid {
		if err := json.Unmarshal([]byte(prefsJSON.String), &prefs); err != nil {
			return "", nil, fmt.Errorf("failed to parse preferences: %w", err)
		}
	}

	token := ""
	if fcmToken.Valid {
		token = fcmToken.String
	}

	return token, &prefs, nil
}

func (r *Repository) UpdateFCMToken(ctx context.Context, id uuid.UUID, token string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE users SET fcm_token = $1 WHERE id = $2",
		token, id)
	return err
}

func (r *Repository) DeleteAccount(ctx context.Context, id uuid.UUID) ([]*domain.File, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var files []*domain.File
	rows, err := tx.QueryContext(ctx, "SELECT id, path FROM files WHERE user_id = $1", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var f domain.File
		if err := rows.Scan(&f.Id, &f.Path); err != nil {
			return nil, err
		}
		files = append(files, &f)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM users WHERE id = $1", id); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return files, nil
}

func (r *Repository) UpdateUsername(ctx context.Context, id uuid.UUID, username string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE users SET username = $1 WHERE id = $2",
		username, id)
	return err
}

func (r *Repository) CheckEmail(ctx context.Context, email string) error {
	row := r.db.QueryRowContext(ctx, "SELECT id FROM users WHERE email = $1", email)
	if err := row.Err(); err != nil {
		return err
	}
	var id int
	err := row.Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return errors.New("email already exists")
		}
		fmt.Printf("error: %v\n", err)
		return err
	}
	return errors.New("email already exists")
}

func (r *Repository) UpdateAccount(ctx context.Context, id uuid.UUID, username string) error {

	_, err := r.db.ExecContext(ctx,
		"UPDATE users SET username = $1 WHERE id = $2",
		username, id)
	return err
}

func (r *Repository) GetUserOnlyHavingPasswordById(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	row := r.db.QueryRowContext(ctx, "SELECT password FROM users WHERE id = $1", id)

	if err := row.Err(); err != nil {
		return nil, err
	}

	var user domain.User

	err := row.Scan(&user.Password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *Repository) ChangePassword(ctx context.Context, user *domain.User) error {
	_, err := r.db.ExecContext(ctx, "UPDATE users SET password = $1 WHERE id = $2", user.Password, user.Id)
	return err
}

func (r *Repository) EmailExists(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", email).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (r *Repository) ResetPasswordByEmail(ctx context.Context, email, newPassword string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE users SET password = $1 WHERE email = $2", newPassword, email)
	return err
}

func (r *Repository) VerifyEmail(ctx context.Context, email string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users 
		SET is_email_verified = true 
		WHERE email = $1 AND is_email_verified = false
	`, email)

	return err
}

func (r *Repository) GetUserNameAndEmailByIdForSendingVerificationEmail(ctx context.Context, id uuid.UUID) (string, string, error) {
	row := r.db.QueryRowContext(ctx, "SELECT username, email FROM users WHERE id = $1 AND is_email_verified = false", id)

	if err := row.Err(); err != nil {
		return "", "", err
	}

	var username, email string
	err := row.Scan(&username, &email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", domain.ErrEmailAlreadyVerified
		}
		return "", "", err
	}
	return username, email, nil
}
