package _interface

import "TaskFlow-Go/internal/dto"

type NotificationService interface {
	ListNotifications(workspaceID string, userID string, isRead *bool, types []string, limit int, cursor string) (*dto.NotificationListResponse, error)
	GetUnreadCount(workspaceID string, userID string) (*dto.UnreadCountResponse, error)
	MarkAsRead(workspaceID string, userID string, notificationID string) (*dto.MarkAsReadResponse, error)
	MarkAllAsRead(workspaceID string, userID string, typeFilter *string) (*dto.MarkAllAsReadResponse, error)
	CreateAnnouncement(workspaceID string, userID string, req *dto.CreateAnnouncementRequest) (*dto.CreateAnnouncementResponse, error)
}
