package services

import (
	"context"
	"log"

	"firebase.google.com/go/v4/messaging"
	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type NotificationService struct {
	fcmClient *messaging.Client
	repo      domain.NotificationRepository
}

func NewNotificationService(fcmClient *messaging.Client, repo domain.NotificationRepository) *NotificationService {
	return &NotificationService{
		fcmClient: fcmClient,
		repo:      repo,
	}
}

func (s *NotificationService) SendNotification(ctx context.Context, userID uuid.UUID, notifType string, title string, body string) error {
	fcmToken, prefs, err := s.repo.GetUserFCMTokenAndPreferences(ctx, userID)
	if err != nil {
		return err
	}

	if fcmToken == "" {
		return nil
	}

	allowed := false
	switch notifType {
	case domain.NotifTypeMotivation:
		allowed = prefs.Notifications.Motivation
	case domain.NotifTypeReminder:
		allowed = prefs.Notifications.Reminders
	case domain.NotifTypeAnnouncement:
		allowed = prefs.Notifications.Updates
	default:
		allowed = true
	}

	if !allowed {
		log.Printf("User (%s) %s notification is disabled. Not sent.", userID, notifType)
		return nil
	}

	message := &messaging.Message{
		Token: fcmToken,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: map[string]string{
			"type": notifType,
			"url":  "scope://home",
		},
	}

	_, err = s.fcmClient.Send(context.Background(), message)
	return err
}
