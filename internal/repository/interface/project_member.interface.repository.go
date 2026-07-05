package _interface

import (
	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
)

type ProjectMemberRepository interface {
	GetByID(projectID, userID string) (*models.ProjectMember, error)
	GetByIDWithRelationRole(projectID, userID string) (*models.ProjectMember, error)
	ListByProjectID(projectID string) ([]models.ProjectMember, error)
	GetProjectMembersByRoleID(roleID string) ([]models.ProjectMember, error)
	UpdateRole(projectID, userID string, roleID string) error
	ValidateMembersExist(projectID string, userIDs []string) ([]string, error)
	Delete(projectID, userID string) error
	DeleteByWorkspace(workspaceID, userID string) error
	Update(member *models.ProjectMember) error
	Create(member *models.ProjectMember) error

	// HasPermission kiểm tra user có permission slug cụ thể trong project không.
	// Query JOIN 3 bảng: project_members → role_permissions → permissions
	HasPermission(projectID, userID, permissionSlug string) (bool, error)
	BulkAddMember(projectID string, users []dto.MemberRolePair) ([]models.ProjectMember, error)
	ListMemberIDs(projectID string) ([]string, error)

	ListWithPagination(projectID string, search string, roleID string, page int, limit int) ([]dto.ProjectMemberInfo, *dto.Pagination, error)
	ListAvailableWorkspaceMembers(workspaceID string, projectID string, search string, page int, limit int) ([]dto.AvailableWorkspaceMember, *dto.Pagination, error)
}
