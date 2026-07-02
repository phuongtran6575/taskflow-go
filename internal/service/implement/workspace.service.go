package implement

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/helper"
	"TaskFlow-Go/internal/models"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
	"TaskFlow-Go/internal/validator"
)

type workspaceService struct {
	workspaceRepo repoInterface.WorkspaceRepository
}

func NewWorkspaceService(workspaceRepo repoInterface.WorkspaceRepository) _interface.WorkspaceService {
	return &workspaceService{workspaceRepo: workspaceRepo}
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

	if req.ConfirmationName != workspace.Name {
		return nil, apperror.ErrInvalidConfirmation
	}

	if err := s.workspaceRepo.Delete(workspaceID); err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to delete workspace")
	}

	return &dto.DeleteWorkspaceResponse{
		Message: fmt.Sprintf("Workspace '%s' has been deleted successfully.", workspace.Name),
	}, nil
}
