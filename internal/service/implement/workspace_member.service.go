package implement

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/helper"
	"TaskFlow-Go/internal/models"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
)

type workspaceMemberService struct {
	memberRepo    repoInterface.WorkspaceMemberRepository
	workspaceRepo repoInterface.WorkspaceRepository
	userRepo      repoInterface.UserRepository
}

func NewWorkspaceMemberService(
	memberRepo repoInterface.WorkspaceMemberRepository,
	workspaceRepo repoInterface.WorkspaceRepository,
	userRepo repoInterface.UserRepository,
) _interface.WorkspaceMemberService {
	return &workspaceMemberService{
		memberRepo:    memberRepo,
		workspaceRepo: workspaceRepo,
		userRepo:      userRepo,
	}
}

func (s *workspaceMemberService) ListMembers(workspaceID string, userID string, page int, limit int, search string, role string) ([]dto.MemberInfo, *dto.Pagination, error) {
	members, pagination, err := s.memberRepo.ListWithPagination(workspaceID, search, role, page, limit)
	if err != nil {
		return nil, nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to list members")
	}
	return members, pagination, nil
}

func (s *workspaceMemberService) GetMemberDetails(workspaceID string, targetUserID string) (*dto.MemberDetailResponse, error) {
	if _, err := s.memberRepo.GetByID(workspaceID, targetUserID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrMemberNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get member")
	}

	member, err := s.memberRepo.GetByIDWithDetails(workspaceID, targetUserID)
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get member details")
	}
	return member, nil
}

func (s *workspaceMemberService) UpdateMemberRole(workspaceID string, userID string, targetUserID string, req *dto.UpdateMemberRoleRequest) (*dto.UpdateRoleResponse, error) {
	authMember, err := s.memberRepo.GetByID(workspaceID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrWorkspaceNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to verify membership")
	}

	targetMember, err := s.memberRepo.GetByID(workspaceID, targetUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrMemberNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get target member")
	}

	if userID == targetUserID {
		return nil, apperror.ErrCannotChangeOwnRole
	}

	newRole := models.WorkspaceRole(req.Role)
	switch newRole {
	case models.WorkspaceRoleOWNER, models.WorkspaceRoleADMIN, models.WorkspaceRoleMEMBER:
	default:
		return nil, apperror.ErrInvalidRole
	}

	if newRole == models.WorkspaceRoleOWNER {
		return nil, apperror.ErrCannotAssignOwnerRole
	}

	if targetMember.Role == newRole {
		return nil, apperror.ErrSameRole
	}

	if authMember.Role == models.WorkspaceRoleADMIN {
		if targetMember.Role == models.WorkspaceRoleOWNER || targetMember.Role == models.WorkspaceRoleADMIN {
			return nil, apperror.ErrForbidden
		}
	}

	if err := s.memberRepo.UpdateRole(workspaceID, targetUserID, newRole); err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to update role")
	}

	targetInfo, err := s.memberRepo.GetMemberWithInfor(workspaceID, targetUserID)
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get target info")
	}

	return &dto.UpdateRoleResponse{
		UserID:       targetUserID,
		FullName:     targetInfo.FullName,
		PreviousRole: string(targetMember.Role),
		CurrentRole:  string(newRole),
		UpdatedBy:    userID,
		UpdatedAt:    time.Now().Format("2006-01-02T15:04:05Z"),
	}, nil
}

func (s *workspaceMemberService) TransferOwnership(workspaceID string, userID string, req *dto.TransferOwnershipRequest) (*dto.TransferOwnershipResponse, error) {
	if req.NewOwnerID == userID {
		return nil, apperror.ErrCannotTransferToSelf
	}

	if _, err := s.memberRepo.GetByID(workspaceID, req.NewOwnerID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrMemberNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get target member")
	}

	targetUser, err := s.userRepo.GetByID(req.NewOwnerID)
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get target user")
	}
	if !targetUser.IsActive {
		return nil, apperror.ErrTargetAccountDisabled
	}

	workspace, err := s.workspaceRepo.GetByID(workspaceID)
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get workspace")
	}

	newOwnerPlanCounts, err := s.workspaceRepo.CountWorkspaceByPlan(
		helper.AllWorkspacePlans(),
		models.WorkspaceRoleOWNER,
		req.NewOwnerID,
	)
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to count target user's workspaces")
	}

	if err := helper.CheckWorkspacePlanLimit(newOwnerPlanCounts, workspace.Plan); err != nil {
		return nil, err
	}

	callerUser, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get caller user")
	}

	if callerUser.PasswordHash != nil {
		if err := helper.VerifyPassword(req.Confirmation, *callerUser.PasswordHash); err != nil {
			return nil, apperror.ErrInvalidConfirmation
		}
	} else {
		if req.Confirmation != workspace.Name {
			return nil, apperror.ErrInvalidConfirmation
		}
	}

	if err := s.memberRepo.UpdateRole(workspaceID, userID, models.WorkspaceRoleADMIN); err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to update previous owner role")
	}
	if err := s.memberRepo.UpdateRole(workspaceID, req.NewOwnerID, models.WorkspaceRoleOWNER); err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to update new owner role")
	}

	return &dto.TransferOwnershipResponse{
		Message: "Ownership transferred successfully.",
		PreviousOwner: dto.OwnerRoleInfo{
			UserID:  userID,
			NewRole: string(models.WorkspaceRoleADMIN),
		},
		NewOwner: dto.OwnerRoleInfo{
			UserID:  req.NewOwnerID,
			NewRole: string(models.WorkspaceRoleOWNER),
		},
	}, nil
}

func (s *workspaceMemberService) KickMember(workspaceID string, userID string, targetUserID string) (*dto.KickMemberResponse, error) {
	if userID == targetUserID {
		return nil, apperror.ErrCannotKickSelf
	}

	authMember, err := s.memberRepo.GetByID(workspaceID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrWorkspaceNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to verify membership")
	}

	targetMember, err := s.memberRepo.GetByID(workspaceID, targetUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrMemberNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get target member")
	}

	if targetMember.Role == models.WorkspaceRoleOWNER {
		return nil, apperror.ErrCannotKickOwner
	}

	if authMember.Role == models.WorkspaceRoleADMIN && targetMember.Role == models.WorkspaceRoleADMIN {
		return nil, apperror.ErrForbidden
	}

	projectIDs, err := s.memberRepo.GetUserProjectIDsInWorkspace(workspaceID, targetUserID)
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get user projects")
	}

	if err := s.memberRepo.Delete(workspaceID, targetUserID); err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to kick member")
	}

	return &dto.KickMemberResponse{
		Message:             "Member has been removed from the workspace.",
		RemovedUserID:       targetUserID,
		RemovedFromProjects: projectIDs,
	}, nil
}

func (s *workspaceMemberService) LeaveWorkspace(workspaceID string, userID string, req *dto.LeaveWorkspaceRequest) (*dto.LeaveWorkspaceResponse, error) {
	if !req.Confirmation {
		return nil, apperror.ErrConfirmationRequired
	}

	member, err := s.memberRepo.GetByID(workspaceID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrWorkspaceNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get member")
	}
	if member.Role == models.WorkspaceRoleOWNER {
		return nil, apperror.ErrOwnerCannotLeave
	}

	if err := s.memberRepo.Delete(workspaceID, userID); err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to leave workspace")
	}

	return &dto.LeaveWorkspaceResponse{
		Message:         "You have left the workspace successfully.",
		LeftWorkspaceID: workspaceID,
	}, nil
}
