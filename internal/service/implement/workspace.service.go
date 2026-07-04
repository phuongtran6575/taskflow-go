package implement

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/helper"
	"TaskFlow-Go/internal/job"
	"TaskFlow-Go/internal/models"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
	"TaskFlow-Go/internal/validator"

	"github.com/google/uuid"
)

type workspaceService struct {
	workspaceRepo      repoInterface.WorkspaceRepository
	roleRepo           repoInterface.RoleRepository
	rolePermissionRepo repoInterface.RolePermissionRepository
	dispatcher         *job.Dispatcher
}

func NewWorkspaceService(
	workspaceRepo repoInterface.WorkspaceRepository,
	roleRepo repoInterface.RoleRepository,
	rolePermissionRepo repoInterface.RolePermissionRepository,
	dispatcher *job.Dispatcher,
) _interface.WorkspaceService {
	return &workspaceService{
		workspaceRepo:      workspaceRepo,
		roleRepo:           roleRepo,
		rolePermissionRepo: rolePermissionRepo,
		dispatcher:         dispatcher,
	}
}

func (s *workspaceService) GetWorkspacesByUserId(userID string) (*dto.WorkspaceListResponse, error) {
	workspaces, total, err := s.workspaceRepo.ListByUserIDWithSummary(userID)
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get workspaces")
	}
	return &dto.WorkspaceListResponse{Data: workspaces, Total: total}, nil
}

func (s *workspaceService) CreateWorkspace(userID string, req *dto.CreateWorkspaceRequest) (*dto.WorkspaceCreateResponse, error) {
	if err := validator.ValidateWorkspaceName(req.Name); err != nil {
		return nil, apperror.NewAppError(400, "VALIDATION_ERROR", err.Error())
	}

	if req.Domain != nil {
		if err := validator.ValidateWorkspaceDomain(*req.Domain); err != nil {
			return nil, apperror.NewAppError(400, "VALIDATION_ERROR", err.Error())
		}
		existingWorkspace, err := s.workspaceRepo.GetWorkspaceByDomain(*req.Domain)
		if err != nil {
			return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to check workspace domain")
		}
		if helper.IsDomainTaken(existingWorkspace, "") {
			return nil, apperror.ErrDomainAlreadyTaken
		}
	}

	planCounts, err := s.workspaceRepo.CountWorkspaceByPlan(
		helper.AllWorkspacePlans(),
		models.WorkspaceRoleOWNER,
		userID,
	)
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to count workspaces")
	}

	if err := helper.CheckWorkspacePlanLimit(planCounts, models.WorkspacePlanFREE); err != nil {
		return nil, err
	}

	workspace := &models.Workspace{
		OwnerID: userID,
		Name:    req.Name,
		Domain:  req.Domain,
	}
	if err := s.workspaceRepo.Create(workspace); err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to create workspace")
	}

	if err := s.workspaceRepo.AddMember(&models.WorkspaceMember{
		WorkspaceID: workspace.ID,
		UserID:      userID,
		Role:        models.WorkspaceRoleOWNER,
		JoinedAt:    time.Now(),
	}); err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to add workspace owner")
	}

	// BR-ROLE-08: Seed default roles (Manager, Developer, Viewer)
	if err := s.seedDefaultRoles(workspace.ID); err != nil {
		return nil, err
	}

	return &dto.WorkspaceCreateResponse{
		ID:        workspace.ID,
		Name:      workspace.Name,
		Domain:    workspace.Domain,
		Plan:      string(workspace.Plan),
		OwnerID:   workspace.OwnerID,
		CreatedAt: workspace.CreatedAt,
	}, nil
}

func (s *workspaceService) GetWorkspaceById(workspaceID string, userID string) (*dto.WorkspaceDetailResponse, error) {
	workspace, err := s.workspaceRepo.GetByIDWithDetail(workspaceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrWorkspaceNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get workspace")
	}

	if _, err := s.workspaceRepo.GetMember(workspaceID, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotAMember
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to verify membership")
	}

	workspace.IsOverLimit = helper.IsWorkspaceOverLimit(
		models.WorkspacePlan(workspace.Plan),
		workspace.MemberCount,
		workspace.ProjectCount,
		0,
	)

	return workspace, nil
}

func (s *workspaceService) UpdateWorkspace(workspaceID string, userID string, req *dto.UpdateWorkspaceRequest) (*dto.UpdateWorkspaceResponse, error) {
	workspace, err := s.workspaceRepo.GetByID(workspaceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrWorkspaceNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get workspace")
	}

	member, err := s.workspaceRepo.GetMember(workspaceID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrWorkspaceNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to verify membership")
	}
	if member.Role != models.WorkspaceRoleOWNER && member.Role != models.WorkspaceRoleADMIN {
		return nil, apperror.ErrForbidden
	}

	if req.Name != nil {
		if err := validator.ValidateWorkspaceName(*req.Name); err != nil {
			return nil, apperror.NewAppError(400, "VALIDATION_ERROR", err.Error())
		}
		workspace.Name = *req.Name
	}

	if req.Domain != nil {
		if *req.Domain != "" {
			if err := validator.ValidateWorkspaceDomain(*req.Domain); err != nil {
				return nil, apperror.NewAppError(400, "VALIDATION_ERROR", err.Error())
			}
			existingWorkspace, err := s.workspaceRepo.GetWorkspaceByDomain(*req.Domain)
			if err != nil {
				return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to check workspace domain")
			}
			if helper.IsDomainTaken(existingWorkspace, workspaceID) {
				return nil, apperror.ErrDomainAlreadyTaken
			}
			workspace.Domain = req.Domain
		} else {
			workspace.Domain = nil
		}
	}

	workspace.UpdatedAt = time.Now()
	if err := s.workspaceRepo.Update(workspace); err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to update workspace")
	}

	return &dto.UpdateWorkspaceResponse{
		ID:        workspace.ID,
		Name:      workspace.Name,
		Domain:    workspace.Domain,
		Plan:      string(workspace.Plan),
		UpdatedAt: workspace.UpdatedAt,
	}, nil
}

