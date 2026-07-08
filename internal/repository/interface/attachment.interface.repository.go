package _interface

import (
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
)

type AttachmentRepository interface {
	WithTx(tx *gorm.DB) AttachmentRepository
	Create(attachment *models.Attachment) error
	GetByID(id string) (*models.Attachment, error)
	Delete(id string) error
	SoftDelete(id string, scheduledAt time.Time) error
	HardDelete(id string) error
	ListExpiredForDeletion() ([]models.Attachment, error)

	ListByTaskIDWithPagination(taskID string, fileType string, page int, limit int) (*dto.AttachmentListResponse, error)
	GetStorageUsageByWorkspace(workspaceID string) (*dto.StorageUsageResponse, error)
}
