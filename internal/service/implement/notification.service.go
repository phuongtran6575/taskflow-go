package implement

import (
	"errors"
	"net/http"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
)

type notificationService struct {
	notifRepo          repoInterface.NotificationRepository
	workspaceMemberRepo repoInterface.WorkspaceMemberRepository
}

func NewNotificationService(
	notifRepo repoInterface.NotificationRepository,
	workspaceMemberRepo repoInterface.WorkspaceMemberRepository,
) _interface.NotificationService {
	return &notificationService{
		notifRepo:          notifRepo,
		workspaceMemberRepo: workspaceMemberRepo,
	}
}

func (s *notificationService) getWorkspaceMemberOrFail(workspaceID, userID string) error {
	_, err := s.workspaceMemberRepo.GetByID(workspaceID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.ErrForbidden
		}
		return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to verify workspace membership")
	}
	return nil
}

func (s *notificationService) ListNotifications(workspaceID string, userID string, isRead *bool, types []string, limit int, cursor string) (*dto.NotificationListResponse, error) {
	if err := s.getWorkspaceMemberOrFail(workspaceID, userID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	result, err := s.notifRepo.ListWithCursor(userID, isRead, types, limit, cursor)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list notifications")
	}
	return result, nil
}

func (s *notificationService) GetUnreadCount(workspaceID string, userID string) (*dto.UnreadCountResponse, error) {
	if err := s.getWorkspaceMemberOrFail(workspaceID, userID); err != nil {
		return nil, err
	}
	result, err := s.notifRepo.CountUnreadByType(userID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get unread count")
	}
	return result, nil
}

func (s *notificationService) MarkAsRead(workspaceID string, userID string, notificationID string) (*dto.MarkAsReadResponse, error) {
	if err := s.getWorkspaceMemberOrFail(workspaceID, userID); err != nil {
		return nil, err
	}

	_, err := s.notifRepo.GetByID(notificationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotificationNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get notification")
	}

	_, err = s.notifRepo.GetRecipient(notificationID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotificationNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get notification recipient")
	}

	if err := s.notifRepo.MarkAsRead(notificationID, userID); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to mark as read")
	}

	now := time.Now().Format(time.RFC3339)
	return &dto.MarkAsReadResponse{
		NotificationID: notificationID,
		IsRead:         true,
		ReadAt:         now,
	}, nil
}

func (s *notificationService) MarkAllAsRead(workspaceID string, userID string, typeFilter *string) (*dto.MarkAllAsReadResponse, error) {
	if err := s.getWorkspaceMemberOrFail(workspaceID, userID); err != nil {
		return nil, err
	}

	marked, err := s.notifRepo.MarkAllByTypeAsRead(userID, typeFilter)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to mark all as read")
	}

	uc, err := s.notifRepo.CountUnreadByType(userID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get unread count")
	}

	return &dto.MarkAllAsReadResponse{
		MarkedReadCount:  int(marked),
		UnreadCountAfter: uc.UnreadCount,
	}, nil
}

func (s *notificationService) CreateAnnouncement(workspaceID string, userID string, req *dto.CreateAnnouncementRequest) (*dto.CreateAnnouncementResponse, error) {
	title := req.Title
	content := req.Content
	targetRoles := req.TargetRoles

	if title == "" || content == "" {
		return nil, apperror.ErrValidation
	}

	validRoles := map[string]bool{"OWNER": true, "ADMIN": true, "MEMBER": true}
	for _, role := range targetRoles {
		if !validRoles[role] {
			return nil, apperror.ErrInvalidTargetRoles
		}
	}

	if len(targetRoles) == 0 {
		targetRoles = []string{"OWNER", "ADMIN", "MEMBER"}
	}

	recipientIDs, err := s.notifRepo.GetWorkspaceMemberIDsByRoles(workspaceID, targetRoles)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get workspace members")
	}

	actorID := userID
	now := time.Now()
	notification := &models.Notification{
		ActorID:      &actorID,
		Type:         models.NotificationTypeANNOUNCEMENT,
		Title:        title,
		Content:      &content,
		ReferenceURL: nil,
		CreatedAt:    now,
	}

	if err := s.notifRepo.Create(notification, recipientIDs); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create announcement")
	}

	return &dto.CreateAnnouncementResponse{
		NotificationID: notification.ID,
		Title:          title,
		Content:        content,
		SentToCount:    len(recipientIDs),
		CreatedAt:      now.Format(time.RFC3339),
	}, nil
}


