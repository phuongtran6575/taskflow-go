package implement

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/database"
	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
	"TaskFlow-Go/internal/notif"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
)

type taskService struct {
	tm                *database.TransactionManager
	taskRepo          repoInterface.TaskRepository
	taskAssigneeRepo  repoInterface.TaskAssigneeRepository
	taskLabelRepo     repoInterface.TaskLabelRepository
	projectRepo       repoInterface.ProjectRepository
	columnRepo        repoInterface.ColumnRepository
	projectMemberRepo repoInterface.ProjectMemberRepository
	labelRepo         repoInterface.LabelRepository
	workspaceRepo     repoInterface.WorkspaceRepository
	activityLogRepo   repoInterface.ActivityLogRepository
	notifRepo         repoInterface.NotificationRepository
	userRepo          repoInterface.UserRepository
	dispatcher        *notif.Dispatcher
}

func NewTaskService(
	tm *database.TransactionManager,
	taskRepo repoInterface.TaskRepository,
	taskAssigneeRepo repoInterface.TaskAssigneeRepository,
	taskLabelRepo repoInterface.TaskLabelRepository,
	projectRepo repoInterface.ProjectRepository,
	columnRepo repoInterface.ColumnRepository,
	projectMemberRepo repoInterface.ProjectMemberRepository,
	labelRepo repoInterface.LabelRepository,
	workspaceRepo repoInterface.WorkspaceRepository,
	activityLogRepo repoInterface.ActivityLogRepository,
	notifRepo repoInterface.NotificationRepository,
	userRepo repoInterface.UserRepository,
	dispatcher *notif.Dispatcher,
) _interface.TaskService {
	return &taskService{
		tm:                tm,
		taskRepo:          taskRepo,
		taskAssigneeRepo:  taskAssigneeRepo,
		taskLabelRepo:     taskLabelRepo,
		projectRepo:       projectRepo,
		columnRepo:        columnRepo,
		projectMemberRepo: projectMemberRepo,
		labelRepo:         labelRepo,
		workspaceRepo:     workspaceRepo,
		activityLogRepo:   activityLogRepo,
		notifRepo:         notifRepo,
		userRepo:          userRepo,
		dispatcher:        dispatcher,
	}
}

func (s *taskService) getProjectOrFail(workspaceID, projectID string) (*models.Project, error) {
	project, err := s.projectRepo.GetByID(projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrProjectNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get project")
	}
	if project.DeletedAt.Valid {
		return nil, apperror.ErrProjectNotFound
	}
	return project, nil
}

func (s *taskService) getWorkspaceOwner(workspaceID string) (string, error) {
	ws, err := s.workspaceRepo.GetByID(workspaceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", apperror.ErrWorkspaceNotFound
		}
		return "", apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get workspace")
	}
	return ws.OwnerID, nil
}

func (s *taskService) logActivity(workspaceID, projectID, userID, entityID string, action models.ActivityAction, metadata map[string]interface{}) {
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
		EntityType:  models.EntityTypeTASK,
		EntityID:    entityID,
		Metadata:    metaStr,
	})
}

func (s *taskService) getUserName(userID string) string {
	u, err := s.userRepo.GetByID(userID)
	if err != nil {
		return userID
	}
	return u.FullName
}

func (s *taskService) dedupStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; !ok {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

func (s *taskService) validateColumn(projectID, columnID string) error {
	column, err := s.columnRepo.GetByID(columnID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NewAppError(http.StatusBadRequest, "INVALID_COLUMN", "Column does not exist")
		}
		return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get column")
	}
	if column.ProjectID != projectID {
		return apperror.ErrInvalidColumn
	}
	return nil
}

func (s *taskService) getMaxPosition(projectID, columnID string, parentID *string) (float64, error) {
	if parentID != nil {
		return s.taskRepo.GetMaxPositionInParent(*parentID)
	}
	return s.taskRepo.GetMaxPositionInColumn(projectID, columnID)
}

