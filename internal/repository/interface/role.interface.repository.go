package _interface

import (
	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
)

type RoleRepository interface {
	Create(role *models.Role) (*models.Role, error)
	GetByID(id string) (*models.Role, error)
	ListByWorkspaceID(workspaceID string) ([]models.Role, error)
	Update(role *models.Role) error
	Delete(id string) error
	CountByWorkspaceID(workspaceID string) (int64, error)

	ListWithPagination(workspaceID string, search string, page int, limit int) ([]dto.RoleSummary, *dto.Pagination, error)
	GetByIDWithDetail(workspaceID string, roleID string) (*dto.RoleDetailResponse, error)
	GetAffectedProjectsByRoleID(roleID string) ([]dto.AffectedProject, int, error)

	// ValidateRoleIDsBelongToWorkspace returns list of role IDs that do NOT belong to the workspace.
	// Returns empty slice if all role IDs are valid. (BR-PRA-04)
	ValidateRoleIDsBelongToWorkspace(roleIDs []string, workspaceID string) ([]string, error)
}
