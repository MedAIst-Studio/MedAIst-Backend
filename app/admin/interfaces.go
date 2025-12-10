package admin

import "context"

type Repository interface {
	GetAllUsers(ctx context.Context) ([]UserInfo, error)
}
