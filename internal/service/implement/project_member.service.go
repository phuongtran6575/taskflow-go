package implement

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
	"TaskFlow-Go/internal/notif"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
)

type projectMemberService struct {
	memberRepo         repoInterface.ProjectMemberRepository
	wsMemberRepo       repoInterface.WorkspaceMemberRepository
	workspaceRepo      repoInterface.WorkspaceRepository
	projectRepo        repoInterface.ProjectRepository
	roleRepo           repoInterface.RoleRepository
	notifRepo          repoInterface.NotificationRepository
	activityLogRepo    repoInterface.ActivityLogRepository
	userRepo           repoInterface.UserRepository
	dispatcher         *notif.Dispatcher
}

func NewProjectMemberService(
	memberRepo repoInterface.ProjectMemberRepository,
	wsMemberRepo repoInterface.WorkspaceMemberRepository,
	workspaceRepo repoInterface.WorkspaceRepository,
	projectRepo repoInterface.ProjectRepository,
	roleRepo repoInterface.RoleRepository,
	notifRepo repoInterface.NotificationRepository,
	activityLogRepo repoInterface.ActivityLogRepository,
	userRepo repoInterface.UserRepository,
	dispatcher *notif.Dispatcher,
) _interface.ProjectMemberService {
	return &projectMemberService{
		memberRepo:      memberRepo,
		wsMemberRepo:    wsMemberRepo,
		workspaceRepo:   workspaceRepo,
		projectRepo:     projectRepo,
		roleRepo:        roleRepo,
		notifRepo:       notifRepo,
		activityLogRepo: activityLogRepo,
		userRepo:        userRepo,
		dispatcher:      dispatcher,
	}
}

func (s *projectMemberService) getProjectOrFail(workspaceID, projectID string) (*models.Project, error) {
	project, err := s.projectRepo.GetByID(projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrProjectNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get project")
	}
	if project.WorkspaceID != workspaceID {
		return nil, apperror.ErrProjectNotFound
	}
	if project.DeletedAt.Valid {
		return nil, apperror.ErrProjectNotFound
	}
	return project, nil
}

func (s *projectMemberService) isWorkspaceOwner(workspaceID, userID string) (bool, error) {
	ws, err := s.workspaceRepo.GetByID(workspaceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, apperror.ErrWorkspaceNotFound
		}
		return false, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get workspace")
	}
	return ws.OwnerID == userID, nil
}

// logActivity ghi activity log cho project member operations (BR-PRA-08)
func (s *projectMemberService) logActivity(workspaceID, projectID, userID string, action models.ActivityAction, metadata map[string]interface{}) {
	wsID := workspaceID
	uID := userID
	var metaStr *string
	if metadata != nil {
		b, _ := json.Marshal(metadata)
		s := string(b)
		metaStr = &s
	}
	_ = s.activityLogRepo.Create(&models.ActivityLog{
		WorkspaceID: &wsID,
		ProjectID:   &projectID,
		UserID:      &uID,
		Action:      action,
		EntityType:  models.EntityTypePROJECT,
		EntityID:    projectID,
		Metadata:    metaStr,
	})
}

func (s *projectMemberService) getUserName(userID string) string {
	u, err := s.userRepo.GetByID(userID)
	if err != nil {
		return userID
	}
	return u.FullName
}

func (s *projectMemberService) sendNotification(recipientID, actorID string, notifType models.NotificationType, title string, content string) {
	now := time.Now()
	notification := &models.Notification{
		ActorID:   &actorID,
		Type:      notifType,
		Title:     title,
		Content:   &content,
		CreatedAt: now,
	}
	_ = s.notifRepo.Create(notification, []string{recipientID})
}

func (s *projectMemberService) ListMembers(workspaceID string, userID string, projectID string, page int, limit int, search string, roleID string) ([]dto.ProjectMemberInfo, *dto.Pagination, error) {
	_, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, nil, err
	}
	return s.memberRepo.ListWithPagination(projectID, search, roleID, page, limit)
}

func (s *projectMemberService) GetAvailableWorkspaceMembers(workspaceID string, userID string, projectID string, search string, page int, limit int) ([]dto.AvailableWorkspaceMember, *dto.Pagination, error) {
	_, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, nil, err
	}
	return s.memberRepo.ListAvailableWorkspaceMembers(workspaceID, projectID, search, page, limit)
}

