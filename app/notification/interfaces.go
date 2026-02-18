package notification

import (
	"context"

	"firebase.google.com/go/v4/messaging"
	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

const (
	NotifTypeMotivation   = "motivation"
	NotifTypeReminder     = "reminder"
	NotifTypeAnnouncement = "update"
)

type Repository interface {
	GetUserFCMTokenAndPreferences(ctx context.Context, id uuid.UUID) (string, *domain.UserPreferences, error)
}

type NotificationService struct {
	fcmClient *messaging.Client
	repo      Repository
}

func NewNotificationService(fcmClient *messaging.Client, repo Repository) *NotificationService {
	return &NotificationService{
		fcmClient: fcmClient,
		repo:      repo,
	}
}