func (s *taskService) validateProjectNotArchived(projectID string) error {
	project, err := s.projectRepo.GetByID(projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.ErrProjectNotFound
		}
		return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get project")
	}
	if project.IsArchived {
		return apperror.ErrProjectArchived
	}
	return nil
}

func (s *taskService) validatePriority(priority string) error {
	if priority == "" {
		return nil
	}
	switch models.TaskPriority(priority) {
	case models.TaskPriorityLOW, models.TaskPriorityMED, models.TaskPriorityHIGH, models.TaskPriorityURGENT:
		return nil
	default:
		return apperror.ErrInvalidPriority
	}
}

func (s *taskService) validateAssignees(workspaceID, projectID string, assigneeIDs []string) ([]string, error) {
	if len(assigneeIDs) == 0 {
		return nil, nil
	}
	deduped := s.dedupStrings(assigneeIDs)

	validIDs, err := s.projectMemberRepo.ListMemberIDs(projectID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to validate assignees")
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
		return nil, &apperror.InvalidAssigneeIDsError{
			AppError:    apperror.NewAppError(http.StatusBadRequest, "INVALID_ASSIGNEE_IDS", "One or more assignees are not project members"),
			InvalidIDs:  invalidIDs,
		}
	}
	return deduped, nil
}

func (s *taskService) validateLabels(projectID string, labelIDs []string) ([]string, error) {
	if len(labelIDs) == 0 {
		return nil, nil
	}
	deduped := s.dedupStrings(labelIDs)

	projectLabels, err := s.labelRepo.ListByProjectID(projectID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get labels")
	}
	valid := make(map[string]struct{}, len(projectLabels))
	for _, l := range projectLabels {
		valid[l.ID] = struct{}{}
	}
	var invalidIDs []string
	for _, lid := range deduped {
		if _, ok := valid[lid]; !ok {
			invalidIDs = append(invalidIDs, lid)
		}
	}
	if len(invalidIDs) > 0 {
		return nil, &apperror.InvalidLabelIDsError{
			AppError:   apperror.NewAppError(http.StatusBadRequest, "INVALID_LABEL_IDS", "One or more labels do not belong to this project"),
			InvalidIDs: invalidIDs,
		}
	}
	return deduped, nil
}

func (s *taskService) ListTasks(workspaceID string, userID string, projectID string, columnID string, priority string, assigneeID string, labelID string, dueDateFrom string, dueDateTo string, hasAssignee *bool, hasLabel *bool, search string, page int, limit int) ([]dto.TaskSummary, *dto.Pagination, error) {
	page = max(page, 1)
	if limit <= 0 {
		limit = 50
	}

	filters := make(map[string]interface{})
	if columnID != "" {
		filters["column_id"] = columnID
	}
	if priority != "" {
		filters["priority"] = priority
	}
	if assigneeID != "" {
		filters["assignee_id"] = assigneeID
	}
	if labelID != "" {
		filters["label_id"] = labelID
	}
	if dueDateFrom != "" {
		filters["due_date_from"] = dueDateFrom
	}
	if dueDateTo != "" {
		filters["due_date_to"] = dueDateTo
	}
	if hasAssignee != nil {
		filters["has_assignee"] = *hasAssignee
	}
	if hasLabel != nil {
		filters["has_label"] = *hasLabel
	}
	if search != "" {
		filters["search"] = search
	}

	tasks, pagination, err := s.taskRepo.ListWithFilters(projectID, filters, page, limit)
	if err != nil {
		return nil, nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list tasks")
	}
	if len(tasks) == 0 {
		return tasks, pagination, nil
	}

	taskIDs := make([]string, len(tasks))
	for i := range tasks {
		taskIDs[i] = tasks[i].ID
	}

	assigneeMap, err := s.taskAssigneeRepo.ListByTaskIDs(taskIDs)
	if err != nil {
		return nil, nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load assignees")
	}

	labelMap, err := s.taskLabelRepo.ListByTaskIDs(taskIDs)
	if err != nil {
		return nil, nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load labels")
	}

	doneColumnIDs := make(map[string]bool)
	cols, _ := s.columnRepo.ListByProjectID(projectID)
	for _, c := range cols {
		if c.IsDone {
			doneColumnIDs[c.ID] = true
		}
	}

	now := time.Now()
	for i := range tasks {
		tasks[i].Assignees = assigneeMap[tasks[i].ID]
		tasks[i].Labels = labelMap[tasks[i].ID]
		if tasks[i].DueDate != nil && tasks[i].DueDate.Before(now) && !doneColumnIDs[tasks[i].Column.ID] {
			tasks[i].IsOverdue = true
		}
	}

	return tasks, pagination, nil
}