func (s *projectMemberService) AddMembersToProject(workspaceID string, userID string, projectID string, req *dto.AddMembersRequest) (*dto.AddMembersResponse, error) {
	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if project.IsArchived {
		return nil, apperror.ErrProjectArchived
	}
	if len(req.Members) == 0 {
		return nil, apperror.ErrMembersRequired
	}

	// BR-PRA-05: Giới hạn batch size tối đa 50 members
	if len(req.Members) > 50 {
		return nil, apperror.ErrMemberBatchSizeExceeded
	}

	var userIDs []string
	var roleIDs []string
	roleIDSet := make(map[string]struct{})
	for _, m := range req.Members {
		userIDs = append(userIDs, m.UserID)
		roleIDs = append(roleIDs, m.RoleID)
		roleIDSet[m.RoleID] = struct{}{}
	}

	// Validate user IDs — tất cả phải là workspace members (BR-PRA-05)
	var invalidUserIDs []string
	for _, uid := range userIDs {
		_, err := s.wsMemberRepo.GetByID(workspaceID, uid)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				invalidUserIDs = append(invalidUserIDs, uid)
				continue
			}
			return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to verify workspace membership")
		}
	}
	if len(invalidUserIDs) > 0 {
		return nil, &apperror.InvalidUserIDsError{
			AppError:       apperror.NewAppError(http.StatusBadRequest, "USER_NOT_IN_WORKSPACE", "One or more users are not workspace members"),
			InvalidUserIDs: invalidUserIDs,
		}
	}

	// BR-PRA-04: Validate role IDs thuộc workspace
	invalidRoleIDs, err := s.roleRepo.ValidateRoleIDsBelongToWorkspace(roleIDs, workspaceID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to validate role IDs")
	}
	if len(invalidRoleIDs) > 0 {
		return nil, &apperror.InvalidRoleIDsError{
			AppError:       apperror.NewAppError(http.StatusBadRequest, "INVALID_ROLE_ID", "One or more role IDs do not belong to this workspace"),
			InvalidRoleIDs: invalidRoleIDs,
		}
	}

	existingMembers, err := 	s.memberRepo.ListByProjectID(projectID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get existing members")
	}
	existingSet := make(map[string]struct{}, len(existingMembers))
	for _, m := range existingMembers {
		existingSet[m.UserID] = struct{}{}
	}

	var toAdd []dto.MemberRolePair
	var skipped []string
	for _, m := range req.Members {
		if _, ok := existingSet[m.UserID]; ok {
			skipped = append(skipped, m.UserID)
		} else {
			toAdd = append(toAdd, m)
		}
	}

	if len(toAdd) == 0 {
		return &dto.AddMembersResponse{
			Added:                []dto.AddedMemberInfo{},
			SkippedAlreadyMember: skipped,
			TotalAdded:           0,
		}, nil
	}

	createdMembers, err := s.memberRepo.BulkAddMember(projectID, toAdd)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to add members")
	}

	// Fetch role names for response and notification
	wsRoles, err := s.roleRepo.ListByWorkspaceID(workspaceID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get roles")
	}
	roleMap := make(map[string]string, len(wsRoles))
	for _, r := range wsRoles {
		roleMap[r.ID] = r.Name
	}

	var addedInfos []dto.AddedMemberInfo
	addedUserInfo := make([]map[string]interface{}, 0, len(createdMembers))
	for _, cm := range createdMembers {
		roleName := ""
		if cm.RoleID != nil {
			roleName = roleMap[*cm.RoleID]
		}
		addedInfos = append(addedInfos, dto.AddedMemberInfo{
			UserID:   cm.UserID,
			FullName: cm.User.FullName,
			ProjectRole: &dto.RoleRef{
				ID:   *cm.RoleID,
				Name: roleName,
			},
			JoinedAt: cm.JoinedAt.Format("2006-01-02T15:04:05Z07:00"),
		})

		// BR-PRA-07: Gửi notification ADDED_TO_PROJECT
		s.dispatcher.DispatchADDEDTOPROJECT(&notif.ADDEDTOPROJECTInput{
			ActorID:     userID,
			ActorName:   s.getUserName(userID),
			RecipientID: cm.UserID,
			ProjectName: project.Name,
			RoleName:    roleName,
			WorkspaceID: workspaceID,
			ProjectID:   projectID,
		})

		addedUserInfo = append(addedUserInfo, map[string]interface{}{
			"user_id":   cm.UserID,
			"role_name": roleName,
		})
	}

	// BR-PRA-08: Activity log cho member_added
	s.logActivity(workspaceID, projectID, userID, models.ActivityActionCREATE, map[string]interface{}{
		"event":       "member_added",
		"added_users": addedUserInfo,
	})

	return &dto.AddMembersResponse{
		Added:                addedInfos,
		SkippedAlreadyMember: skipped,
		TotalAdded:           len(addedInfos),
	}, nil
}

