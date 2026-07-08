package _interface

import (
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
)

type NotificationRepository interface {
	WithTx(tx *gorm.DB) NotificationRepository
	Create(notification *models.Notification, recipients []string) error
	GetByID(id string) (*models.Notification, error)
	GetRecipient(notificationID, recipientID string) (*models.NotificationRecipient, error)
	ListWithCursor(recipientID string, isRead *bool, types []string, limit int, cursor string) (*dto.NotificationListResponse, error)
	CountUnreadByType(recipientID string) (*dto.UnreadCountResponse, error)
	MarkAsRead(notificationID, recipientID string) error
	MarkAllByTypeAsRead(recipientID string, notifType *string) (int64, error)
	GetWorkspaceMemberIDsByRoles(workspaceID string, roles []string) ([]string, error)

	FindUnreadCOMMENTEDByTask(taskID, recipientID string, since time.Time) (*models.Notification, error)
	UpdateNotification(id string, title, content string) error
	CreateWithRecipients(notification *models.Notification, recipients []string) error

	FindTaskDueNotification(taskID string) (*models.TaskDueNotification, error)
	CreateTaskDueNotification(tdn *models.TaskDueNotification) error
	DeleteTaskDueNotification(taskID string) error

	DeleteOldNotifications(before time.Time) (int64, error)
	DeleteOrphanNotifications() (int64, error)

	IsUserProjectMember(projectID, userID string) (bool, error)
}
