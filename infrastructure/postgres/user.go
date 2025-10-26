package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/MedAIst-Studio/MedAIist-Backend/app/user"
	"github.com/MedAIst-Studio/MedAIist-Backend/domain"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

func (r *Repository) GetUserById(ctx context.Context, id uuid.UUID) (*user.GetCurrentUserResponse, error) {
	row := r.db.QueryRowContext(ctx, "SELECT id, username,  email, created_at FROM users WHERE id = $1", id)

	if err := row.Err(); err != nil {
		return nil, err
	}

	var user user.GetCurrentUserResponse

	err := row.Scan(&user.Id, &user.Username, &user.Email, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *Repository) DeleteAccount(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var fileIds []uuid.UUID
	rows, err := tx.QueryContext(ctx, "SELECT id FROM files WHERE user_id = $1", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var fileId uuid.UUID
		if err := rows.Scan(&fileId); err != nil {
			return nil, err
		}
		fileIds = append(fileIds, fileId)
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

	return fileIds, nil
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
