package implement

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
	_interface "TaskFlow-Go/internal/repository/interface"
)

type notificationRepository struct{ db *gorm.DB }

func NewNotificationRepository(db *gorm.DB) _interface.NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) Create(notification *models.Notification, recipients []string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(notification).Error; err != nil {
			return err
		}
		for _, recipientID := range recipients {
			nr := models.NotificationRecipient{
				NotificationID: notification.ID,
				RecipientID:    recipientID,
			}
			if err := tx.Create(&nr).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *notificationRepository) GetByID(id string) (*models.Notification, error) {
	var n models.Notification
	err := r.db.Where("id = ?", id).First(&n).Error
	return &n, err
}

func (r *notificationRepository) GetRecipient(notificationID, recipientID string) (*models.NotificationRecipient, error) {
	var nr models.NotificationRecipient
	err := r.db.Where("notification_id = ? AND recipient_id = ?", notificationID, recipientID).First(&nr).Error
	return &nr, err
}

func (r *notificationRepository) ListWithCursor(recipientID string, isRead *bool, types []string, limit int, cursor string) (*dto.NotificationListResponse, error) {
	type row struct {
		ID           string          `gorm:"column:id"`
		Type         string          `gorm:"column:type"`
		Title        string          `gorm:"column:title"`
		Content      *string         `gorm:"column:content"`
		ReferenceURL *string         `gorm:"column:reference_url"`
		ActorID      *string         `gorm:"column:actor_id"`
		ActorName    string          `gorm:"column:actor_name"`
		ActorAvatar  *string         `gorm:"column:actor_avatar"`
		IsRead       bool            `gorm:"column:is_read"`
		ReadAt       *time.Time      `gorm:"column:read_at"`
		CreatedAt    time.Time       `gorm:"column:created_at"`
		DeletedAt    gorm.DeletedAt  `gorm:"column:deleted_at"`
	}

	query := r.db.Table("notification_recipients nr").
		Select(`
			n.id, n.type, n.title, n.content, n.reference_url,
			n.actor_id, u.full_name as actor_name, u.avatar_url as actor_avatar,
			nr.is_read, nr.read_at, n.created_at, n.deleted_at
		`).
		Joins("JOIN notifications n ON n.id = nr.notification_id").
		Joins("LEFT JOIN users u ON u.id = n.actor_id").
		Where("nr.recipient_id = ?", recipientID)

	if isRead != nil {
		query = query.Where("nr.is_read = ?", *isRead)
	}
	if len(types) > 0 {
		query = query.Where("n.type IN ?", types)
	}

	if cursor != "" {
		b, err := base64.StdEncoding.DecodeString(cursor)
		if err == nil {
			parts := strings.SplitN(string(b), ":", 2)
			if len(parts) == 2 {
				t, err := time.Parse(time.RFC3339Nano, parts[0])
				if err == nil {
					query = query.Where("(n.created_at < ? OR (n.created_at = ? AND n.id < ?))", t, t, parts[1])
				}
			}
		}
	}

	query = query.Order("n.created_at DESC, n.id DESC")

	fetchLimit := limit + 1
	var rows []row
	if err := query.Limit(fetchLimit).Scan(&rows).Error; err != nil {
		return nil, err
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	data := make([]dto.NotificationInfo, len(rows))
	var nextCursor *string
	for i, row := range rows {
		var actor *dto.NotificationActor
		if row.ActorID != nil {
			actor = &dto.NotificationActor{
				UserID:   *row.ActorID,
				FullName: row.ActorName,
				AvatarURL: row.ActorAvatar,
			}
		}

		var readAt *string
		if row.ReadAt != nil {
			s := row.ReadAt.Format(time.RFC3339)
			readAt = &s
		}

		data[i] = dto.NotificationInfo{
			ID:           row.ID,
			Type:         row.Type,
			Title:        row.Title,
			Content:      row.Content,
			ReferenceURL: row.ReferenceURL,
			Actor:        actor,
			IsRead:       row.IsRead,
			ReadAt:       readAt,
			CreatedAt:    row.CreatedAt.Format(time.RFC3339),
		}
	}

	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		raw := fmt.Sprintf("%s:%s", last.CreatedAt.Format(time.RFC3339Nano), last.ID)
		encoded := base64.StdEncoding.EncodeToString([]byte(raw))
		nextCursor = &encoded
	}

	var unreadCount int64
	r.db.Model(&models.NotificationRecipient{}).
		Where("recipient_id = ? AND is_read = false", recipientID).
		Count(&unreadCount)

	return &dto.NotificationListResponse{
		Data:        data,
		HasMore:     hasMore,
		NextCursor:  nextCursor,
		UnreadCount: int(unreadCount),
	}, nil
}

func (r *notificationRepository) CountUnreadByType(recipientID string) (*dto.UnreadCountResponse, error) {
	type countRow struct {
		Type  string `gorm:"column:type"`
		Count int    `gorm:"column:count"`
	}

	var rows []countRow
	err := r.db.Table("notification_recipients nr").
		Select("n.type, COUNT(*) as count").
		Joins("JOIN notifications n ON n.id = nr.notification_id").
		Where("nr.recipient_id = ? AND nr.is_read = false", recipientID).
		Group("n.type").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	byType := map[string]int{
		"ASSIGNED":         0,
		"MENTIONED":        0,
		"COMMENTED":        0,
		"STATUS_CHANGED":   0,
		"ADDED_TO_PROJECT": 0,
		"TASK_DUE_SOON":    0,
		"ANNOUNCEMENT":     0,
	}
	total := 0
	for _, row := range rows {
		byType[row.Type] = row.Count
		total += row.Count
	}

	return &dto.UnreadCountResponse{
		UnreadCount: total,
		ByType:      byType,
	}, nil
}

func (r *notificationRepository) MarkAsRead(notificationID, recipientID string) error {
	return r.db.Model(&models.NotificationRecipient{}).
		Where("notification_id = ? AND recipient_id = ?", notificationID, recipientID).
		Updates(map[string]interface{}{"is_read": true, "read_at": gorm.Expr("NOW()")}).Error
}

func (r *notificationRepository) MarkAllByTypeAsRead(recipientID string, notifType *string) (int64, error) {
	query := r.db.Model(&models.NotificationRecipient{}).
		Where("recipient_id = ? AND is_read = false", recipientID)
	if notifType != nil {
		query = query.Where("notification_id IN (SELECT id FROM notifications WHERE type = ?)", *notifType)
	}
	result := query.Updates(map[string]interface{}{"is_read": true, "read_at": gorm.Expr("NOW()")})
	return result.RowsAffected, result.Error
}

func (r *notificationRepository) GetWorkspaceMemberIDsByRoles(workspaceID string, roles []string) ([]string, error) {
	var ids []string
	err := r.db.Table("workspace_members").
		Select("user_id").
		Where("workspace_id = ? AND role IN ?", workspaceID, roles).
		Pluck("user_id", &ids).Error
	return ids, err
}

func (r *notificationRepository) FindUnreadCOMMENTEDByTask(taskID, recipientID string, since time.Time) (*models.Notification, error) {
	var n models.Notification
	pattern := "%/tasks/" + taskID + "%"
	err := r.db.Raw(`
		SELECT n.* FROM notifications n
		JOIN notification_recipients nr ON nr.notification_id = n.id
		WHERE n.type = ?
		AND n.reference_url LIKE ?
		AND nr.recipient_id = ?
		AND nr.is_read = false
		AND n.created_at >= ?
		ORDER BY n.created_at DESC
		LIMIT 1
	`, models.NotificationTypeCOMMENTED, pattern, recipientID, since).Scan(&n).Error
	if err != nil {
		return nil, err
	}
	if n.ID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	return &n, nil
}

func (r *notificationRepository) UpdateNotification(id string, title, content string) error {
	return r.db.Model(&models.Notification{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"title":      title,
			"content":    content,
			"created_at": gorm.Expr("NOW()"),
		}).Error
}

func (r *notificationRepository) CreateWithRecipients(notification *models.Notification, recipients []string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(notification).Error; err != nil {
			return err
		}
		for _, recipientID := range recipients {
			nr := models.NotificationRecipient{
				NotificationID: notification.ID,
				RecipientID:    recipientID,
			}
			if err := tx.Create(&nr).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *notificationRepository) FindTaskDueNotification(taskID string) (*models.TaskDueNotification, error) {
	var tdn models.TaskDueNotification
	err := r.db.Where("task_id = ?", taskID).First(&tdn).Error
	return &tdn, err
}

func (r *notificationRepository) CreateTaskDueNotification(tdn *models.TaskDueNotification) error {
	return r.db.Create(tdn).Error
}

func (r *notificationRepository) DeleteTaskDueNotification(taskID string) error {
	return r.db.Where("task_id = ?", taskID).Delete(&models.TaskDueNotification{}).Error
}

func (r *notificationRepository) DeleteOldNotifications(before time.Time) (int64, error) {
	result := r.db.Exec(`
		DELETE FROM notification_recipients nr
		USING notifications n
		WHERE n.id = nr.notification_id
		AND n.created_at < ?
	`, before)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (r *notificationRepository) DeleteOrphanNotifications() (int64, error) {
	result := r.db.Exec(`
		DELETE FROM notifications n
		WHERE NOT EXISTS (
			SELECT 1 FROM notification_recipients
			WHERE notification_id = n.id
		)
	`)
	return result.RowsAffected, result.Error
}

func (r *notificationRepository) IsUserProjectMember(projectID, userID string) (bool, error) {
	var count int64
	err := r.db.Table("project_members").
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Count(&count).Error
	return count > 0, err
}
