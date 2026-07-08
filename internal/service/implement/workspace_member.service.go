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
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
)

type workspaceMemberService struct {
	memberRepo        repoInterface.WorkspaceMemberRepository
	workspaceRepo     repoInterface.WorkspaceRepository
	userRepo          repoInterface.UserRepository
	projectMemberRepo repoInterface.ProjectMemberRepository
	tm                *database.TransactionManager
	notifRepo         repoInterface.NotificationRepository
	activityLogRepo   repoInterface.ActivityLogRepository
}

func NewWorkspaceMemberService(
	memberRepo repoInterface.WorkspaceMemberRepository,
	workspaceRepo repoInterface.WorkspaceRepository,
	userRepo repoInterface.UserRepository,
	projectMemberRepo repoInterface.ProjectMemberRepository,
	tm *database.TransactionManager,
	notifRepo repoInterface.NotificationRepository,
	activityLogRepo repoInterface.ActivityLogRepository,
) _interface.WorkspaceMemberService {
	return &workspaceMemberService{
		memberRepo:        memberRepo,
		workspaceRepo:     workspaceRepo,
		userRepo:          userRepo,
		projectMemberRepo: projectMemberRepo,
		tm:                tm,
		notifRepo:         notifRepo,
		activityLogRepo:   activityLogRepo,
	}
}

// da review
func (s *workspaceMemberService) ListMembers(workspaceID string, userID string, page int, limit int, search string, role string) ([]dto.MemberInfo, *dto.Pagination, error) {
	members, pagination, err := s.memberRepo.ListWithPagination(workspaceID, search, role, page, limit)
	if err != nil {
		return nil, nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to list members")
	}
	return members, pagination, nil
}

// da review
func (s *workspaceMemberService) GetMemberDetails(workspaceID string, targetUserID string) (*dto.MemberDetailResponse, error) {
	member, err := s.memberRepo.GetByIDWithDetails(workspaceID, targetUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrMemberNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get member details")
	}
	return member, nil
}

