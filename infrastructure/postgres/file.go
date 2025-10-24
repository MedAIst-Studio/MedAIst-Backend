package postgres

import (
	"context"

	"github.com/MedAIst-Studio/MedAIist-Backend/app/file"
	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

func (r *Repository) SaveFileInformation(ctx context.Context, file *domain.File) error {
	query := `INSERT INTO files (id, user_id, title, size, path, created_at) VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.ExecContext(ctx, query, file.Id, file.UserId, file.Name, file.Size, file.Path, file.CreatedAt)
	if err != nil {
		return domain.ErrInternalServer
	}
	return nil
}

func (r *Repository) GetFilesByUserID(ctx context.Context, userId uuid.UUID) ([]file.File, error) {
	query := `SELECT id, title, size,  created_at FROM files WHERE user_id = $1`
	rows, err := r.db.QueryContext(ctx, query, userId)
	if err != nil {
		return nil, domain.ErrInternalServer
	}
	defer rows.Close()

	var files []file.File
	for rows.Next() {
		var f file.File
		if err := rows.Scan(&f.Id, &f.Title, &f.Size, &f.CreatedAt); err != nil {
			return nil, domain.ErrInternalServer
		}
		files = append(files, f)
	}
	return files, nil
}

func (r *Repository) DeleteFileById(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM files WHERE id = $1", id)
	if err != nil {
		return err
	}

	if rowsAffected, _ := result.RowsAffected(); rowsAffected == 0 {
		return domain.ErrFileNotFound
	}

	return nil
}

func (r *Repository) DoesUserHaveFile(ctx context.Context, userId uuid.UUID, fileId uuid.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM files WHERE id = $1 AND user_id = $2)`
	err := r.db.QueryRowContext(ctx, query, fileId, userId).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}
