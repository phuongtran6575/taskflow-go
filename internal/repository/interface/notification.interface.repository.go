package _interface

import (
	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
)

type NotificationRepository interface {
	Create(notification *models.Notification, recipients []string) error
	GetByID(id string) (*models.Notification, error)
	GetRecipient(notificationID, recipientID string) (*models.NotificationRecipient, error)
	ListWithCursor(recipientID string, isRead *bool, types []string, limit int, cursor string) (*dto.NotificationListResponse, error)
	CountUnreadByType(recipientID string) (*dto.UnreadCountResponse, error)
	MarkAsRead(notificationID, recipientID string) error
	MarkAllByTypeAsRead(recipientID string, notifType *string) (int64, error)
	GetWorkspaceMemberIDsByRoles(workspaceID string, roles []string) ([]string, error)
}
