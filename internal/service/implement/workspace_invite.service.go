package implement

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/activitylog"
	"TaskFlow-Go/internal/database"
	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/helper"
	"TaskFlow-Go/internal/models"
	"TaskFlow-Go/internal/notif"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
)

type workspaceInviteService struct {
	inviteRepo      repoInterface.WorkspaceInviteRepository
	memberRepo      repoInterface.WorkspaceMemberRepository
	workspaceRepo   repoInterface.WorkspaceRepository
	notifRepo       repoInterface.NotificationRepository
	userRepo        repoInterface.UserRepository
	activityLogRepo repoInterface.ActivityLogRepository
	dispatcher      *notif.Dispatcher
	tm              *database.TransactionManager
}

func NewWorkspaceInviteService(
	inviteRepo repoInterface.WorkspaceInviteRepository,
	memberRepo repoInterface.WorkspaceMemberRepository,
	workspaceRepo repoInterface.WorkspaceRepository,
	notifRepo repoInterface.NotificationRepository,
	userRepo repoInterface.UserRepository,
	activityLogRepo repoInterface.ActivityLogRepository,
	dispatcher *notif.Dispatcher,
	tm *database.TransactionManager,
) _interface.WorkspaceInviteService {
	return &workspaceInviteService{
		inviteRepo:      inviteRepo,
		memberRepo:      memberRepo,
		workspaceRepo:   workspaceRepo,
		notifRepo:       notifRepo,
		userRepo:        userRepo,
		activityLogRepo: activityLogRepo,
		dispatcher:      dispatcher,
		tm:              tm,
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

	// BR-INV-06: Validate expires_at (min 5 phút, max 1 năm)
	if req.ExpiresAt != nil {
		now := time.Now()
		if req.ExpiresAt.Before(now.Add(5 * time.Minute)) {
			return nil, apperror.NewAppError(400, "INVALID_EXPIRES_AT", "expires_at must be at least 5 minutes from now")
		}
		if req.ExpiresAt.After(now.AddDate(1, 0, 0)) {
			return nil, apperror.NewAppError(400, "INVALID_EXPIRES_AT", "expires_at cannot exceed 1 year from now")
		}
	}

	// BR-INV-03: Kiểm tra giới hạn invite link theo plan
	activeCount, err := s.inviteRepo.CountActiveByWorkspaceID(workspaceID)
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to check invite limit")
	}
	if err := helper.CheckInviteLimit(workspace.Plan, activeCount); err != nil {
		return nil, err
	}

	// BR-INV-01: Sinh code với retry + unique check
	code, err := s.generateUniqueCode(workspace.Name)
	if err != nil {
		return nil, err
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

// BR-INV-01: Sinh code với retry (max 5 lần), nếu vẫn trùng tăng length lên 8
func (s *workspaceInviteService) generateUniqueCode(workspaceName string) (string, error) {
	const maxAttempts = 5
	const defaultLength = 6
	const extendedLength = 8

	for length := defaultLength; length <= extendedLength; length += 2 {
		for attempt := 0; attempt < maxAttempts; attempt++ {
			code, err := helper.GenerateInviteCode(workspaceName)
			if err != nil {
				return "", apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to generate invite code")
			}
			count, err := s.inviteRepo.CountByCode(code)
			if err != nil {
				return "", apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to check code uniqueness")
			}
			if count == 0 {
				return code, nil
			}
		}
	}

	// Fallback: lần cuối với extendedLength
	code, err := helper.GenerateInviteCode(workspaceName)
	if err != nil {
		return "", apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to generate invite code")
	}
	return code, nil
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

// BR-INV-05: Luồng Join với Transaction + Activity Log
func (s *workspaceInviteService) JoinWorkspaceByCode(code string, userID string) (*dto.JoinWorkspaceResponse, error) {
	invite, err := s.inviteRepo.GetByCode(code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrInviteNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get invite")
	}

	// BR-INV-02: Validate status theo thứ tự ưu tiên
	if invite.DeletedAt.Valid {
		return nil, apperror.ErrInviteRevoked
	}
	if invite.ExpiresAt != nil && invite.ExpiresAt.Before(time.Now()) {
		return nil, apperror.ErrInviteExpired
	}
	if invite.MaxUses != nil && invite.UsesCount >= *invite.MaxUses {
		return nil, apperror.ErrInviteExhausted
	}

	// Check user chưa là member
	if _, err := s.memberRepo.GetByID(invite.WorkspaceID, userID); err == nil {
		return nil, apperror.ErrAlreadyAMember
	}

	workspace, err := s.workspaceRepo.GetByID(invite.WorkspaceID)
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get workspace")
	}

	// Check workspace member limit
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

	// BR-INV-07: Lấy thông tin user join và creator
	joiningUser, _ := s.userRepo.GetByID(userID)
	userName := userID
	if joiningUser != nil {
		userName = joiningUser.FullName
	}

	createdByName := "Hệ thống"
	if invite.CreatedBy != nil {
		creator, _ := s.userRepo.GetByID(*invite.CreatedBy)
		if creator != nil {
			createdByName = creator.FullName
		}
	}

	// BR-INV-05: Transaction
	actorName := s.getUserName(userID)
	meta := activitylog.MemberJoined(userID, role, invite.ID)
	desc := activitylog.GenerateDescription(actorName, meta)

	err = s.tm.Execute(func(tx *gorm.DB) error {
		memberTx := s.memberRepo.WithTx(tx)
		inviteTx := s.inviteRepo.WithTx(tx)

		// INSERT member
		if err := memberTx.Create(&models.WorkspaceMember{
			WorkspaceID: invite.WorkspaceID,
			UserID:      userID,
			Role:        models.WorkspaceRole(role),
			JoinedAt:    time.Now(),
		}); err != nil {
			return apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to add member")
		}

		// UPDATE uses_count
		if err := inviteTx.IncrementUses(invite.ID); err != nil {
			return apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to increment uses")
		}

		// INSERT activity log
		s.logActivityInTx(tx, invite.WorkspaceID, "", userID, workspace.ID, models.ActivityActionCREATE, meta, desc, nil)
		return nil
	})
	if err != nil {
		return nil, err
	}

	// BR-INV-07: Notification cho user vừa join
	s.dispatcher.DispatchADDEDTOWORKSPACEForUser(&notif.ADDEDTOWORKSPACEForUserInput{
		RecipientID:   userID,
		WorkspaceName: workspace.Name,
		Role:          role,
		WorkspaceID:   workspace.ID,
	})

	// BR-INV-07: Notification cho OWNER và ADMIN (trừ người join)
	adminIDs, _ := s.notifRepo.GetWorkspaceMemberIDsByRoles(workspace.ID, []string{"OWNER", "ADMIN"})
	var filteredIDs []string
	for _, id := range adminIDs {
		if id != userID {
			filteredIDs = append(filteredIDs, id)
		}
	}
	if len(filteredIDs) > 0 {
		s.dispatcher.DispatchADDEDTOWORKSPACEForAdmin(&notif.ADDEDTOWORKSPACEForAdminInput{
			AdminIDs:      filteredIDs,
			UserName:      userName,
			WorkspaceName: workspace.Name,
			Role:          role,
			WorkspaceID:   workspace.ID,
			CreatedByName: createdByName,
		})
	}

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

// BR-INV-04: Quyền thu hồi theo đúng rule
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

	// BR-INV-04: Logic kiểm tra quyền
	switch member.Role {
	case models.WorkspaceRoleOWNER:
		// OWNER được phép thu hồi mọi link
	case models.WorkspaceRoleADMIN:
		// ADMIN chỉ được thu hồi link do chính mình tạo
		if invite.CreatedBy == nil || *invite.CreatedBy != userID {
			return nil, apperror.ErrForbidden
		}
	default:
		return nil, apperror.ErrForbidden
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

func (s *workspaceInviteService) logActivityInTx(tx *gorm.DB, workspaceID, projectID, userID, entityID string, action models.ActivityAction, metadata map[string]interface{}, description string, entitySnapshot map[string]interface{}) {
	wsID := workspaceID
	uID := userID
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
	_ = tx.Create(&models.ActivityLog{
		WorkspaceID:    &wsID,
		UserID:         &uID,
		Action:         action,
		EntityType:     models.EntityTypeWORKSPACE,
		EntityID:       entityID,
		Description:    descPtr,
		Metadata:       metaStr,
		EntitySnapshot: snapStr,
	}).Error
}

func (s *workspaceInviteService) getUserName(userID string) string {
	u, err := s.userRepo.GetByID(userID)
	if err != nil {
		return userID
	}
	return u.FullName
}
