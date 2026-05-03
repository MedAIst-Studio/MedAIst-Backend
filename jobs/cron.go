package jobs

import (
	"context"
	"log"
	"time"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

func SetupCronJobs(
	notifRepo domain.NotificationRepository,
	notifService domain.NotificationService,
	fileRepo domain.FileRepository,
	fileStorage domain.FileStorageService,
) (*cron.Cron, error) {
	loc, err := time.LoadLocation("Europe/Istanbul")
	if err != nil {
		return nil, err
	}

	c := cron.New(cron.WithLocation(loc))

	// Morning Motivation at 09:00
	_, err = c.AddFunc("0 9 * * *", func() {
		ctx := context.Background()
		userIDs, err := notifRepo.GetUsersForMorningMotivation(ctx)
		if err != nil {
			log.Printf("Cron: failed to get users for morning motivation: %v", err)
			return
		}

		log.Printf("Cron: Sending morning motivation to %d users", len(userIDs))
		for _, uid := range userIDs {
			err := notifService.SendNotification(
				ctx,
				uid,
				domain.NotifTypeMotivation,
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
		userIDs, err := notifRepo.GetLazyUsersForEveningReminder(ctx)
		if err != nil {
			log.Printf("Cron: failed to get users for evening reminder: %v", err)
			return
		}

		log.Printf("Cron: Sending evening reminder to %d lazy users", len(userIDs))
		for _, uid := range userIDs {
			err := notifService.SendNotification(
				ctx,
				uid,
				domain.NotifTypeReminder,
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

	// Image cleanup at 03:00 — delete unsaved images older than 7 days
	_, err = c.AddFunc("0 3 * * *", func() {
		cleanupExpiredImages(fileRepo, fileStorage)
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

// cleanupExpiredImages deletes unsaved image files that are older than 7 days.
func cleanupExpiredImages(fileRepo domain.FileRepository, fileStorage domain.FileStorageService) {
	ctx := context.Background()
	cutoff := time.Now().AddDate(0, 0, -7)

	files, err := fileRepo.GetExpiredUnsavedImages(ctx, cutoff)
	if err != nil {
		log.Printf("Cron: failed to get expired unsaved images: %v", err)
		return
	}

	if len(files) == 0 {
		log.Println("Cron: No expired unsaved images to clean up")
		return
	}

	log.Printf("Cron: Cleaning up %d expired unsaved images", len(files))

	// Collect paths and IDs
	paths := make([]string, 0, len(files))
	ids := make([]uuid.UUID, 0, len(files))
	for _, f := range files {
		paths = append(paths, f.Path)
		ids = append(ids, f.Id)
	}

	// Delete from storage
	if err := fileStorage.DeleteFiles(ctx, paths); err != nil {
		log.Printf("Cron: failed to delete files from storage: %v", err)
		// Continue with DB cleanup even if storage deletion partially fails
	}

	// Delete from database
	if err := fileRepo.DeleteFilesByIDs(ctx, ids); err != nil {
		log.Printf("Cron: failed to delete expired images from database: %v", err)
		return
	}

	log.Printf("Cron: Successfully cleaned up %d expired unsaved images", len(files))
}