func (s *taskService) CreateTask(workspaceID string, userID string, projectID string, req *dto.CreateTaskRequest) (*dto.TaskCreateResponse, error) {
	if req.Title == "" || req.ColumnID == "" {
		return nil, apperror.ErrValidation
	}

	if err := s.validateProjectNotArchived(projectID); err != nil {
		return nil, err
	}

	if err := s.validateColumn(projectID, req.ColumnID); err != nil {
		return nil, err
	}

	if err := s.validatePriority(req.Priority); err != nil {
		return nil, err
	}

	if req.StartDate != nil && req.DueDate != nil && req.StartDate.After(*req.DueDate) {
		return nil, apperror.ErrInvalidDateRange
	}

	validAssigneeIDs, err := s.validateAssignees(workspaceID, projectID, req.AssigneeIDs)
	if err != nil {
		return nil, err
	}

	validLabelIDs, err := s.validateLabels(projectID, req.LabelIDs)
	if err != nil {
		return nil, err
	}

	priority := models.TaskPriority(req.Priority)
	if req.Priority == "" {
		priority = models.TaskPriorityMED
	}

	var task models.Task
	err = s.tm.Execute(func(tx *gorm.DB) error {
		var project models.Project
		if err := tx.Set("gorm:query_option", "FOR UPDATE").
			Where("id = ?", projectID).First(&project).Error; err != nil {
			return err
		}
		nextNum := project.LastTaskNumber + 1
		if err := tx.Model(&project).Update("last_task_number", nextNum).Error; err != nil {
			return err
		}

		maxPos, err := s.getMaxPosition(projectID, req.ColumnID, nil)
		if err != nil {
			return err
		}
		newPos := maxPos + 1000

		task = models.Task{
			ProjectID:   projectID,
			ColumnID:    req.ColumnID,
			CreatorID:   &userID,
			TaskNumber:  nextNum,
			Title:       req.Title,
			Description: req.Description,
			Priority:    priority,
			StartDate:   req.StartDate,
			DueDate:     req.DueDate,
			Position:    newPos,
		}
		if err := tx.Create(&task).Error; err != nil {
			return err
		}

		if len(validLabelIDs) > 0 {
			var taskLabels []models.TaskLabel
			for _, lid := range validLabelIDs {
				taskLabels = append(taskLabels, models.TaskLabel{TaskID: task.ID, LabelID: lid})
			}
			if err := tx.Create(&taskLabels).Error; err != nil {
				return err
			}
		}

		if len(validAssigneeIDs) > 0 {
			var taskAssignees []models.TaskAssignee
			for _, aid := range validAssigneeIDs {
				taskAssignees = append(taskAssignees, models.TaskAssignee{
					TaskID: task.ID, UserID: aid, AssignedByID: &userID,
				})
			}
			if err := tx.Create(&taskAssignees).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create task")
	}

	resp, err := s.taskRepo.GetCreateTaskResponse(task.ID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get create response")
	}
	resp.Assignees, err = s.taskAssigneeRepo.ListByTaskID(task.ID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get assignees")
	}
	resp.Labels, err = s.taskLabelRepo.ListByTaskID(task.ID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get labels")
	}

	s.logActivity(workspaceID, projectID, userID, task.ID, models.ActivityActionCREATE, map[string]interface{}{
		"task_number": task.TaskNumber,
		"title":       task.Title,
		"column_id":   task.ColumnID,
	})

	taskRef := s.dispatcher.FormatTaskRef(resp.Column.Title, task.TaskNumber)
	actorName := s.getUserName(userID)
	for _, aid := range validAssigneeIDs {
		s.dispatcher.DispatchASSIGNED(&notif.ASSIGNEDInput{
			ActorID:     userID,
			ActorName:   actorName,
			RecipientID: aid,
			TaskRef:     taskRef,
			TaskTitle:   task.Title,
			ProjectName: projectID,
			DueDate:     task.DueDate,
			WorkspaceID: workspaceID,
			ProjectID:   projectID,
			TaskID:      task.ID,
		})
	}

	return resp, nil
}

func (s *taskService) GetTaskById(workspaceID string, userID string, projectID string, taskID string) (*dto.TaskDetailResponse, error) {
	task, err := s.taskRepo.GetByIDWithDetail(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrTaskNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get task")
	}
	if task.Project.ID != projectID {
		return nil, apperror.ErrTaskNotFound
	}

	task.Assignees, err = s.taskAssigneeRepo.ListByTaskID(taskID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get assignees")
	}
	task.Labels, err = s.taskLabelRepo.ListByTaskID(taskID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get labels")
	}
	return task, nil
}

func (s *taskService) UpdateTask(workspaceID string, userID string, projectID string, taskID string, req *dto.UpdateTaskRequest) (*dto.UpdateTaskResponse, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrTaskNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get task")
	}
	if task.ProjectID != projectID {
		return nil, apperror.ErrTaskNotFound
	}

	project, err := s.projectRepo.GetByID(projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrProjectNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get project")
	}
	if project.IsArchived {
		return nil, apperror.ErrProjectArchived
	}

	var changes []dto.FieldChange

	if req.Title != nil {
		if *req.Title == "" {
			return nil, apperror.NewAppError(http.StatusBadRequest, "VALIDATION_ERROR", "Title cannot be empty")
		}
		if *req.Title != task.Title {
			changes = append(changes, dto.FieldChange{Field: "title", OldValue: task.Title, NewValue: *req.Title})
			task.Title = *req.Title
		}
	}

	if req.Description != nil {
		oldVal := task.Description
		newVal := *req.Description
		task.Description = req.Description
		if (oldVal == nil && newVal != "") || (oldVal != nil && *oldVal != newVal) {
			changes = append(changes, dto.FieldChange{Field: "description", OldValue: oldVal, NewValue: newVal})
		}
	}

	if req.Priority != nil {
		if err := s.validatePriority(*req.Priority); err != nil {
			return nil, err
		}
		if *req.Priority != string(task.Priority) {
			changes = append(changes, dto.FieldChange{Field: "priority", OldValue: string(task.Priority), NewValue: *req.Priority})
			task.Priority = models.TaskPriority(*req.Priority)
		}
	}

	if req.StartDate != nil || req.DueDate != nil {
		newStart, newDue := req.StartDate, req.DueDate
		if newStart == nil {
			newStart = task.StartDate
		}
		if newDue == nil {
			newDue = task.DueDate
		}
		if newStart != nil && newDue != nil && newStart.After(*newDue) {
			return nil, apperror.ErrInvalidDateRange
		}
	}

	if req.StartDate != nil {
		oldVal := task.StartDate
		task.StartDate = req.StartDate
		if (oldVal == nil && req.StartDate != nil) || (oldVal != nil && req.StartDate != nil && !oldVal.Equal(*req.StartDate)) {
			changes = append(changes, dto.FieldChange{Field: "start_date", OldValue: oldVal, NewValue: *req.StartDate})
		}
	}

	if req.DueDate != nil {
		oldVal := task.DueDate
		task.DueDate = req.DueDate
		if (oldVal == nil && req.DueDate != nil) || (oldVal != nil && req.DueDate != nil && !oldVal.Equal(*req.DueDate)) {
			changes = append(changes, dto.FieldChange{Field: "due_date", OldValue: oldVal, NewValue: *req.DueDate})
		}
	}

	if len(changes) == 0 {
		return nil, apperror.NewAppError(http.StatusBadRequest, "VALIDATION_ERROR", "No fields to update")
	}

	task.UpdatedAt = time.Now()
	if err := s.taskRepo.Update(task); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update task")
	}

	changesCopy := make([]dto.FieldChange, len(changes))
	copy(changesCopy, changes)
	s.logActivity(workspaceID, projectID, userID, taskID, models.ActivityActionUPDATE, map[string]interface{}{
		"action":  "UPDATE",
		"changes": changesCopy,
	})

	return &dto.UpdateTaskResponse{
		ID:          task.ID,
		TaskNumber:  task.TaskNumber,
		TaskRef:     fmt.Sprintf("%s-%d", project.Key, task.TaskNumber),
		Title:       task.Title,
		Description: task.Description,
		Priority:    string(task.Priority),
		StartDate:   task.StartDate,
		DueDate:     task.DueDate,
		UpdatedAt:   task.UpdatedAt,
		Changes:     changes,
	}, nil
}

