package implement

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/activitylog"
	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/helper"
	"TaskFlow-Go/internal/models"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
)

type roleService struct {
	roleRepo           repoInterface.RoleRepository
	rolePermissionRepo repoInterface.RolePermissionRepository
	permRepo           repoInterface.PermissionRepository
	projMemRepo        repoInterface.ProjectMemberRepository
	workspaceRepo      repoInterface.WorkspaceRepository
	activityLogRepo    repoInterface.ActivityLogRepository
	userRepo           repoInterface.UserRepository
}

func NewRoleService(
	roleRepo repoInterface.RoleRepository,
	rolePermissionRepo repoInterface.RolePermissionRepository,
	permRepo repoInterface.PermissionRepository,
	projMemRepo repoInterface.ProjectMemberRepository,
	workspaceRepo repoInterface.WorkspaceRepository,
	activityLogRepo repoInterface.ActivityLogRepository,
	userRepo repoInterface.UserRepository,
) _interface.RoleService {
	return &roleService{
		roleRepo:           roleRepo,
		rolePermissionRepo: rolePermissionRepo,
		permRepo:           permRepo,
		projMemRepo:        projMemRepo,
		workspaceRepo:      workspaceRepo,
		activityLogRepo:    activityLogRepo,
		userRepo:           userRepo,
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

func (s *roleService) logActivity(workspaceID, projectID, userID, entityID string, action models.ActivityAction, metadata map[string]interface{}, description string, entitySnapshot map[string]interface{}) {
	var metaStr *string
	if metadata != nil {
		b, _ := json.Marshal(metadata)
		str := string(b)
		metaStr = &str
	}
	var snapStr *string
	if entitySnapshot != nil {
		b, _ := json.Marshal(entitySnapshot)
		str := string(b)
		snapStr = &str
	}
	var descPtr *string
	if description != "" {
		descPtr = &description
	}
	_ = s.activityLogRepo.Create(&models.ActivityLog{
		WorkspaceID:    &workspaceID,
		UserID:         &userID,
		Action:         action,
		EntityType:     models.EntityTypeROLE,
		EntityID:       entityID,
		Description:    descPtr,
		Metadata:       metaStr,
		EntitySnapshot: snapStr,
	})
}

func (s *roleService) getUserName(userID string) string {
	u, err := s.userRepo.GetByID(userID)
	if err != nil {
		return userID
	}
	return u.FullName
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
	actorName := s.getUserName(userID)
	meta := map[string]interface{}{
		"event":            "role_created",
		"name":             role.Name,
		"permission_count": len(validatedPermIDs),
	}
	desc := activitylog.GenerateDescription(actorName, meta)
	s.logActivity(workspaceID, "", userID, role.ID, models.ActivityActionCREATE, meta, desc, nil)

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

	var roleChanges []activitylog.ChangeField
	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if err := s.validateRoleName(trimmed); err != nil {
			return nil, err
		}
		if s.isRoleNameTaken(workspaceID, trimmed, roleID) {
			return nil, apperror.ErrRoleNameAlreadyExists
		}
		roleChanges = append(roleChanges, activitylog.BuildChangeField("name", role.Name, trimmed))
		role.Name = trimmed
	}
	if req.Description != nil {
		roleChanges = append(roleChanges, activitylog.BuildChangeField("description", role.Description, req.Description))
		role.Description = req.Description
	}
	role.UpdatedAt = time.Now()
	if err := s.roleRepo.Update(role); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update role")
	}

	// BR-ROLE-07: log activity
	if len(roleChanges) > 0 {
		actorName := s.getUserName(userID)
		meta := activitylog.TaskUpdated(roleChanges)
		desc := activitylog.GenerateDescription(actorName, meta)
		s.logActivity(workspaceID, "", userID, role.ID, models.ActivityActionUPDATE, meta, desc, nil)
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
		actorName := s.getUserName(userID)
		meta := map[string]interface{}{
			"event":       "permissions_assigned",
			"added_slugs": slugs,
			"action":      "gán quyền",
		}
		desc := activitylog.GenerateDescription(actorName, meta)
		s.logActivity(workspaceID, "", userID, roleID, models.ActivityActionUPDATE, meta, desc, nil)
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
		actorName := s.getUserName(userID)
		meta := map[string]interface{}{
			"event":         "permissions_removed",
			"removed_slugs": slugs,
			"action":        "gỡ quyền",
		}
		desc := activitylog.GenerateDescription(actorName, meta)
		s.logActivity(workspaceID, "", userID, roleID, models.ActivityActionUPDATE, meta, desc, nil)
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

	actorName := s.getUserName(userID)
	meta := map[string]interface{}{
		"event":          "role_deleted",
		"name":           role.Name,
		"had_permissions": hadPerms,
	}
	desc := activitylog.GenerateDescription(actorName, meta)
	s.logActivity(workspaceID, "", userID, roleID, models.ActivityActionDELETE, meta, desc, nil)

	return nil
}
