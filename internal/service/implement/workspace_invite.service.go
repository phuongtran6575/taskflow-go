package implement

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/helper"
	"TaskFlow-Go/internal/models"
	"TaskFlow-Go/internal/notif"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
)

type workspaceInviteService struct {
	inviteRepo     repoInterface.WorkspaceInviteRepository
	memberRepo     repoInterface.WorkspaceMemberRepository
	workspaceRepo  repoInterface.WorkspaceRepository
	notifRepo      repoInterface.NotificationRepository
	userRepo       repoInterface.UserRepository
	dispatcher     *notif.Dispatcher
}

func NewWorkspaceInviteService(
	inviteRepo repoInterface.WorkspaceInviteRepository,
	memberRepo repoInterface.WorkspaceMemberRepository,
	workspaceRepo repoInterface.WorkspaceRepository,
	notifRepo repoInterface.NotificationRepository,
	userRepo repoInterface.UserRepository,
	dispatcher *notif.Dispatcher,
) _interface.WorkspaceInviteService {
	return &workspaceInviteService{
		inviteRepo:    inviteRepo,
		memberRepo:    memberRepo,
		workspaceRepo: workspaceRepo,
		notifRepo:     notifRepo,
		userRepo:      userRepo,
		dispatcher:    dispatcher,
	}
}

func (s *workspaceInviteService) ListInvites(workspaceID string, userID string, status string, page int, limit int) ([]dto.InviteInfo, *dto.Pagination, error) {
	if _, err := s.workspaceRepo.GetByID(workspaceID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, apperror.ErrWorkspaceNotFound
		}
		return nil, nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get workspace")
	}

	member, err := s.memberRepo.GetByID(workspaceID, userID)
	if err != nil {
		return nil, nil, apperror.ErrWorkspaceNotFound
	}
	if member.Role != models.WorkspaceRoleOWNER && member.Role != models.WorkspaceRoleADMIN {
		return nil, nil, apperror.ErrForbidden
	}

	invites, pagination, err := s.inviteRepo.ListWithPagination(workspaceID, status, page, limit)
	if err != nil {
		return nil, nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to list invites")
	}
	return invites, pagination, nil
}

func (s *workspaceInviteService) CreateInvite(workspaceID string, userID string, req *dto.CreateInviteRequest) (*dto.InviteCreateResponse, error) {
	workspace, err := s.workspaceRepo.GetByID(workspaceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrWorkspaceNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get workspace")
	}

	memberInfo, err := s.memberRepo.GetMemberWithInfor(workspaceID, userID)
	if err != nil {
		return nil, apperror.ErrWorkspaceNotFound
	}

	role := req.Role
	if role == "" {
		role = string(models.WorkspaceRoleMEMBER)
	}
	switch models.WorkspaceRole(role) {
	case models.WorkspaceRoleADMIN, models.WorkspaceRoleMEMBER:
	default:
		return nil, apperror.NewAppError(400, "INVALID_ROLE", "Role must be MEMBER or ADMIN")
	}

	if req.MaxUses != nil && *req.MaxUses < 1 {
		return nil, apperror.NewAppError(400, "INVALID_MAX_USES", "max_uses must be >= 1")
	}

	if req.ExpiresAt != nil && req.ExpiresAt.Before(time.Now()) {
		return nil, apperror.NewAppError(400, "INVALID_EXPIRES_AT", "expires_at cannot be in the past")
	}

	activeCount, err := s.inviteRepo.CountActiveByWorkspaceID(workspaceID)
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to check invite limit")
	}
	if activeCount >= 50 {
		return nil, apperror.ErrInviteLinkLimitReached
	}

	memberCount, err := s.memberRepo.CountMembers(workspaceID)
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to check member count")
	}
	if err := helper.CheckMemberLimit(workspace.Plan, memberCount); err != nil {
		return nil, err
	}

	code, err := helper.GenerateInviteCode()
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to generate invite code")
	}

	now := time.Now()
	invite := &models.WorkspaceInvite{
		WorkspaceID: workspaceID,
		Code:        code,
		Role:        role,
		MaxUses:     req.MaxUses,
		ExpiresAt:   req.ExpiresAt,
		CreatedBy:   &userID,
		CreatedAt:   now,
	}
	if err := s.inviteRepo.Create(invite); err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to create invite")
	}

	return &dto.InviteCreateResponse{
		ID:        invite.ID,
		Code:      invite.Code,
		URL:       fmt.Sprintf("https://app.example.com/invite/%s", invite.Code),
		Role:      invite.Role,
		MaxUses:   invite.MaxUses,
		UsesCount: 0,
		ExpiresAt: invite.ExpiresAt,
		Status:    "ACTIVE",
		CreatedBy: dto.InviteCreatorInfo{
			UserID:   userID,
			FullName: memberInfo.FullName,
		},
		CreatedAt: now,
	}, nil
}

