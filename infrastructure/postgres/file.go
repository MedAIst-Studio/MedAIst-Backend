package postgres

import (
	"context"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

func (r *Repository) SaveFileInformation(ctx context.Context, file *domain.File) error {
	query := `INSERT INTO files (id, user_id, title, path, created_at) VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.ExecContext(ctx, query, file.Id, file.UserId, file.Name, file.Path, file.CreatedAt)
	if err != nil {
		return domain.ErrInternalServer
	}
	return nil
}
