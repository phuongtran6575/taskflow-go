package implement

import (
	"encoding/json"
	"gorm.io/gorm"
	"strings"
	"time"
	"unicode/utf8"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/helper"
	"TaskFlow-Go/internal/models"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
	"errors"
	"net/http"
)

type roleService struct {
	roleRepo           repoInterface.RoleRepository
	rolePermissionRepo repoInterface.RolePermissionRepository
	permRepo           repoInterface.PermissionRepository
	projMemRepo        repoInterface.ProjectMemberRepository
	workspaceRepo      repoInterface.WorkspaceRepository
	activityLogRepo    repoInterface.ActivityLogRepository
}

func NewRoleService(
	roleRepo repoInterface.RoleRepository,
	rolePermissionRepo repoInterface.RolePermissionRepository,
	permRepo repoInterface.PermissionRepository,
	projMemRepo repoInterface.ProjectMemberRepository,
	workspaceRepo repoInterface.WorkspaceRepository,
	activityLogRepo repoInterface.ActivityLogRepository,
) _interface.RoleService {
	return &roleService{
		roleRepo:           roleRepo,
		rolePermissionRepo: rolePermissionRepo,
		permRepo:           permRepo,
		projMemRepo:        projMemRepo,
		workspaceRepo:      workspaceRepo,
		activityLogRepo:    activityLogRepo,
	}
}

func (s *roleService) safeDescription(d *string) string {
	if d == nil {
		return ""
	}
	return *d
}

// BR-ROLE-01: case-insensitive, trim whitespace comparison
func (s *roleService) isRoleNameTaken(workspaceID, name, excludeID string) bool {
	roles, err := s.roleRepo.ListByWorkspaceID(workspaceID)
	if err != nil {
		return false
	}
	trimmed := strings.TrimSpace(name)
	for _, r := range roles {
		if strings.EqualFold(strings.TrimSpace(r.Name), trimmed) && r.ID != excludeID {
			return true
		}
	}
	return false
}

// BR-ROLE-01: validate role name (trim + length in runes)
func (s *roleService) validateRoleName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return apperror.NewAppError(http.StatusBadRequest, "VALIDATION_ERROR", "Role name must not be empty or only whitespace")
	}
	runeCount := utf8.RuneCountInString(trimmed)
	if runeCount < 2 || runeCount > 50 {
		return apperror.NewAppError(http.StatusBadRequest, "VALIDATION_ERROR", "Role name must be between 2 and 50 characters")
	}
	return nil
}

// BR-ROLE-06: deduplicate and validate permission IDs
func (s *roleService) validatePermissionIDs(ids []string) ([]string, *apperror.InvalidPermissionIDsError) {
	if len(ids) == 0 {
		return nil, nil
	}
	deduped := make([]string, 0, len(ids))
	seen := make(map[string]struct{})
	for _, id := range ids {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			deduped = append(deduped, id)
		}
	}
	found, invalidIDs, err := s.permRepo.ValidatePermissionIDs(deduped)
	if err != nil {
		return nil, &apperror.InvalidPermissionIDsError{
			AppError:   apperror.ErrInvalidPermissionIDs,
			InvalidIDs: deduped,
		}
	}
	if len(invalidIDs) > 0 {
		return nil, &apperror.InvalidPermissionIDsError{
			AppError:   apperror.ErrInvalidPermissionIDs,
			InvalidIDs: invalidIDs,
		}
	}
	return found, nil
}

func (s *roleService) logActivity(workspaceID, userID, action string, entityID string, metadata map[string]interface{}) {
	metaBytes, _ := json.Marshal(metadata)
	metaStr := string(metaBytes)
	_ = s.activityLogRepo.Create(&models.ActivityLog{
		WorkspaceID: &workspaceID,
		UserID:      &userID,
		Action:      models.ActivityAction(action),
		EntityType:  models.EntityTypeROLE,
		EntityID:    entityID,
		Metadata:    &metaStr,
	})
}

func (s *roleService) ListRoles(workspaceID string, userID string, search string, page int, limit int) ([]dto.RoleSummary, *dto.Pagination, error) {
	return s.roleRepo.ListWithPagination(workspaceID, search, page, limit)
}

