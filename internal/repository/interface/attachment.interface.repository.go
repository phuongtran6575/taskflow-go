package _interface

import (
	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
)

type AttachmentRepository interface {
	Create(attachment *models.Attachment) error
	GetByID(id string) (*models.Attachment, error)
	Delete(id string) error

	ListByTaskIDWithPagination(taskID string, fileType string, page int, limit int) (*dto.AttachmentListResponse, error)
	GetStorageUsageByWorkspace(workspaceID string) (*dto.StorageUsageResponse, error)
}
