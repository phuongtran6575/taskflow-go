package job

import (
	"log"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/models"
)

func processNotificationCleanup(db *gorm.DB) {
	log.Println("[job] Running notification cleanup...")

	type wsPlan struct {
		ID   string
		Plan string
	}
	var workspaces []wsPlan
	db.Model(&models.Workspace{}).Select("id, plan").Scan(&workspaces)

	planDays := map[string]int{
		"FREE":        30,
		"PRO":         90,
		"ENTERPRISE":  365,
	}

	totalDeleted := int64(0)
	for _, ws := range workspaces {
		days, ok := planDays[ws.Plan]
		if !ok {
			days = 30
		}
		before := time.Now().AddDate(0, 0, -days)

		pattern := "/workspaces/" + ws.ID + "/"
		result := db.Exec(`
			DELETE FROM notification_recipients nr
			USING notifications n
			WHERE n.id = nr.notification_id
			AND n.created_at < ?
			AND n.reference_url LIKE ?
		`, before, pattern+"%")
		if result.Error != nil {
			log.Printf("[job] Cleanup failed for workspace %s: %v", ws.ID, result.Error)
		} else {
			totalDeleted += result.RowsAffected
		}
	}

	// Workspace-level notifications (ADDED_TO_WORKSPACE) have ref URLs like /workspaces/{id}
	// They need exact match
	for _, ws := range workspaces {
		days, ok := planDays[ws.Plan]
		if !ok {
			days = 30
		}
		before := time.Now().AddDate(0, 0, -days)
		exactPattern := "/workspaces/" + ws.ID
		result := db.Exec(`
			DELETE FROM notification_recipients nr
			USING notifications n
			WHERE n.id = nr.notification_id
			AND n.created_at < ?
			AND n.reference_url = ?
		`, before, exactPattern)
		if result.Error != nil {
			log.Printf("[job] Cleanup (exact) failed for workspace %s: %v", ws.ID, result.Error)
		} else {
			totalDeleted += result.RowsAffected
		}
	}

	var orphanCount int64
	db.Raw(`
		SELECT COUNT(*) FROM notifications n
		WHERE NOT EXISTS (
			SELECT 1 FROM notification_recipients
			WHERE notification_id = n.id
		)
	`).Scan(&orphanCount)

	db.Exec(`
		DELETE FROM notifications n
		WHERE NOT EXISTS (
			SELECT 1 FROM notification_recipients
			WHERE notification_id = n.id
		)
	`)

	log.Printf("[job] Notification cleanup: %d recipients deleted, %d orphans purged", totalDeleted, orphanCount)
}

func startDailyNotificationCleanup(db *gorm.DB) {
	processNotificationCleanup(db)

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("[job] Running daily notification cleanup...")
		processNotificationCleanup(db)
	}
}
