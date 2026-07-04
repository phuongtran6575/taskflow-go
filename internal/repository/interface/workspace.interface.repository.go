package _interface

import (
	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
)

type WorkspaceRepository interface {
	WithTx(tx *gorm.DB) WorkspaceRepository

	Create(workspace *models.Workspace) error
	GetByID(id string) (*models.Workspace, error)
	Update(workspace *models.Workspace) error
	Delete(id string) error
	GetWorkspaceByDomain(domain string) (*models.Workspace, error)
	CountWorkspaceByPlan(plans []models.WorkspacePlan, role models.WorkspaceRole, userID string) ([]dto.WorkspacePlanCount, error)
	AddMember(member *models.WorkspaceMember) error
	GetMember(workspaceID, userID string) (*models.WorkspaceMember, error)

	ListByUserIDWithSummary(userID string) ([]dto.WorkspaceSummary, int, error)
	GetByIDWithDetail(workspaceID string) (*dto.WorkspaceDetailResponse, error)
}
