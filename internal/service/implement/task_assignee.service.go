package implement

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/helper"

	"TaskFlow-Go/internal/activitylog"
	"TaskFlow-Go/internal/database"
	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
	"TaskFlow-Go/internal/notif"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
)

type taskAssigneeService struct {
	taskRepo          repoInterface.TaskRepository
	projectRepo       repoInterface.ProjectRepository
	projectMemberRepo repoInterface.ProjectMemberRepository
	assigneeRepo      repoInterface.TaskAssigneeRepository
	workspaceRepo     repoInterface.WorkspaceRepository
	notifRepo         repoInterface.NotificationRepository
	activityLogRepo   repoInterface.ActivityLogRepository
	userRepo          repoInterface.UserRepository
	dispatcher        *notif.Dispatcher
	tm                *database.TransactionManager
}

func NewTaskAssigneeService(
	taskRepo repoInterface.TaskRepository,
	projectRepo repoInterface.ProjectRepository,
	projectMemberRepo repoInterface.ProjectMemberRepository,
	assigneeRepo repoInterface.TaskAssigneeRepository,
	workspaceRepo repoInterface.WorkspaceRepository,
	notifRepo repoInterface.NotificationRepository,
	activityLogRepo repoInterface.ActivityLogRepository,
	userRepo repoInterface.UserRepository,
	dispatcher *notif.Dispatcher,
	tm *database.TransactionManager,
) _interface.TaskAssigneeService {
	return &taskAssigneeService{
		taskRepo:          taskRepo,
		projectRepo:       projectRepo,
		projectMemberRepo: projectMemberRepo,
		assigneeRepo:      assigneeRepo,
		workspaceRepo:     workspaceRepo,
		notifRepo:         notifRepo,
		activityLogRepo:   activityLogRepo,
		userRepo:          userRepo,
		dispatcher:        dispatcher,
		tm:                tm,
	}
}

func (s *taskAssigneeService) getProjectOrFail(workspaceID, projectID string) (*models.Project, error) {
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

func (s *taskAssigneeService) getTaskOrFail(projectID, taskID string) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrTaskNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get task")
	}
	if task.ProjectID != projectID || task.DeletedAt.Valid {
		return nil, apperror.ErrTaskNotFound
	}
	return task, nil
}

func (s *taskAssigneeService) getTaskRef(task *models.Task, project *models.Project) string {
	return fmt.Sprintf("%s-%d", project.Key, task.TaskNumber)
}

func (s *taskAssigneeService) getWorkspaceOwner(workspaceID string) (string, error) {
	ws, err := s.workspaceRepo.GetByID(workspaceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", apperror.ErrWorkspaceNotFound
		}
		return "", apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get workspace")
	}
	return ws.OwnerID, nil
}

func (s *taskAssigneeService) getUserName(userID string) string {
	u, err := s.userRepo.GetByID(userID)
	if err != nil {
		return userID
	}
	return u.FullName
}

func (s *taskAssigneeService) logActivity(workspaceID, projectID, userID, entityID string, action models.ActivityAction, metadata map[string]interface{}, description string, entitySnapshot map[string]interface{}) {
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
		EntityType:     models.EntityTypeTASK,
		EntityID:       entityID,
		Description:    descPtr,
		Metadata:       metaStr,
		EntitySnapshot: snapStr,
	})
}

func (s *taskAssigneeService) logActivityInTx(tx *gorm.DB, workspaceID, projectID, userID, entityID string, action models.ActivityAction, metadata map[string]interface{}, description string, entitySnapshot map[string]interface{}) {
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
		EntityType:     models.EntityTypeTASK,
		EntityID:       entityID,
		Description:    descPtr,
		Metadata:       metaStr,
		EntitySnapshot: snapStr,
	})
}

func (s *taskAssigneeService) ListAssignees(workspaceID string, userID string, projectID string, taskID string) (*dto.AssigneeListResponse, error) {
	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	task, err := s.getTaskOrFail(projectID, taskID)
	if err != nil {
		return nil, err
	}

	assignees, err := s.assigneeRepo.ListByTaskIDWithDetail(taskID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list assignees")
	}

	return &dto.AssigneeListResponse{
		TaskID:  taskID,
		TaskRef: s.getTaskRef(task, project),
		Data:    assignees,
		Total:   len(assignees),
	}, nil
}

