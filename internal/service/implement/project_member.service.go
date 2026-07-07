package implement

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/activitylog"
	"TaskFlow-Go/internal/database"
	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
	"TaskFlow-Go/internal/notif"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
)

type projectMemberService struct {
	tm                 *database.TransactionManager
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
	tm *database.TransactionManager,
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
		tm:              tm,
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
func (s *projectMemberService) logActivity(workspaceID, projectID, userID, entityID string, action models.ActivityAction, metadata map[string]interface{}, description string, entitySnapshot map[string]interface{}) {
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
		ProjectID:      &projectID,
		UserID:         &uID,
		Action:         action,
		EntityType:     models.EntityTypePROJECT,
		EntityID:       entityID,
		Description:    descPtr,
		Metadata:       metaStr,
		EntitySnapshot: snapStr,
	})
}

func (s *projectMemberService) logActivityInTx(tx *gorm.DB, workspaceID, projectID, userID, entityID string, action models.ActivityAction, metadata map[string]interface{}, description string, entitySnapshot map[string]interface{}) {
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
		ProjectID:      &projectID,
		UserID:         &uID,
		Action:         action,
		EntityType:     models.EntityTypePROJECT,
		EntityID:       entityID,
		Description:    descPtr,
		Metadata:       metaStr,
		EntitySnapshot: snapStr,
	}).Error
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

	actorName := s.getUserName(userID)
	addedUserInfo := make([]map[string]string, 0, len(toAdd))
	for _, m := range toAdd {
		addedUserInfo = append(addedUserInfo, map[string]string{
			"user_id": m.UserID,
			"role_id": m.RoleID,
		})
	}
	meta := activitylog.ProjectMemberAdded(addedUserInfo)
	desc := activitylog.GenerateDescription(actorName, meta)
	snap := activitylog.BuildProjectSnapshot(project.Name, project.Key)

	var createdMembers []models.ProjectMember
	err = s.tm.Execute(func(tx *gorm.DB) error {
		var innerErr error
		createdMembers, innerErr = s.memberRepo.BulkAddMember(projectID, toAdd)
		if innerErr != nil {
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to add members")
		}
		s.logActivityInTx(tx, workspaceID, projectID, userID, projectID, models.ActivityActionCREATE, meta, desc, snap)
		return nil
	})
	if err != nil {
		return nil, err
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
	}

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

	prevRoleName := ""
	if prevRoleRef != nil {
		prevRoleName = prevRoleRef.Name
	}
	actorName := s.getUserName(userID)
	targetFullName := s.getUserName(targetUserID)
	meta := activitylog.ProjectRoleChanged(targetUserID, targetFullName, prevRoleName, req.RoleID)
	desc := activitylog.GenerateDescription(actorName, meta)
	snap := activitylog.BuildProjectSnapshot(project.Name, project.Key)

	err = s.tm.Execute(func(tx *gorm.DB) error {
		if err := tx.Model(&models.ProjectMember{}).
			Where("project_id = ? AND user_id = ?", projectID, targetUserID).
			Update("role_id", req.RoleID).Error; err != nil {
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update role")
		}
		s.logActivityInTx(tx, workspaceID, projectID, userID, projectID, models.ActivityActionUPDATE, meta, desc, snap)
		return nil
	})
	if err != nil {
		return nil, err
	}

	updatedMember, err := s.memberRepo.GetByIDWithRelationRole(projectID, targetUserID)
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

	newRoleName := ""
	if currentRoleRef != nil {
		newRoleName = currentRoleRef.Name
	}

	// BR-PRA-07: Gửi notification STATUS_CHANGED
	s.sendNotification(
		targetUserID, userID,
		models.NotificationTypeANNOUNCEMENT,
		fmt.Sprintf("Role của bạn trong %s đã thay đổi", project.Name),
		fmt.Sprintf("Role của bạn được đổi từ %s sang %s.", prevRoleName, newRoleName),
	)

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

	actorName := s.getUserName(userID)
	targetFullName := s.getUserName(targetUserID)
	meta := activitylog.ProjectMemberRemoved(targetUserID, targetFullName, "manager")
	desc := activitylog.GenerateDescription(actorName, meta)
	snap := activitylog.BuildProjectSnapshot(project.Name, project.Key)

	err = s.tm.Execute(func(tx *gorm.DB) error {
		if err := tx.Where("project_id = ? AND user_id = ?", projectID, targetUserID).
			Delete(&models.ProjectMember{}).Error; err != nil {
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to remove member")
		}
		s.logActivityInTx(tx, workspaceID, projectID, userID, projectID, models.ActivityActionUPDATE, meta, desc, snap)
		return nil
	})
	if err != nil {
		return nil, err
	}

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

	actorName := s.getUserName(userID)
	selfFullName := s.getUserName(userID)
	meta := activitylog.ProjectMemberRemoved(userID, selfFullName, "self")
	desc := activitylog.GenerateDescription(actorName, meta)
	snap := activitylog.BuildProjectSnapshot(project.Name, project.Key)

	err = s.tm.Execute(func(tx *gorm.DB) error {
		if err := tx.Where("project_id = ? AND user_id = ?", projectID, userID).
			Delete(&models.ProjectMember{}).Error; err != nil {
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to leave project")
		}
		s.logActivityInTx(tx, workspaceID, projectID, userID, projectID, models.ActivityActionUPDATE, meta, desc, snap)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &dto.LeaveProjectResponse{
		Message:       "You have left the project successfully.",
		LeftProjectID: projectID,
	}, nil
}
