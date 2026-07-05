package job

import (
	"log"
	"os"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/models"
)

func permanentDeleteAttachments(db *gorm.DB) {
	var attachments []models.Attachment
	if err := db.Unscoped().
		Where("deleted_at IS NOT NULL AND scheduled_delete_at IS NOT NULL AND scheduled_delete_at <= NOW()").
		Find(&attachments).Error; err != nil {
		log.Printf("[job] Failed to query expired attachments: %v", err)
		return
	}

	for _, att := range attachments {
		if err := os.Remove(att.FileURL); err != nil && !os.IsNotExist(err) {
			log.Printf("[job] Failed to delete file %s (will retry next cycle): %v", att.FileURL, err)
			continue
		}

		if err := db.Unscoped().Where("id = ?", att.ID).Delete(&models.Attachment{}).Error; err != nil {
			log.Printf("[job] Failed to hard delete attachment %s from DB: %v", att.ID, err)
			continue
		}

		log.Printf("[job] Permanently deleted attachment %s (%s)", att.ID, att.FileName)
	}
}

func startDailyPermanentDelete(db *gorm.DB) {
	permanentDeleteAttachments(db)

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("[job] Running daily permanent delete for attachments...")
		permanentDeleteAttachments(db)
	}
}
