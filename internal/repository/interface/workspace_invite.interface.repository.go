package _interface

import (
	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
)

type WorkspaceInviteRepository interface {
	WithTx(tx *gorm.DB) WorkspaceInviteRepository
	Create(invite *models.WorkspaceInvite) error
	GetByID(id string) (*models.WorkspaceInvite, error)
	GetByCode(code string) (*models.WorkspaceInvite, error)
	IncrementUses(id string) error
	CountByCode(code string) (int64, error)

	ListWithPagination(workspaceID string, status string, page int, limit int) ([]dto.InviteInfo, *dto.Pagination, error)
	GetByCodeWithPreview(code string) (*dto.InvitePreviewResponse, error)
	CountActiveByWorkspaceID(workspaceID string) (int64, error)
	SoftDelete(id string) error
}