func (s *taskService) DeleteTask(workspaceID string, userID string, projectID string, taskID string) (*dto.TaskDeleteResponse, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrTaskNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get task")
	}
	if task.ProjectID != projectID {
		return nil, apperror.ErrTaskNotFound
	}

	if err := s.validateProjectNotArchived(projectID); err != nil {
		return nil, err
	}

	project, err := s.projectRepo.GetByID(projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrProjectNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get project")
	}

	subtaskIDs, err := s.taskRepo.ListIDsByParentID(taskID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list subtasks")
	}

	if err := s.taskRepo.CascadeDelete(taskID); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete task")
	}

	s.logActivity(workspaceID, projectID, userID, taskID, models.ActivityActionDELETE, map[string]interface{}{
		"task_number":    task.TaskNumber,
		"title":          task.Title,
		"subtask_count":  len(subtaskIDs),
	})

	return &dto.TaskDeleteResponse{
		Message:              fmt.Sprintf("Task '%s-%d' has been deleted.", project.Key, task.TaskNumber),
		DeletedTaskID:        taskID,
		DeletedSubtasksCount: len(subtaskIDs),
	}, nil
}

func (s *taskService) CreateSubtask(workspaceID string, userID string, projectID string, taskID string, req *dto.CreateSubtaskRequest) (*dto.SubtaskCreateResponse, error) {
	if req.Title == "" {
		return nil, apperror.NewAppError(http.StatusBadRequest, "VALIDATION_ERROR", "Title is required")
	}

	parent, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrTaskNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get parent task")
	}
	if parent.ProjectID != projectID {
		return nil, apperror.ErrTaskNotFound
	}
	if parent.ParentID != nil {
		return nil, apperror.ErrCannotCreateSubtaskOfSubtask
	}

	var subtaskCount int64
	if err := s.taskRepo.CountByParentID(taskID, &subtaskCount); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to count subtasks")
	}
	if subtaskCount >= 100 {
		return nil, apperror.ErrSubtaskLimitReached
	}

	if err := s.validateProjectNotArchived(projectID); err != nil {
		return nil, err
	}

	if err := s.validatePriority(req.Priority); err != nil {
		return nil, err
	}

	validAssigneeIDs, err := s.validateAssignees(workspaceID, projectID, req.AssigneeIDs)
	if err != nil {
		return nil, err
	}

	targetColumnID := parent.ColumnID
	if req.ColumnID != nil && *req.ColumnID != "" {
		if err := s.validateColumn(projectID, *req.ColumnID); err != nil {
			return nil, err
		}
		targetColumnID = *req.ColumnID
	}

	priority := models.TaskPriority(req.Priority)
	if req.Priority == "" {
		priority = models.TaskPriorityMED
	}

	var subtask models.Task
	err = s.tm.Execute(func(tx *gorm.DB) error {
		var project models.Project
		if err := tx.Set("gorm:query_option", "FOR UPDATE").
			Where("id = ?", projectID).First(&project).Error; err != nil {
			return err
		}
		nextNum := project.LastTaskNumber + 1
		if err := tx.Model(&project).Update("last_task_number", nextNum).Error; err != nil {
			return err
		}

		maxPos, err := s.getMaxPosition(projectID, "", &taskID)
		if err != nil {
			return err
		}
		newPos := maxPos + 1000

		subtask = models.Task{
			ProjectID:  projectID,
			ColumnID:   targetColumnID,
			CreatorID:  &userID,
			ParentID:   &taskID,
			TaskNumber: nextNum,
			Title:      req.Title,
			Priority:   priority,
			DueDate:    req.DueDate,
			Position:   newPos,
		}
		if err := tx.Create(&subtask).Error; err != nil {
			return err
		}

		if len(validAssigneeIDs) > 0 {
			var taskAssignees []models.TaskAssignee
			for _, aid := range validAssigneeIDs {
				taskAssignees = append(taskAssignees, models.TaskAssignee{
					TaskID: subtask.ID, UserID: aid, AssignedByID: &userID,
				})
			}
			if err := tx.Create(&taskAssignees).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create subtask")
	}

	column, err := s.columnRepo.GetByID(targetColumnID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get column")
	}

	project, _ := s.projectRepo.GetByID(projectID)

	assignees, err := s.taskAssigneeRepo.ListByTaskID(subtask.ID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get assignees")
	}

	s.logActivity(workspaceID, projectID, userID, subtask.ID, models.ActivityActionCREATE, map[string]interface{}{
		"task_number": subtask.TaskNumber,
		"title":       subtask.Title,
		"parent_id":   taskID,
		"column_id":   subtask.ColumnID,
	})

	return &dto.SubtaskCreateResponse{
		ID:         subtask.ID,
		TaskNumber: subtask.TaskNumber,
		TaskRef:    fmt.Sprintf("%s-%d", project.Key, subtask.TaskNumber),
		Title:      subtask.Title,
		Priority:   string(subtask.Priority),
		DueDate:    subtask.DueDate,
		Column:     dto.ColumnRef{ID: column.ID, Title: column.Title},
		Parent: dto.TaskParentRef{
			ID:         parent.ID,
			TaskNumber: parent.TaskNumber,
			TaskRef:    fmt.Sprintf("%s-%d", project.Key, parent.TaskNumber),
			Title:      parent.Title,
		},
		Assignees: assignees,
		Position:  subtask.Position,
		CreatedAt: subtask.CreatedAt,
	}, nil
}

func (s *taskService) ListSubtasks(workspaceID string, userID string, projectID string, taskID string) (*dto.SubtaskListResponse, error) {
	resp, err := s.taskRepo.ListSubtasks(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrTaskNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list subtasks")
	}
	if len(resp.Data) > 0 {
		subtaskIDs := make([]string, len(resp.Data))
		for i := range resp.Data {
			subtaskIDs[i] = resp.Data[i].ID
		}
		assigneeMap, err := s.taskAssigneeRepo.ListByTaskIDs(subtaskIDs)
		if err != nil {
			return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load assignees")
		}
		for i := range resp.Data {
			resp.Data[i].Assignees = assigneeMap[resp.Data[i].ID]
		}
	}
	return resp, nil
}

func (s *taskService) GetMyTasks(workspaceID string, userID string, priority string, projectID string, columnID string, dueDateFrom string, dueDateTo string, overdue *bool, includeArchived bool, includeSubtasks bool, search string, sortBy string, sortDir string, page int, limit int) (*dto.MyTaskListResponse, error) {
	page = max(page, 1)
	if limit <= 0 {
		limit = 20
	}

	filters := make(map[string]interface{})
	if priority != "" {
		filters["priority"] = priority
	}
	if projectID != "" {
		filters["project_id"] = projectID
	}
	if columnID != "" {
		filters["column_id"] = columnID
	}
	if dueDateFrom != "" {
		filters["due_date_from"] = dueDateFrom
	}
	if dueDateTo != "" {
		filters["due_date_to"] = dueDateTo
	}
	if overdue != nil {
		filters["overdue"] = *overdue
	}
	if !includeArchived {
		filters["exclude_archived"] = true
	}
	if !includeSubtasks {
		filters["exclude_subtasks"] = true
	}
	if search != "" {
		filters["search"] = search
	}

	if sortBy == "" {
		sortBy = "br_default"
	}

	tasks, summary, pagination, err := s.taskAssigneeRepo.ListMyTasks(userID, workspaceID, filters, page, limit, sortBy, sortDir)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get my tasks")
	}

	for i := range tasks {
		if tasks[i].DueDate != nil && tasks[i].DueDate.Before(time.Now()) {
			tasks[i].IsOverdue = true
		}
	}
	return &dto.MyTaskListResponse{Data: tasks, Pagination: *pagination, Summary: summary}, nil
}