func (s *workspaceService) UpgradePlan(workspaceID string, userID string, req *dto.UpgradePlanRequest) (*dto.PlanUpgradeResponse, error) {
	workspace, err := s.workspaceRepo.GetByID(workspaceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrWorkspaceNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get workspace")
	}

	member, err := s.workspaceRepo.GetMember(workspaceID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrWorkspaceNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to verify membership")
	}
	if member.Role != models.WorkspaceRoleOWNER {
		return nil, apperror.ErrForbidden
	}

	newPlan := models.WorkspacePlan(req.NewPlan)
	switch newPlan {
	case models.WorkspacePlanFREE, models.WorkspacePlanPRO, models.WorkspacePlanENTERPRISE:
	default:
		return nil, apperror.ErrInvalidPlan
	}

	if workspace.Plan == newPlan {
		return nil, apperror.ErrAlreadyOnThisPlan
	}

	planRank := map[models.WorkspacePlan]int{
		models.WorkspacePlanFREE:       0,
		models.WorkspacePlanPRO:        1,
		models.WorkspacePlanENTERPRISE: 2,
	}
	if planRank[newPlan] < planRank[workspace.Plan] {
		return nil, apperror.ErrPlanDowngradeNotAllowed
	}

	prevPlan := workspace.Plan
	workspace.Plan = newPlan
	workspace.UpdatedAt = time.Now()
	if err := s.workspaceRepo.Update(workspace); err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to upgrade plan")
	}

	return &dto.PlanUpgradeResponse{
		ID:           workspace.ID,
		PreviousPlan: string(prevPlan),
		CurrentPlan:  string(workspace.Plan),
		UpgradedAt:   workspace.UpdatedAt,
	}, nil
}

func (s *workspaceService) DeleteWorkspace(workspaceID string, userID string, req *dto.DeleteWorkspaceRequest) (*dto.DeleteWorkspaceResponse, error) {
	workspace, err := s.workspaceRepo.GetByID(workspaceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrWorkspaceNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get workspace")
	}

	member, err := s.workspaceRepo.GetMember(workspaceID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrWorkspaceNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to verify membership")
	}
	if member.Role != models.WorkspaceRoleOWNER {
		return nil, apperror.ErrForbidden
	}

	if strings.TrimSpace(req.ConfirmationName) != strings.TrimSpace(workspace.Name) {
		return nil, apperror.ErrInvalidConfirmation
	}

	if err := s.workspaceRepo.Delete(workspaceID); err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to delete workspace")
	}

	s.dispatcher.CascadeSoftDeleteWorkspace(workspaceID)

	return &dto.DeleteWorkspaceResponse{
		Message: fmt.Sprintf("Workspace '%s' has been deleted successfully.", workspace.Name),
	}, nil
}

// BR-ROLE-08: Seed 3 default roles when workspace is created
func (s *workspaceService) seedDefaultRoles(workspaceID string) error {
	type seedRole struct {
		Name        string
		Description string
		Permissions []string
	}

	allPerms := []string{
		"task:view", "task:create", "task:update", "task:delete", "task:assign", "task:move", "task:set_priority",
		"project:view", "project:update", "project:delete", "project:manage_members", "project:archive",
		"column:create", "column:update", "column:delete",
		"comment:create", "comment:update_own", "comment:delete_own", "comment:delete_any",
		"label:create", "label:update", "label:delete", "label:assign",
		"attachment:upload", "attachment:delete_own", "attachment:delete_any",
	}

	roles := []seedRole{
		{
			Name:        "Manager",
			Description: "Quản lý toàn diện project và thành viên",
			Permissions: allPerms,
		},
		{
			Name:        "Developer",
			Description: "Thành viên thực thi, quản lý task và công việc",
			Permissions: []string{
				"task:view", "task:create", "task:update", "task:delete", "task:assign", "task:move", "task:set_priority",
				"comment:create", "comment:update_own", "comment:delete_own",
				"attachment:upload", "attachment:delete_own",
				"label:assign",
			},
		},
		{
			Name:        "Viewer",
			Description: "Chỉ xem nội dung, không chỉnh sửa",
			Permissions: []string{
				"task:view",
				"comment:create", "comment:update_own", "comment:delete_own",
			},
		},
	}

	for _, r := range roles {
		desc := r.Description
		role := &models.Role{
			ID:          uuid.New().String(),
			WorkspaceID: workspaceID,
			Name:        r.Name,
			Description: &desc,
		}
		if _, err := s.roleRepo.Create(role); err != nil {
			return apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to create default role: "+r.Name)
		}
		if err := s.rolePermissionRepo.BulkCreate(role.ID, r.Permissions); err != nil {
			return apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to assign permissions to default role: "+r.Name)
		}
	}

	return nil
}