func (s *taskAssigneeService) GetAvailableAssignees(workspaceID string, userID string, projectID string, taskID string, search string, page int, limit int) (*dto.AvailableAssigneeListResponse, error) {
	page = max(page, 1)
	if limit <= 0 {
		limit = 20
	}

	_, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	_, err = s.getTaskOrFail(projectID, taskID)
	if err != nil {
		return nil, err
	}

	data, pagination, err := s.assigneeRepo.ListAvailableForTask(taskID, projectID, search, page, limit)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get available assignees")
	}

	return &dto.AvailableAssigneeListResponse{
		Data:       data,
		Pagination: *pagination,
	}, nil
}

func (s *taskAssigneeService) AssignMembersToTask(workspaceID string, userID string, projectID string, taskID string, req *dto.AssignMembersRequest) (*dto.AssignMembersResponse, error) {
	if len(req.UserIDs) == 0 {
		return nil, apperror.ErrUserIDsRequired
	}

	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if project.IsArchived {
		return nil, apperror.ErrProjectArchived
	}

	task, err := s.getTaskOrFail(projectID, taskID)
	if err != nil {
		return nil, err
	}

	// BR-ASSIGN-02: Validate batch — dedup + check project_members OR workspace OWNER
	deduped := helper.DedupStrings(req.UserIDs)

	validIDs, err := s.projectMemberRepo.ListMemberIDs(projectID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to validate members")
	}
	validMap := make(map[string]struct{}, len(validIDs)+1)
	for _, id := range validIDs {
		validMap[id] = struct{}{}
	}

	wsOwnerID, err := s.getWorkspaceOwner(workspaceID)
	if err == nil && wsOwnerID != "" {
		validMap[wsOwnerID] = struct{}{}
	}

	var invalidIDs []string
	for _, uid := range deduped {
		if _, ok := validMap[uid]; !ok {
			invalidIDs = append(invalidIDs, uid)
		}
	}
	if len(invalidIDs) > 0 {
		return nil, &apperror.InvalidUserIDsError{
			AppError:       apperror.NewAppError(http.StatusBadRequest, "INVALID_USER_IDS", "One or more users are not project members or workspace owners"),
			InvalidUserIDs: invalidIDs,
		}
	}

	existing, err := s.assigneeRepo.ListTaskAssigneesByTaskID(taskID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get assignees")
	}
	existingSet := make(map[string]struct{}, len(existing))
	for _, a := range existing {
		existingSet[a.UserID] = struct{}{}
	}

	// BR-ASSIGN-03: Giới hạn 20 — all-or-nothing
	var toAdd []string
	for _, uid := range deduped {
		if _, ok := existingSet[uid]; !ok {
			toAdd = append(toAdd, uid)
		}
	}
	currentCount := len(existing)
	if currentCount+len(toAdd) > 20 {
		return nil, &apperror.AssigneeLimitError{
			AppError: apperror.NewAppError(http.StatusBadRequest, "ASSIGNEE_LIMIT_REACHED",
				fmt.Sprintf("Maximum 20 assignees per task (current: %d, can add: %d)", currentCount, 20-currentCount)),
			Current: currentCount,
			Limit:   20,
			CanAdd:  20 - currentCount,
		}
	}

	var addedIDs []string
	var skippedIDs []string
	for _, uid := range deduped {
		if _, ok := existingSet[uid]; ok {
			skippedIDs = append(skippedIDs, uid)
			continue
		}
		addedIDs = append(addedIDs, uid)
	}

	// Prepare activity log data (outside transaction)
	actorName := s.getUserName(userID)
	taskRef := s.getTaskRef(task, project)
	addedUsers := make([]map[string]string, 0, len(addedIDs))
	for _, id := range addedIDs {
		addedUsers = append(addedUsers, map[string]string{"user_id": id, "full_name": s.getUserName(id)})
	}
	logMeta := activitylog.AssigneesAdded(addedUsers)
	logDesc := activitylog.GenerateDescription(actorName, logMeta)
	logSnap := activitylog.BuildTaskSnapshot(taskRef, task.Title, project.Key)

	if err := s.tm.Execute(func(tx *gorm.DB) error {
		for _, uid := range addedIDs {
			assignee := models.TaskAssignee{
				TaskID:       taskID,
				UserID:       uid,
				AssignedByID: &userID,
			}
			if err := s.assigneeRepo.WithTx(tx).Create(&assignee); err != nil {
				return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to assign member")
			}
		}
		s.logActivityInTx(tx, workspaceID, projectID, userID, taskID, models.ActivityActionUPDATE, logMeta, logDesc, logSnap)
		return nil
	}); err != nil {
		return nil, err
	}

	all, err := s.assigneeRepo.ListByTaskID(taskID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get assignees")
	}
	infoMap := make(map[string]dto.AssigneeInfo, len(all))
	for _, a := range all {
		infoMap[a.UserID] = a
	}

	added := make([]dto.AddedAssigneeInfo, 0, len(addedIDs))
	for _, id := range addedIDs {
		info := infoMap[id]
		assignedAt := ""
		if info.AssignedAt != nil {
			assignedAt = *info.AssignedAt
		}
		added = append(added, dto.AddedAssigneeInfo{
			UserID:     info.UserID,
			FullName:   info.FullName,
			AvatarURL:  info.AvatarURL,
			AssignedAt: assignedAt,
		})
	}

	// BR-ASSIGN-05: Gửi notification riêng cho từng assignee mới (trừ self)
	for _, aid := range addedIDs {
		s.dispatcher.DispatchASSIGNED(&notif.ASSIGNEDInput{
			ActorID:     userID,
			ActorName:   actorName,
			RecipientID: aid,
			TaskRef:     taskRef,
			TaskTitle:   task.Title,
			ProjectName: project.Name,
			DueDate:     task.DueDate,
			WorkspaceID: workspaceID,
			ProjectID:   projectID,
			TaskID:      taskID,
		})
	}

	return &dto.AssignMembersResponse{
		TaskID:                 taskID,
		TaskRef:                taskRef,
		Added:                  added,
		SkippedAlreadyAssigned: skippedIDs,
		TotalAssigneesAfter:    len(all),
	}, nil
}

