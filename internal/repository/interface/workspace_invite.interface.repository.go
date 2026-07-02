package _interface

import (
	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
)

type WorkspaceInviteRepository interface {
	Create(invite *models.WorkspaceInvite) error
	GetByID(id string) (*models.WorkspaceInvite, error)
	GetByCode(code string) (*models.WorkspaceInvite, error)
	IncrementUses(id string) error

	ListWithPagination(workspaceID string, status string, page int, limit int) ([]dto.InviteInfo, *dto.Pagination, error)
	GetByCodeWithPreview(code string) (*dto.InvitePreviewResponse, error)
	CountActiveByWorkspaceID(workspaceID string) (int64, error)
	SoftDelete(id string) error
}
