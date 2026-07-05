package job

import (
	"log"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/models"
)

type dueTaskRow struct {
	TaskID      string
	TaskTitle   string
	DueDate     time.Time
	ProjectID   string
	WorkspaceID string
	ProjectKey  string
}

func processTaskDueSoon(db *gorm.DB) {
	log.Println("[job] Processing TASK_DUE_SOON notifications...")

	var tasks []dueTaskRow
	err := db.Raw(`
		SELECT t.id AS task_id, t.title AS task_title, t.due_date,
		       p.id AS project_id, p.workspace_id, p.key AS project_key
		FROM tasks t
		JOIN projects p ON p.id = t.project_id
		JOIN columns c ON c.id = t.column_id
		WHERE t.due_date BETWEEN NOW() AND NOW() + INTERVAL '25 hours'
		  AND t.deleted_at IS NULL
		  AND p.deleted_at IS NULL
		  AND p.is_archived = false
		  AND t.parent_id IS NULL
		  AND c.position < (
		      SELECT MAX(c2.position) FROM columns c2
		      WHERE c2.project_id = t.project_id AND c2.deleted_at IS NULL
		  )
	`).Scan(&tasks).Error
	if err != nil {
		log.Printf("[job] Failed to query due tasks: %v", err)
		return
	}

	for _, task := range tasks {
		var count int64
		db.Model(&models.TaskDueNotification{}).
			Where("task_id = ?", task.TaskID).
			Count(&count)
		if count > 0 {
			continue
		}

		var assigneeIDs []string
		db.Table("task_assignees").
			Where("task_id = ?", task.TaskID).
			Pluck("user_id", &assigneeIDs)
		if len(assigneeIDs) == 0 {
			continue
		}

		taskRef := task.ProjectKey + "-"
		var taskNum int64
		db.Table("tasks").Select("task_number").
			Where("id = ?", task.TaskID).Scan(&taskNum)
		taskRef += itoa(int(taskNum))

		title := "Task " + taskRef + " sắp đến hạn"
		content := "'" + task.TaskTitle + "' sẽ đến hạn trong 24 giờ nữa (" + task.DueDate.Format("2006-01-02 15:04 MST") + ")."
		refURL := "/workspaces/" + task.WorkspaceID + "/projects/" + task.ProjectID + "/tasks/" + task.TaskID

		now := time.Now()
		notif := models.Notification{
			ActorID:      nil,
			Type:         models.NotificationTypeTASKDUESOON,
			Title:        title,
			Content:      &content,
			ReferenceURL: &refURL,
			CreatedAt:    now,
		}

		db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&notif).Error; err != nil {
				return err
			}
			for _, uid := range assigneeIDs {
				nr := models.NotificationRecipient{
					NotificationID: notif.ID,
					RecipientID:    uid,
				}
				if err := tx.Create(&nr).Error; err != nil {
					return err
				}
			}
			tdn := models.TaskDueNotification{
				TaskID: task.TaskID,
				SentAt: time.Now(),
			}
			return tx.Create(&tdn).Error
		})
	}

	log.Printf("[job] TASK_DUE_SOON processed %d tasks", len(tasks))
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

func startTaskDueSoonCron(db *gorm.DB) {
	processTaskDueSoon(db)

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		processTaskDueSoon(db)
	}
}
