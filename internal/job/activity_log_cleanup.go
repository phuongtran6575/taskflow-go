package job

import (
	"log"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/models"
)

func processActivityLogCleanup(db *gorm.DB) {
	log.Println("[job] Running activity log cleanup...")

	type wsPlan struct {
		ID   string
		Plan string
	}
	var workspaces []wsPlan
	db.Model(&models.Workspace{}).Select("id, plan").Where("deleted_at IS NULL").Scan(&workspaces)

	planDays := map[string]int{
		"FREE":       7,
		"PRO":        90,
		"ENTERPRISE": 365,
	}

	totalDeleted := int64(0)
	for _, ws := range workspaces {
		days, ok := planDays[ws.Plan]
		if !ok {
			days = 7
		}
		before := time.Now().AddDate(0, 0, -days)

		result := db.Exec(`
			DELETE FROM activity_logs
			WHERE workspace_id = ?
			AND created_at < ?
		`, ws.ID, before)
		if result.Error != nil {
			log.Printf("[job] Activity log cleanup failed for workspace %s: %v", ws.ID, result.Error)
		} else {
			totalDeleted += result.RowsAffected
		}
	}

	log.Printf("[job] Activity log cleanup: %d rows deleted", totalDeleted)
}

func startDailyActivityLogCleanup(db *gorm.DB) {
	processActivityLogCleanup(db)

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("[job] Running daily activity log cleanup...")
		processActivityLogCleanup(db)
	}
}