// BR-ROLE-02/BR-ROLE-01/BR-ROLE-06/BR-ROLE-07
func (s *roleService) CreateRole(workspaceID string, userID string, req *dto.CreateRoleRequest) (*dto.RoleCreateResponse, error) {
	req.Name = strings.TrimSpace(req.Name)
	if err := s.validateRoleName(req.Name); err != nil {
		return nil, err
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
	// BR-ROLE-02: count existing roles and check limit
	currentCount, err := s.roleRepo.CountByWorkspaceID(workspaceID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to count roles")
	}
	if err := helper.CheckRoleLimit(workspace.Plan, int(currentCount)); err != nil {
		return nil, err
	}

	// BR-ROLE-06: validate permissions (dedup + batch + all-or-nothing)
	var validatedPermIDs []string
	if len(req.PermissionIDs) > 0 {
		var permErr *apperror.InvalidPermissionIDsError
		validatedPermIDs, permErr = s.validatePermissionIDs(req.PermissionIDs)
		if permErr != nil {
			return nil, permErr
		}
	}

	role, err := s.roleRepo.Create(&models.Role{
		Name:        req.Name,
		Description: req.Description,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create role")
	}

	if len(validatedPermIDs) > 0 {
		if err := s.rolePermissionRepo.BulkCreate(role.ID, validatedPermIDs); err != nil {
			return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to assign permissions")
		}
	}

	// BR-ROLE-07: log activity
	s.logActivity(workspaceID, userID, "CREATE", role.ID, map[string]interface{}{
		"name":             role.Name,
		"permission_count": len(validatedPermIDs),
	})

	var permissions map[string][]dto.PermissionInfo
	var permissionCount int
	if len(validatedPermIDs) > 0 {
		perms, count, err := s.permRepo.GetListPermissionsByModule(validatedPermIDs)
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

// BR-ROLE-01/BR-ROLE-07
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

	var oldName string
	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if err := s.validateRoleName(trimmed); err != nil {
			return nil, err
		}
		if s.isRoleNameTaken(workspaceID, trimmed, roleID) {
			return nil, apperror.ErrRoleNameAlreadyExists
		}
		oldName = role.Name
		role.Name = trimmed
	}
	if req.Description != nil {
		role.Description = req.Description
	}
	role.UpdatedAt = time.Now()
	if err := s.roleRepo.Update(role); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update role")
	}

	// BR-ROLE-07: log activity
	if req.Name != nil {
		s.logActivity(workspaceID, userID, "UPDATE", role.ID, map[string]interface{}{
			"old_name": oldName,
			"new_name": role.Name,
		})
	}
	if req.Description != nil {
		s.logActivity(workspaceID, userID, "UPDATE", role.ID, map[string]interface{}{
			"field": "description",
		})
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

	return &dto.RoleUpdateResponse{
		ID:              role.ID,
		Name:            role.Name,
		Description:     s.safeDescription(role.Description),
		UpdatedAt:       role.UpdatedAt,
		PermissionCount: permissionCount,
		MemberCount:     len(members),
	}, nil
}

// BR-ROLE-06/BR-ROLE-07
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

	// BR-ROLE-06: dedup + batch validate + all-or-nothing
	var validatedPermIDs []string
	if len(req.PermissionIDs) > 0 {
		var permErr *apperror.InvalidPermissionIDsError
		validatedPermIDs, permErr = s.validatePermissionIDs(req.PermissionIDs)
		if permErr != nil {
			return nil, permErr
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
	for _, id := range validatedPermIDs {
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

	// BR-ROLE-07: log activity
	addedPermissions, err := s.permRepo.GetListPermissions(toAdd)
	if err == nil {
		slugs := make([]string, len(addedPermissions))
		for i, p := range addedPermissions {
			slugs[i] = p.Slug
		}
		s.logActivity(workspaceID, userID, "UPDATE", roleID, map[string]interface{}{
			"added_slugs": slugs,
		})
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

// BR-ROLE-06/BR-ROLE-07
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

	// BR-ROLE-06: dedup + batch validate for remove (still validate they exist in system)
	var validatedPermIDs []string
	if len(req.PermissionIDs) > 0 {
		var permErr *apperror.InvalidPermissionIDsError
		validatedPermIDs, permErr = s.validatePermissionIDs(req.PermissionIDs)
		if permErr != nil {
			return nil, permErr
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

	var toRemove []string
	var notFound []string
	for _, id := range validatedPermIDs {
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

	// BR-ROLE-07: log activity
	removedPermissions, err := s.permRepo.GetListPermissions(toRemove)
	if err == nil {
		slugs := make([]string, len(removedPermissions))
		for i, p := range removedPermissions {
			slugs[i] = p.Slug
		}
		s.logActivity(workspaceID, userID, "UPDATE", roleID, map[string]interface{}{
			"removed_slugs": slugs,
		})
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

// BR-ROLE-03/BR-ROLE-07
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

	// BR-ROLE-03: check if role is in use with affected projects
	projects, totalMembers, err := s.roleRepo.GetAffectedProjectsByRoleID(roleID)
	if err != nil {
		return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check role usage")
	}
	if len(projects) > 0 {
		return &apperror.RoleInUseError{
			AppError:             apperror.ErrRoleInUse,
			AffectedProjects:     projects,
			TotalAffectedMembers: totalMembers,
		}
	}

	// BR-ROLE-07: log before delete
	hadPerms := make([]string, 0)
	existingPerms, err := s.rolePermissionRepo.GetPermissionsByRoleID(roleID)
	if err == nil {
		for _, rp := range existingPerms {
			hadPerms = append(hadPerms, rp.PermissionID)
		}
	}

	// BR-ROLE-03: hard delete (DeletedAt removed from model)
	if err := s.roleRepo.Delete(roleID); err != nil {
		return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete role")
	}

	s.logActivity(workspaceID, userID, "DELETE", roleID, map[string]interface{}{
		"name":            role.Name,
		"had_permissions": hadPerms,
	})

	return nil
}