func (s *taskAssigneeService) UnassignMembersFromTask(workspaceID string, userID string, projectID string, taskID string, req *dto.UnassignMembersRequest) (*dto.UnassignMembersResponse, error) {
	if len(req.UserIDs) == 0 {
		return nil, apperror.ErrUserIDsRequired
	}

	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if project.IsArchived {
		return nil, apperror.ErrProjectArchived
	}

	task, err := s.getTaskOrFail(projectID, taskID)
	if err != nil {
		return nil, err
	}

	all, err := s.assigneeRepo.ListByTaskID(taskID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get assignees")
	}
	assigneeSet := make(map[string]struct{}, len(all))
	infoMap := make(map[string]dto.AssigneeInfo, len(all))
	for _, a := range all {
		assigneeSet[a.UserID] = struct{}{}
		infoMap[a.UserID] = a
	}

	var removed []dto.UserRef
	var skipped []dto.SkippedUser
	for _, uid := range req.UserIDs {
		if _, ok := assigneeSet[uid]; !ok {
			skipped = append(skipped, dto.SkippedUser{UserID: uid, Reason: "NOT_AN_ASSIGNEE"})
			continue
		}
		info := infoMap[uid]
		removed = append(removed, dto.UserRef{UserID: uid, FullName: info.FullName})
	}

	// Prepare activity log data (outside transaction)
	actorName := s.getUserName(userID)
	taskRef := s.getTaskRef(task, project)
	removedUsers := make([]map[string]string, 0, len(removed))
	for _, r := range removed {
		removedUsers = append(removedUsers, map[string]string{"user_id": r.UserID, "full_name": r.FullName})
	}
	logMeta := activitylog.AssigneesRemoved(removedUsers)
	logDesc := activitylog.GenerateDescription(actorName, logMeta)
	logSnap := activitylog.BuildTaskSnapshot(taskRef, task.Title, project.Key)

	if err := s.tm.Execute(func(tx *gorm.DB) error {
		for _, r := range removed {
			if err := s.assigneeRepo.WithTx(tx).Delete(taskID, r.UserID); err != nil {
				return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to unassign member")
			}
		}
		s.logActivityInTx(tx, workspaceID, projectID, userID, taskID, models.ActivityActionUPDATE, logMeta, logDesc, logSnap)
		return nil
	}); err != nil {
		return nil, err
	}

	remaining, _ := s.assigneeRepo.ListTaskAssigneesByTaskID(taskID)

	return &dto.UnassignMembersResponse{
		TaskID:              taskID,
		TaskRef:             s.getTaskRef(task, project),
		Removed:             removed,
		SkippedNotAssigned:  skipped,
		TotalAssigneesAfter: len(remaining),
	}, nil
}

