package implement

import (
	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
	"errors"
	"net/http"
	"time"
)

type roleService struct {
	roleRepo           repoInterface.RoleRepository
	rolePermissionRepo repoInterface.RolePermissionRepository
	permRepo           repoInterface.PermissionRepository
	projMemRepo        repoInterface.ProjectMemberRepository
	workspaceRepo      repoInterface.WorkspaceRepository
}

func NewRoleService(
	roleRepo repoInterface.RoleRepository,
	rolePermissionRepo repoInterface.RolePermissionRepository,
	permRepo repoInterface.PermissionRepository,
	projMemRepo repoInterface.ProjectMemberRepository,
	workspaceRepo repoInterface.WorkspaceRepository,
) _interface.RoleService {
	return &roleService{
		roleRepo:           roleRepo,
		rolePermissionRepo: rolePermissionRepo,
		permRepo:           permRepo,
		projMemRepo:        projMemRepo,
		workspaceRepo:      workspaceRepo,
	}
}

func (s *roleService) safeDescription(d *string) string {
	if d == nil {
		return ""
	}
	return *d
}

func (s *roleService) isRoleNameTaken(workspaceID, name, excludeID string) bool {
	roles, err := 	s.roleRepo.ListByWorkspaceID(workspaceID)
	if err != nil {
		return false
	}
	for _, r := range roles {
		if r.Name == name && r.ID != excludeID {
			return true
		}
	}
	return false
}

func (s *roleService) getRoleLimit(plan models.WorkspacePlan) int {
	switch plan {
	case models.WorkspacePlanFREE:
		return 20
	case models.WorkspacePlanPRO:
		return 50
	default:
		return -1
	}
}

func (s *roleService) ListRoles(workspaceID string, userID string, search string, page int, limit int) ([]dto.RoleSummary, *dto.Pagination, error) {
	return s.roleRepo.ListWithPagination(workspaceID, search, page, limit)
}

func (s *roleService) CreateRole(workspaceID string, userID string, req *dto.CreateRoleRequest) (*dto.RoleCreateResponse, error) {
	if len(req.Name) < 1 || len(req.Name) > 50 {
		return nil, apperror.NewAppError(http.StatusBadRequest, "VALIDATION_ERROR", "Role name must be between 1 and 50 characters")
	}
	if s.isRoleNameTaken(workspaceID, req.Name, "") {
		return nil, apperror.ErrRoleNameAlreadyExists
	}
	workspace, err := s.workspaceRepo.GetByID(workspaceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrWorkspaceNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get workspace")
	}
	roles, err := 	s.roleRepo.ListByWorkspaceID(workspaceID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to count roles")
	}
	limit := s.getRoleLimit(workspace.Plan)
	if limit > 0 && len(roles) >= limit {
		return nil, apperror.ErrRoleLimitReached
	}

	if len(req.PermissionIDs) > 0 {
		var count int64
		for _, id := range req.PermissionIDs {
			_, err := s.permRepo.GetByID(id)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, apperror.ErrInvalidPermissionIDs
				}
				return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to validate permissions")
			}
			count++
		}
		_ = count
	}

	role, err := s.roleRepo.Create(&models.Role{
		Name:        req.Name,
		Description: req.Description,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create role")
	}

	if len(req.PermissionIDs) > 0 {
		if err := s.rolePermissionRepo.BulkCreate(role.ID, req.PermissionIDs); err != nil {
			return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to assign permissions")
		}
	}

	var permissions map[string][]dto.PermissionInfo
	var permissionCount int
	if len(req.PermissionIDs) > 0 {
		perms, count, err := s.permRepo.GetListPermissionsByModule(req.PermissionIDs)
		if err != nil {
			return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get permissions")
		}
		permissions = perms
		permissionCount = *count
	} else {
		permissions = make(map[string][]dto.PermissionInfo)
	}

	return &dto.RoleCreateResponse{
		ID:              role.ID,
		Name:            role.Name,
		Description:     s.safeDescription(role.Description),
		Permissions:     permissions,
		PermissionCount: permissionCount,
		MemberCount:     0,
		UpdatedAt:       time.Now(),
	}, nil
}

func (s *roleService) GetRoleById(workspaceID string, userID string, roleID string) (*dto.RoleDetailResponse, error) {
	result, err := s.roleRepo.GetByIDWithDetail(workspaceID, roleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrRoleNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get role details")
	}
	return result, nil
}

func (s *roleService) UpdateRole(workspaceID string, userID string, roleID string, req *dto.UpdateRoleRequest) (*dto.RoleUpdateResponse, error) {
	role, err := s.roleRepo.GetByID(roleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrRoleNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get role")
	}
	if role.WorkspaceID != workspaceID {
		return nil, apperror.ErrRoleNotFound
	}

	if req.Name != nil {
		if len(*req.Name) < 1 || len(*req.Name) > 50 {
			return nil, apperror.NewAppError(http.StatusBadRequest, "VALIDATION_ERROR", "Role name must be between 1 and 50 characters")
		}
		if s.isRoleNameTaken(workspaceID, *req.Name, roleID) {
			return nil, apperror.ErrRoleNameAlreadyExists
		}
		role.Name = *req.Name
	}
	if req.Description != nil {
		role.Description = req.Description
	}
	role.UpdatedAt = time.Now()
	if err := s.roleRepo.Update(role); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update role")
	}

	rolePermissions, err := s.rolePermissionRepo.GetPermissionsByRoleID(roleID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get permissions")
	}
	permissionCount := len(rolePermissions)
	members, err := s.projMemRepo.GetProjectMembersByRoleID(roleID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get member count")
	}

	name := role.Name
	desc := s.safeDescription(role.Description)
	return &dto.RoleUpdateResponse{
		ID:              role.ID,
		Name:            name,
		Description:     desc,
		UpdatedAt:       role.UpdatedAt,
		PermissionCount: permissionCount,
		MemberCount:     len(members),
	}, nil
}

