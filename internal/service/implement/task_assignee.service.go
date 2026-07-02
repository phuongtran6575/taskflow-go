package implement

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
)

type taskAssigneeService struct {
	taskRepo          repoInterface.TaskRepository
	projectRepo       repoInterface.ProjectRepository
	projectMemberRepo repoInterface.ProjectMemberRepository
	assigneeRepo      repoInterface.TaskAssigneeRepository
}

func NewTaskAssigneeService(
	taskRepo repoInterface.TaskRepository,
	projectRepo repoInterface.ProjectRepository,
	projectMemberRepo repoInterface.ProjectMemberRepository,
	assigneeRepo repoInterface.TaskAssigneeRepository,
) _interface.TaskAssigneeService {
	return &taskAssigneeService{
		taskRepo:          taskRepo,
		projectRepo:       projectRepo,
		projectMemberRepo: projectMemberRepo,
		assigneeRepo:      assigneeRepo,
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

	invalidIDs, err := s.projectMemberRepo.ValidateMembersExist(projectID, req.UserIDs)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to validate members")
	}
	if len(invalidIDs) > 0 {
		return nil, apperror.NewAppError(http.StatusBadRequest, "INVALID_USER_IDS",
			fmt.Sprintf("Users are not project members: %v", invalidIDs))
	}

	existing, err := 	s.assigneeRepo.ListTaskAssigneesByTaskID(taskID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get assignees")
	}
	existingSet := make(map[string]struct{}, len(existing))
	for _, a := range existing {
		existingSet[a.UserID] = struct{}{}
	}

	if len(existing)+len(req.UserIDs) > 20 {
		return nil, apperror.ErrAssigneeLimitReached
	}

	var addedIDs []string
	var skippedIDs []string
	for _, uid := range req.UserIDs {
		if _, ok := existingSet[uid]; ok {
			skippedIDs = append(skippedIDs, uid)
			continue
		}
		assignee := models.TaskAssignee{
			TaskID:       taskID,
			UserID:       uid,
			AssignedByID: &userID,
		}
		if err := s.assigneeRepo.Create(&assignee); err != nil {
			return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to assign member")
		}
		addedIDs = append(addedIDs, uid)
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

	return &dto.AssignMembersResponse{
		TaskID:               taskID,
		TaskRef:              s.getTaskRef(task, project),
		Added:                added,
		SkippedAlreadyAssigned: skippedIDs,
		TotalAssigneesAfter:  len(all),
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
		if err := s.assigneeRepo.Delete(taskID, uid); err != nil {
			return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to unassign member")
		}
		info := infoMap[uid]
		removed = append(removed, dto.UserRef{UserID: uid, FullName: info.FullName})
	}

	remaining, _ := 	s.assigneeRepo.ListTaskAssigneesByTaskID(taskID)

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
		all, _ := 	s.assigneeRepo.ListTaskAssigneesByTaskID(taskID)
		return &dto.SelfAssignResponse{
			TaskID:              taskID,
			TaskRef:             s.getTaskRef(task, project),
			Message:             "You are already assigned to this task.",
			TotalAssigneesAfter: len(all),
		}, nil
	}

	current, err := 	s.assigneeRepo.ListTaskAssigneesByTaskID(taskID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get assignees")
	}
	if len(current) >= 20 {
		return nil, apperror.ErrAssigneeLimitReached
	}

	assignee := models.TaskAssignee{
		TaskID:       taskID,
		UserID:       userID,
		AssignedByID: &userID,
	}
	if err := s.assigneeRepo.Create(&assignee); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to self-assign")
	}

	all, _ := 	s.assigneeRepo.ListTaskAssigneesByTaskID(taskID)

	return &dto.SelfAssignResponse{
		TaskID:              taskID,
		TaskRef:             s.getTaskRef(task, project),
		Message:             "You have been assigned to this task.",
		AssignedAt:          assignee.AssignedAt.Format(time.RFC3339),
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

	_ = s.assigneeRepo.Delete(taskID, userID)

	remaining, _ := 	s.assigneeRepo.ListTaskAssigneesByTaskID(taskID)

	return &dto.SelfUnassignResponse{
		TaskID:              taskID,
		TaskRef:             s.getTaskRef(task, project),
		Message:             "You have been unassigned from this task.",
		TotalAssigneesAfter: len(remaining),
	}, nil
}
