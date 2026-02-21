package domain

import (
	"context"

	"firebase.google.com/go/v4/messaging"
	"github.com/google/uuid"
)

// ---------- Notification Constants ----------

const (
	NotifTypeMotivation   = "motivation"
	NotifTypeReminder     = "reminder"
	NotifTypeAnnouncement = "update"
)

// ---------- Notification Interfaces ----------

type NotificationRepository interface {
	GetUserFCMTokenAndPreferences(ctx context.Context, id uuid.UUID) (string, *UserPreferences, error)
	GetUsersForMorningMotivation(ctx context.Context) ([]uuid.UUID, error)
	GetLazyUsersForEveningReminder(ctx context.Context) ([]uuid.UUID, error)
}

type NotificationService interface {
	SendNotification(ctx context.Context, userID uuid.UUID, notifType string, title string, body string) error
}

// ---------- FCM Client ----------

type FCMClient interface {
	Send(ctx context.Context, message *messaging.Message) (string, error)
}
