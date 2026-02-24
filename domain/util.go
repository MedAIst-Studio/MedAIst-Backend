package domain

import (
	"context"
	"os"

	"github.com/google/uuid"
)

type ContextKey string

const (
	UserIDKey ContextKey = "userID"
	TokenKey  ContextKey = "token"
)

func GetUserID(ctx context.Context) uuid.UUID {
	userID := ctx.Value(UserIDKey).(string)
	return uuid.MustParse(userID)
}

func GetUserIDFromString(userID string) uuid.UUID {
	return uuid.MustParse(userID)
}

func IsProdEnv() bool {
	return os.Getenv("ENV") == "production"
}
