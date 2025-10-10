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
	row := r.db.QueryRowContext(ctx, "SELECT id, username, role, email, created_at FROM users WHERE id = $1", id)

	if err := row.Err(); err != nil {
		return nil, err
	}

	var user user.GetCurrentUserResponse

	err := row.Scan(&user.Id, &user.Username, &user.Role, &user.Email, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *Repository) DeleteAccount(ctx context.Context, id uuid.UUID) error {

	_, err := r.db.ExecContext(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return err
	}
	return nil
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

func (r *Repository) GetUserByIdForAdmin(ctx context.Context, id uuid.UUID) (*user.GetUserResponse, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, username, email, is_email_verified 
		FROM users 
		WHERE id = $1`, id)

	if err := row.Err(); err != nil {
		return nil, err
	}

	var responseUser user.GetUserResponse

	err := row.Scan(&responseUser.ID, &responseUser.Username, &responseUser.Email, &responseUser.IsEmailVerified)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	return &responseUser, nil
}

func (r *Repository) GetUsers(ctx context.Context, page, limit int) (user.GetUsersResponse, error) {
	offset := (page - 1) * limit
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, username, email 
		FROM users 
		ORDER BY created_at DESC 
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users user.GetUsersResponse
	for rows.Next() {
		var u user.User
		if err := rows.Scan(&u.Id, &u.Username, &u.Email); err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