// chua review xong
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

	actorName := s.getUserName(userID)
	targetFullName := s.getUserName(targetUserID)
	logMeta := activitylog.WorkspaceRoleChanged(targetUserID, targetFullName, string(targetMember.Role), string(newRole))
	logDesc := activitylog.GenerateDescription(actorName, logMeta)

	workspace, err := s.workspaceRepo.GetByID(workspaceID)
	var logSnap map[string]interface{}
	if err == nil {
		logSnap = activitylog.BuildWorkspaceSnapshot(workspace.Name)
	}

	if err := s.tm.Execute(func(tx *gorm.DB) error {
		if err := s.memberRepo.WithTx(tx).UpdateRole(workspaceID, targetUserID, newRole); err != nil {
			return apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to update role")
		}
		if workspace != nil {
			s.logActivityInTx(tx, workspaceID, "", userID, workspaceID, models.ActivityActionUPDATE, logMeta, logDesc, logSnap)
		}
		return nil
	}); err != nil {
		return nil, err
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

// chua review xong
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

	actorName := s.getUserName(userID)
	targetName := s.getUserName(req.NewOwnerID)
	meta := activitylog.OwnershipTransferred(userID, actorName, req.NewOwnerID, targetName)
	desc := activitylog.GenerateDescription(actorName, meta)
	snap := activitylog.BuildWorkspaceSnapshot(workspace.Name)

	err = s.tm.Execute(func(tx *gorm.DB) error {
		memberTx := s.memberRepo.WithTx(tx)
		wsTx := s.workspaceRepo.WithTx(tx)

		if err := memberTx.UpdateRole(workspaceID, userID, models.WorkspaceRoleADMIN); err != nil {
			return apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to update previous owner role")
		}
		if err := memberTx.UpdateRole(workspaceID, req.NewOwnerID, models.WorkspaceRoleOWNER); err != nil {
			return apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to update new owner role")
		}

		workspace.OwnerID = req.NewOwnerID
		if err := wsTx.Update(workspace); err != nil {
			return apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to update workspace owner")
		}

		s.logActivityInTx(tx, workspaceID, "", userID, workspaceID, models.ActivityActionUPDATE, meta, desc, snap)
		return nil
	})
	if err != nil {
		return nil, err
	}

	notifTitle := "Bạn đã được chuyển quyền OWNER"
	notifContent := fmt.Sprintf("Bạn đã được chuyển quyền OWNER của workspace %s bởi %s.", workspace.Name, actorName)
	notification := &models.Notification{
		ActorID: &userID,
		Type:    models.NotificationTypeANNOUNCEMENT,
		Title:   notifTitle,
		Content: &notifContent,
	}
	if err := s.notifRepo.Create(notification, []string{req.NewOwnerID}); err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to send notification")
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

	workspace, err := s.workspaceRepo.GetByID(workspaceID)
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get workspace")
	}

	actorName := s.getUserName(userID)
	targetName := s.getUserName(targetUserID)
	logMeta := activitylog.WorkspaceMemberRemoved(targetUserID, targetName, string(authMember.Role))
	logDesc := activitylog.GenerateDescription(actorName, logMeta)
	logSnap := activitylog.BuildWorkspaceSnapshot(workspace.Name)

	if err := s.tm.Execute(func(tx *gorm.DB) error {
		notifTitle := fmt.Sprintf("Bạn đã bị xóa khỏi workspace %s", workspace.Name)
		notifContent := fmt.Sprintf("Bạn không còn là thành viên của %s.", workspace.Name)
		notification := &models.Notification{
			ActorID: &userID,
			Type:    models.NotificationTypeANNOUNCEMENT,
			Title:   notifTitle,
			Content: &notifContent,
		}
		if err := s.notifRepo.WithTx(tx).Create(notification, []string{targetUserID}); err != nil {
			return apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to send notification")
		}
		if err := s.memberRepo.WithTx(tx).Delete(workspaceID, targetUserID); err != nil {
			return apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to kick member")
		}
		if err := s.projectMemberRepo.WithTx(tx).DeleteByWorkspace(workspaceID, targetUserID); err != nil {
			return apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to remove member from projects")
		}
		s.logActivityInTx(tx, workspaceID, "", userID, workspaceID, models.ActivityActionDELETE, logMeta, logDesc, logSnap)
		return nil
	}); err != nil {
		return nil, err
	}

	return &dto.KickMemberResponse{
		Message:             "Member has been removed from the workspace.",
		RemovedUserID:       targetUserID,
		RemovedFromProjects: projectIDs,
	}, nil
}

// Da review
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

	if err := s.projectMemberRepo.DeleteByWorkspace(workspaceID, userID); err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to remove from projects")
	}

	return &dto.LeaveWorkspaceResponse{
		Message:         "You have left the workspace successfully.",
		LeftWorkspaceID: workspaceID,
	}, nil
}

func (s *workspaceMemberService) logActivity(workspaceID, projectID, userID, entityID string, action models.ActivityAction, metadata map[string]interface{}, description string, entitySnapshot map[string]interface{}) {
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
	_ = s.activityLogRepo.Create(&models.ActivityLog{
		WorkspaceID:    &wsID,
		UserID:         &uID,
		Action:         action,
		EntityType:     models.EntityTypeWORKSPACE,
		EntityID:       entityID,
		Description:    descPtr,
		Metadata:       metaStr,
		EntitySnapshot: snapStr,
	})
}

func (s *workspaceMemberService) logActivityInTx(tx *gorm.DB, workspaceID, projectID, userID, entityID string, action models.ActivityAction, metadata map[string]interface{}, description string, entitySnapshot map[string]interface{}) {
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

func (s *workspaceMemberService) getUserName(userID string) string {
	u, err := s.userRepo.GetByID(userID)
	if err != nil {
		return userID
	}
	return u.FullName
}