func (s *workspaceInviteService) GetInvitePreview(code string) (*dto.InvitePreviewResponse, error) {
	preview, err := s.inviteRepo.GetByCodeWithPreview(code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrInviteNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get invite preview")
	}
	return preview, nil
}

func (s *workspaceInviteService) JoinWorkspaceByCode(code string, userID string) (*dto.JoinWorkspaceResponse, error) {
	invite, err := s.inviteRepo.GetByCode(code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrInviteNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get invite")
	}

	if invite.DeletedAt.Valid {
		return nil, apperror.ErrInviteRevoked
	}
	if invite.ExpiresAt != nil && invite.ExpiresAt.Before(time.Now()) {
		return nil, apperror.ErrInviteExpired
	}
	if invite.MaxUses != nil && invite.UsesCount >= *invite.MaxUses {
		return nil, apperror.ErrInviteExhausted
	}

	if _, err := s.memberRepo.GetByID(invite.WorkspaceID, userID); err == nil {
		return nil, apperror.ErrAlreadyAMember
	}

	workspace, err := s.workspaceRepo.GetByID(invite.WorkspaceID)
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get workspace")
	}

	memberCount, err := s.memberRepo.CountMembers(invite.WorkspaceID)
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to check member count")
	}
	if err := helper.CheckMemberLimit(workspace.Plan, memberCount); err != nil {
		return nil, err
	}

	role := invite.Role
	if role == "" {
		role = string(models.WorkspaceRoleMEMBER)
	}

	if err := s.memberRepo.Create(&models.WorkspaceMember{
		WorkspaceID: invite.WorkspaceID,
		UserID:      userID,
		Role:        models.WorkspaceRole(role),
		JoinedAt:    time.Now(),
	}); err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to add member")
	}

	if err := s.inviteRepo.IncrementUses(invite.ID); err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to increment uses")
	}

	s.dispatcher.DispatchADDEDTOWORKSPACEForUser(&notif.ADDEDTOWORKSPACEForUserInput{
		RecipientID:   userID,
		WorkspaceName: workspace.Name,
		Role:          role,
		WorkspaceID:   workspace.ID,
	})

	adminIDs, _ := s.notifRepo.GetWorkspaceMemberIDsByRoles(workspace.ID, []string{"OWNER", "ADMIN"})
	user, _ := s.userRepo.GetByID(userID)
	userName := userID
	if user != nil {
		userName = user.FullName
	}
	s.dispatcher.DispatchADDEDTOWORKSPACEForAdmin(&notif.ADDEDTOWORKSPACEForAdminInput{
		AdminIDs:      adminIDs,
		UserName:      userName,
		WorkspaceName: workspace.Name,
		Role:          role,
		WorkspaceID:   workspace.ID,
	})

	return &dto.JoinWorkspaceResponse{
		Message: "You have successfully joined the workspace.",
		Workspace: struct {
			ID     string  `json:"id"`
			Name   string  `json:"name"`
			Domain *string `json:"domain"`
		}{
			ID:     workspace.ID,
			Name:   workspace.Name,
			Domain: workspace.Domain,
		},
		JoinedAsRole: role,
	}, nil
}

func (s *workspaceInviteService) RevokeInvite(workspaceID string, userID string, inviteID string) (*dto.RevokeInviteResponse, error) {
	member, err := s.memberRepo.GetByID(workspaceID, userID)
	if err != nil {
		return nil, apperror.ErrWorkspaceNotFound
	}

	invite, err := s.inviteRepo.GetByID(inviteID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrInviteNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get invite")
	}
	if invite.WorkspaceID != workspaceID {
		return nil, apperror.ErrInviteNotFound
	}
	if invite.DeletedAt.Valid {
		return nil, apperror.ErrAlreadyRevoked
	}

	if member.Role == models.WorkspaceRoleADMIN {
		creatorMember, err := s.memberRepo.GetByID(workspaceID, *invite.CreatedBy)
		if err == nil && creatorMember.Role == models.WorkspaceRoleOWNER {
			return nil, apperror.ErrForbidden
		}
	}

	if err := s.inviteRepo.SoftDelete(inviteID); err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to revoke invite")
	}

	return &dto.RevokeInviteResponse{
		Message:   "Invite link has been revoked.",
		InviteID:  inviteID,
		RevokedAt: time.Now(),
	}, nil
}



