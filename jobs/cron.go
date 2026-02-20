package jobs

import (
	"context"
	"log"

	"github.com/MedAIst-Studio/MedAIist-Backend/app/notification"
	"github.com/robfig/cron/v3"
)

func SetupCronJobs(repo notification.Repository, notifService *notification.NotificationService) (*cron.Cron, error) {
	c := cron.New()

	// Morning Motivation at 09:00
	_, err := c.AddFunc("0 9 * * *", func() {
		ctx := context.Background()
		userIDs, err := repo.GetUsersForMorningMotivation(ctx)
		if err != nil {
			log.Printf("Cron: failed to get users for morning motivation: %v", err)
			return
		}

		log.Printf("Cron: Sending morning motivation to %d users", len(userIDs))
		for _, uid := range userIDs {
			err := notifService.SendNotification(
				ctx,
				uid,
				notification.NotifTypeMotivation,
				"Günaydın!",
				"Güne harika bir başlangıç yap! Hedeflerine bir adım daha yaklaşmak için bugün neler yapacaksın?",
			)
			if err != nil {
				log.Printf("Cron: failed to send morning motivation to %v: %v", uid, err)
			}
		}
	})
	if err != nil {
		return nil, err
	}

	// Evening lazy reminder at 21:00
	_, err = c.AddFunc("0 21 * * *", func() {
		ctx := context.Background()
		userIDs, err := repo.GetLazyUsersForEveningReminder(ctx)
		if err != nil {
			log.Printf("Cron: failed to get users for evening reminder: %v", err)
			return
		}

		log.Printf("Cron: Sending evening reminder to %d lazy users", len(userIDs))
		for _, uid := range userIDs {
			err := notifService.SendNotification(
				ctx,
				uid,
				notification.NotifTypeReminder,
				"Eksik bir şeyler var...",
				"Bugün henüz bir çalışma yapmadın. Zinciri kırmamak için hemen küçük bir test çözmeye ne dersin?",
			)
			if err != nil {
				log.Printf("Cron: failed to send evening reminder to %v: %v", uid, err)
			}
		}
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}