func (s *projectMemberService) UpdateMemberRole(workspaceID string, userID string, projectID string, targetUserID string, req *dto.UpdateProjectMemberRoleRequest) (*dto.UpdateProjectMemberRoleResponse, error) {
	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if targetUserID == userID {
		return nil, apperror.ErrCannotChangeOwnRole
	}

	member, err := 	s.memberRepo.GetByIDWithRelationRole(projectID, targetUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrMemberNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get member")
	}

	isOwner, err := s.isWorkspaceOwner(workspaceID, targetUserID)
	if err != nil {
		return nil, err
	}
	if isOwner {
		return nil, apperror.ErrCannotChangeWorkspaceOwnerRole
	}

	invalidRoleIDs, err := s.roleRepo.ValidateRoleIDsBelongToWorkspace([]string{req.RoleID}, workspaceID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to validate role ID")
	}
	if len(invalidRoleIDs) > 0 {
		return nil, apperror.ErrInvalidRoleID
	}

	if member.RoleID != nil && *member.RoleID == req.RoleID {
		return nil, apperror.ErrSameRole
	}

	prevRoleRef := &dto.RoleRef{}
	if member.RoleID != nil && member.Role != nil {
		prevRoleRef = &dto.RoleRef{
			ID:   *member.RoleID,
			Name: member.Role.Name,
		}
	}

	if err := s.memberRepo.UpdateRole(projectID, targetUserID, req.RoleID); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update role")
	}

	updatedMember, err := 	s.memberRepo.GetByIDWithRelationRole(projectID, targetUserID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get updated member")
	}

	currentRoleRef := &dto.RoleRef{}
	if updatedMember.RoleID != nil && updatedMember.Role != nil {
		currentRoleRef = &dto.RoleRef{
			ID:   *updatedMember.RoleID,
			Name: updatedMember.Role.Name,
		}
	}

	// BR-PRA-07: Gửi notification STATUS_CHANGED
	oldRoleName := ""
	if prevRoleRef != nil {
		oldRoleName = prevRoleRef.Name
	}
	newRoleName := ""
	if currentRoleRef != nil {
		newRoleName = currentRoleRef.Name
	}
	s.sendNotification(
		targetUserID, userID,
		models.NotificationTypeANNOUNCEMENT,
		fmt.Sprintf("Role của bạn trong %s đã thay đổi", project.Name),
		fmt.Sprintf("Role của bạn được đổi từ %s sang %s.", oldRoleName, newRoleName),
	)

	// BR-PRA-08: Activity log cho role_changed
	s.logActivity(workspaceID, projectID, userID, models.ActivityActionUPDATE, map[string]interface{}{
		"event":         "role_changed",
		"user_id":       targetUserID,
		"old_role_name": oldRoleName,
		"new_role_name": newRoleName,
	})

	return &dto.UpdateProjectMemberRoleResponse{
		UserID:       member.UserID,
		FullName:     member.User.FullName,
		PreviousRole: prevRoleRef,
		CurrentRole:  currentRoleRef,
		UpdatedBy:    userID,
		UpdatedAt:    updatedMember.JoinedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (s *projectMemberService) RemoveMemberFromProject(workspaceID string, userID string, projectID string, targetUserID string) (*dto.RemoveProjectMemberResponse, error) {
	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if targetUserID == userID {
		return nil, apperror.ErrCannotRemoveSelf
	}

	member, err := s.memberRepo.GetByID(projectID, targetUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrMemberNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get member")
	}

	isOwner, err := s.isWorkspaceOwner(workspaceID, targetUserID)
	if err != nil {
		return nil, err
	}
	if isOwner {
		return nil, apperror.ErrCannotRemoveWorkspaceOwner
	}

	// BR-PRA-07: Gửi notification ANNOUNCEMENT trước khi DELETE
	s.sendNotification(
		targetUserID, userID,
		models.NotificationTypeANNOUNCEMENT,
		fmt.Sprintf("Bạn đã bị xóa khỏi project %s", project.Name),
		fmt.Sprintf("Bạn không còn là thành viên của project %s.", project.Name),
	)

	if err := s.memberRepo.Delete(projectID, targetUserID); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to remove member")
	}

	// BR-PRA-08: Activity log cho member_removed
	s.logActivity(workspaceID, projectID, userID, models.ActivityActionUPDATE, map[string]interface{}{
		"event":      "member_removed",
		"user_id":    targetUserID,
		"removed_by": "manager",
	})

	return &dto.RemoveProjectMemberResponse{
		Message:       "Member has been removed from the project.",
		RemovedUserID: member.UserID,
	}, nil
}

func (s *projectMemberService) LeaveProject(workspaceID string, userID string, projectID string, req *dto.LeaveProjectRequest) (*dto.LeaveProjectResponse, error) {
	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if !req.Confirmation {
		return nil, apperror.ErrConfirmationRequired
	}

	_, err = s.memberRepo.GetByID(projectID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotAProjectMember
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get member")
	}

	isOwner, err := s.isWorkspaceOwner(workspaceID, userID)
	if err != nil {
		return nil, err
	}
	if isOwner {
		return nil, apperror.ErrWorkspaceOwnerCannotLeave
	}

	// BR-PRA-07: Gửi notification ANNOUNCEMENT trước khi DELETE
	s.sendNotification(
		userID, userID,
		models.NotificationTypeANNOUNCEMENT,
		fmt.Sprintf("Bạn đã rời khỏi project %s", project.Name),
		fmt.Sprintf("Bạn không còn là thành viên của project %s.", project.Name),
	)

	if err := s.memberRepo.Delete(projectID, userID); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to leave project")
	}

	// BR-PRA-08: Activity log cho member_removed (self)
	s.logActivity(workspaceID, projectID, userID, models.ActivityActionUPDATE, map[string]interface{}{
		"event":      "member_removed",
		"user_id":    userID,
		"removed_by": "self",
	})

	return &dto.LeaveProjectResponse{
		Message:       "You have left the project successfully.",
		LeftProjectID: projectID,
	}, nil
}
