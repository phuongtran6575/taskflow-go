package _interface

import "TaskFlow-Go/internal/dto"

type RoleService interface {
	ListRoles(workspaceID string, userID string, search string, page int, limit int) ([]dto.RoleSummary, *dto.Pagination, error)
	CreateRole(workspaceID string, userID string, req *dto.CreateRoleRequest) (*dto.RoleCreateResponse, error)
	GetRoleById(workspaceID string, userID string, roleID string) (*dto.RoleDetailResponse, error)
	UpdateRole(workspaceID string, userID string, roleID string, req *dto.UpdateRoleRequest) (*dto.RoleUpdateResponse, error)
	AssignPermissionsToRole(workspaceID string, userID string, roleID string, req *dto.AssignPermissionsRequest) (*dto.AssignPermissionsResponse, error)
	RemovePermissionsFromRole(workspaceID string, userID string, roleID string, req *dto.RemovePermissionsRequest) (*dto.RemovePermissionsResponse, error)
	DeleteRole(workspaceID string, userID string, roleID string) error
}