func (s *roleService) AssignPermissionsToRole(workspaceID string, userID string, roleID string, req *dto.AssignPermissionsRequest) (*dto.AssignPermissionsResponse, error) {
	role, err := s.roleRepo.GetByID(roleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrRoleNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get role")
	}
	if role.WorkspaceID != workspaceID {
		return nil, apperror.ErrRoleNotFound
	}

	if len(req.PermissionIDs) > 0 {
		for _, id := range req.PermissionIDs {
			_, err := s.permRepo.GetByID(id)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, apperror.ErrInvalidPermissionIDs
				}
				return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to validate permissions")
			}
		}
	}

	existingRolePerms, err := s.rolePermissionRepo.GetPermissionsByRoleID(roleID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get existing permissions")
	}
	existingSet := make(map[string]struct{}, len(existingRolePerms))
	for _, rp := range existingRolePerms {
		existingSet[rp.PermissionID] = struct{}{}
	}

	var toAdd []string
	var skipped []string
	for _, id := range req.PermissionIDs {
		if _, ok := existingSet[id]; ok {
			skipped = append(skipped, id)
		} else {
			toAdd = append(toAdd, id)
		}
	}

	if len(toAdd) == 0 {
		return &dto.AssignPermissionsResponse{
			RoleID:               roleID,
			Added:                []dto.PermissionAssignedInfo{},
			SkippedAlreadyExists: skipped,
			PermissionCountAfter: len(existingRolePerms),
			UpdatedAt:            time.Now(),
		}, nil
	}

	if err := s.rolePermissionRepo.BulkCreate(roleID, toAdd); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to assign permissions")
	}

	addedPermissions, err := s.permRepo.GetListPermissions(toAdd)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get added permissions")
	}

	allPerms, err := s.rolePermissionRepo.GetPermissionsByRoleID(roleID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to count permissions after assignment")
	}

	return &dto.AssignPermissionsResponse{
		RoleID:               roleID,
		Added:                addedPermissions,
		SkippedAlreadyExists: skipped,
		PermissionCountAfter: len(allPerms),
		UpdatedAt:            time.Now(),
	}, nil
}

func (s *roleService) RemovePermissionsFromRole(workspaceID string, userID string, roleID string, req *dto.RemovePermissionsRequest) (*dto.RemovePermissionsResponse, error) {
	role, err := s.roleRepo.GetByID(roleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrRoleNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get role")
	}
	if role.WorkspaceID != workspaceID {
		return nil, apperror.ErrRoleNotFound
	}

	existingRolePerms, err := s.rolePermissionRepo.GetPermissionsByRoleID(roleID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get existing permissions")
	}
	existingSet := make(map[string]struct{}, len(existingRolePerms))
	for _, rp := range existingRolePerms {
		existingSet[rp.PermissionID] = struct{}{}
	}

	var toRemove []string
	var notFound []string
	for _, id := range req.PermissionIDs {
		if _, ok := existingSet[id]; ok {
			toRemove = append(toRemove, id)
		} else {
			notFound = append(notFound, id)
		}
	}

	if len(toRemove) == 0 {
		return &dto.RemovePermissionsResponse{
			RoleID:               roleID,
			Removed:              []dto.PermissionAssignedInfo{},
			SkippedNotExists:     notFound,
			PermissionCountAfter: len(existingRolePerms),
			UpdatedAt:            time.Now(),
		}, nil
	}

	if err := s.rolePermissionRepo.BulkDelete(roleID, toRemove); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to remove permissions")
	}

	removedPermissions, err := s.permRepo.GetListPermissions(toRemove)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get removed permissions")
	}

	allPerms, err := s.rolePermissionRepo.GetPermissionsByRoleID(roleID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to count permissions after removal")
	}

	return &dto.RemovePermissionsResponse{
		RoleID:               roleID,
		Removed:              removedPermissions,
		SkippedNotExists:     notFound,
		PermissionCountAfter: len(allPerms),
		UpdatedAt:            time.Now(),
	}, nil
}

func (s *roleService) DeleteRole(workspaceID string, userID string, roleID string) error {
	role, err := s.roleRepo.GetByID(roleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.ErrRoleNotFound
		}
		return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get role")
	}
	if role.WorkspaceID != workspaceID {
		return apperror.ErrRoleNotFound
	}

	members, err := s.projMemRepo.GetProjectMembersByRoleID(roleID)
	if err != nil {
		return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check role usage")
	}
	if len(members) > 0 {
		return apperror.ErrRoleInUse
	}

	if err := s.roleRepo.Delete(roleID); err != nil {
		return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete role")
	}
	return nil
}