func (s *taskService) SearchTasks(workspaceID string, userID string, projectID string, search string, priority string, assigneeID string, labelID string, creatorID string, columnID string, dueDateFrom string, dueDateTo string, hasAssignee *bool, hasLabel *bool, overdue *bool, includeSubtasks bool, sortBy string, sortDir string, page int, limit int) (*dto.TaskSearchResponse, error) {
	filters := make(map[string]interface{})

	if priority != "" {
		filters["priority"] = []string{priority}
	}
	if assigneeID != "" {
		filters["assignee_id"] = []string{assigneeID}
	}
	if labelID != "" {
		filters["label_id"] = []string{labelID}
	}
	if creatorID != "" {
		filters["creator_id"] = creatorID
	}
	if columnID != "" {
		filters["column_id"] = columnID
	}
	if dueDateFrom != "" {
		filters["due_date_from"] = dueDateFrom
	}
	if dueDateTo != "" {
		filters["due_date_to"] = dueDateTo
	}
	if hasAssignee != nil {
		filters["has_assignee"] = *hasAssignee
	}
	if hasLabel != nil {
		filters["has_label"] = *hasLabel
	}
	if overdue != nil {
		filters["overdue"] = *overdue
	}
	if search != "" {
		filters["search"] = search
	}
	if includeSubtasks {
		filters["include_subtasks"] = true
	}

	tasks, pagination, err := s.taskRepo.SearchWithFilters(projectID, filters, page, limit, sortBy, sortDir)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to search tasks")
	}

	filtersApplied := make(map[string]interface{})
	if search != "" {
		filtersApplied["search"] = search
	}
	if priority != "" {
		filtersApplied["priority"] = []string{priority}
	}
	if assigneeID != "" {
		filtersApplied["assignee_id"] = []string{assigneeID}
	}
	if labelID != "" {
		filtersApplied["label_id"] = []string{labelID}
	}
	if creatorID != "" {
		filtersApplied["creator_id"] = creatorID
	}
	if columnID != "" {
		filtersApplied["column_id"] = columnID
	}
	if dueDateFrom != "" {
		filtersApplied["due_date_from"] = dueDateFrom
	}
	if dueDateTo != "" {
		filtersApplied["due_date_to"] = dueDateTo
	}
	if hasAssignee != nil {
		filtersApplied["has_assignee"] = *hasAssignee
	}
	if hasLabel != nil {
		filtersApplied["has_label"] = *hasLabel
	}
	if overdue != nil {
		filtersApplied["overdue"] = *overdue
	}

	return &dto.TaskSearchResponse{
		Data:           tasks,
		Pagination:     *pagination,
		FiltersApplied: filtersApplied,
	}, nil
}