func (s *taskAssigneeService) SelfAssignToTask(workspaceID string, userID string, projectID string, taskID string) (*dto.SelfAssignResponse, error) {
	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if project.IsArchived {
		return nil, apperror.ErrProjectArchived
	}

	task, err := s.getTaskOrFail(projectID, taskID)
	if err != nil {
		return nil, err
	}

	existing, err := s.assigneeRepo.GetByID(taskID, userID)
	if err == nil && existing != nil {
		all, _ := s.assigneeRepo.ListTaskAssigneesByTaskID(taskID)
		return &dto.SelfAssignResponse{
			TaskID:              taskID,
			TaskRef:             s.getTaskRef(task, project),
			Message:             "You are already assigned to this task.",
			TotalAssigneesAfter: len(all),
		}, nil
	}

	current, err := s.assigneeRepo.ListTaskAssigneesByTaskID(taskID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get assignees")
	}
	if len(current) >= 20 {
		return nil, &apperror.AssigneeLimitError{
			AppError: apperror.NewAppError(http.StatusBadRequest, "ASSIGNEE_LIMIT_REACHED",
				fmt.Sprintf("Maximum 20 assignees per task (current: %d, can add: %d)", len(current), 20-len(current))),
			Current: len(current),
			Limit:   20,
			CanAdd:  20 - len(current),
		}
	}

	actorName := s.getUserName(userID)
	taskRef := s.getTaskRef(task, project)
	logMeta := map[string]interface{}{
		"event":     "self_assigned",
		"user_id":   userID,
		"full_name": actorName,
	}
	logDesc := fmt.Sprintf("%s đã tự gán task %s", actorName, taskRef)
	logSnap := activitylog.BuildTaskSnapshot(taskRef, task.Title, project.Key)

	var assignedAt string
	if err := s.tm.Execute(func(tx *gorm.DB) error {
		assignee := models.TaskAssignee{
			TaskID:       taskID,
			UserID:       userID,
			AssignedByID: &userID,
		}
		if err := s.assigneeRepo.WithTx(tx).Create(&assignee); err != nil {
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to self-assign")
		}
		assignedAt = assignee.AssignedAt.Format(time.RFC3339)
		s.logActivityInTx(tx, workspaceID, projectID, userID, taskID, models.ActivityActionUPDATE, logMeta, logDesc, logSnap)
		return nil
	}); err != nil {
		return nil, err
	}

	all, _ := s.assigneeRepo.ListTaskAssigneesByTaskID(taskID)

	return &dto.SelfAssignResponse{
		TaskID:              taskID,
		TaskRef:             s.getTaskRef(task, project),
		Message:             "You have been assigned to this task.",
		AssignedAt:          assignedAt,
		TotalAssigneesAfter: len(all),
	}, nil
}

func (s *taskAssigneeService) SelfUnassignFromTask(workspaceID string, userID string, projectID string, taskID string) (*dto.SelfUnassignResponse, error) {
	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if project.IsArchived {
		return nil, apperror.ErrProjectArchived
	}

	task, err := s.getTaskOrFail(projectID, taskID)
	if err != nil {
		return nil, err
	}

	actorName := s.getUserName(userID)
	taskRef := s.getTaskRef(task, project)
	logMeta := map[string]interface{}{
		"event":     "self_unassigned",
		"user_id":   userID,
		"full_name": actorName,
	}
	logDesc := fmt.Sprintf("%s đã tự bỏ gán task %s", actorName, taskRef)
	logSnap := activitylog.BuildTaskSnapshot(taskRef, task.Title, project.Key)

	if err := s.tm.Execute(func(tx *gorm.DB) error {
		if err := s.assigneeRepo.WithTx(tx).Delete(taskID, userID); err != nil {
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to unassign")
		}
		s.logActivityInTx(tx, workspaceID, projectID, userID, taskID, models.ActivityActionUPDATE, logMeta, logDesc, logSnap)
		return nil
	}); err != nil {
		return nil, err
	}

	remaining, _ := s.assigneeRepo.ListTaskAssigneesByTaskID(taskID)

	return &dto.SelfUnassignResponse{
		TaskID:              taskID,
		TaskRef:             s.getTaskRef(task, project),
		Message:             "You have been unassigned from this task.",
		TotalAssigneesAfter: len(remaining),
	}, nil
}
