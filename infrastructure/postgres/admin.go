package postgres

import (
	"context"
	"database/sql"

	"github.com/MedAIst-Studio/MedAIist-Backend/app/admin"
)

func (r *Repository) GetAllUsers(ctx context.Context) ([]admin.UserInfo, error) {
	query := `
		SELECT u.username, u.email, o.oauth_provider
		FROM users u
		LEFT JOIN oauth o ON u.id = o.user_id
		ORDER BY u.created_at DESC;
    `

	rows, err := r.db.QueryContext(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []admin.UserInfo
	for rows.Next() {
		var info admin.UserInfo
		var provider sql.NullString

		if err := rows.Scan(&info.Username, &info.Email, &provider); err != nil {
			return nil, err
		}
		if provider.Valid {
			info.Provider = &provider.String
		} else {
			info.Provider = nil
		}
		results = append(results, info)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return results, nil

}
