package notification

import (
	"context"
	"log"

	"firebase.google.com/go/v4/messaging"
	"github.com/google/uuid"
)

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
	case NotifTypeMotivation:
		allowed = prefs.Notifications.Motivation
	case NotifTypeReminder:
		allowed = prefs.Notifications.Reminders
	case NotifTypeAnnouncement:
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
