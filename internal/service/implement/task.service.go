package implement

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
)

type taskService struct {
	taskRepo           repoInterface.TaskRepository
	taskAssigneeRepo   repoInterface.TaskAssigneeRepository
	taskLabelRepo      repoInterface.TaskLabelRepository
	projectRepo        repoInterface.ProjectRepository
	columnRepo         repoInterface.ColumnRepository
	projectMemberRepo  repoInterface.ProjectMemberRepository
	labelRepo          repoInterface.LabelRepository
}

func NewTaskService(
	taskRepo repoInterface.TaskRepository,
	taskAssigneeRepo repoInterface.TaskAssigneeRepository,
	taskLabelRepo repoInterface.TaskLabelRepository,
	projectRepo repoInterface.ProjectRepository,
	columnRepo repoInterface.ColumnRepository,
	projectMemberRepo repoInterface.ProjectMemberRepository,
	labelRepo repoInterface.LabelRepository,
) _interface.TaskService {
	return &taskService{
		taskRepo:          taskRepo,
		taskAssigneeRepo:  taskAssigneeRepo,
		taskLabelRepo:     taskLabelRepo,
		projectRepo:       projectRepo,
		columnRepo:        columnRepo,
		projectMemberRepo: projectMemberRepo,
		labelRepo:         labelRepo,
	}
}

func (s *taskService) ListTasks(workspaceID string, userID string, projectID string, columnID string, priority string, assigneeID string, labelID string, dueDateFrom string, dueDateTo string, hasAssignee *bool, search string, page int, limit int) ([]dto.TaskSummary, *dto.Pagination, error) {
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
	if search != "" {
		filters["search"] = search
	}

	tasks, pagination, err := s.taskRepo.ListWithFilters(projectID, filters, page, limit)
	if err != nil {
		return nil, nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to list tasks")
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
		return nil, nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to load assignees")
	}

	labelMap, err := s.taskLabelRepo.ListByTaskIDs(taskIDs)
	if err != nil {
		return nil, nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to load labels")
	}

	now := time.Now()
	for i := range tasks {
		tasks[i].Assignees = assigneeMap[tasks[i].ID]
		tasks[i].Labels = labelMap[tasks[i].ID]
		if tasks[i].DueDate != nil && tasks[i].DueDate.Before(now) {
			tasks[i].IsOverdue = true
		}
	}

	return tasks, pagination, nil
}

func (s *taskService) CreateTask(workspaceID string, userID string, projectID string, req *dto.CreateTaskRequest) (*dto.TaskCreateResponse, error) {
	if req.Title == "" || req.ColumnID == "" {
		return nil, apperror.ErrValidation
	}

	project, err := s.projectRepo.GetByID(projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrProjectNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get project")
	}
	if project.IsArchived {
		return nil, apperror.ErrProjectArchived
	}

	column, err := s.columnRepo.GetByID(req.ColumnID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewAppError(400, "INVALID_COLUMN", "Column does not exist")
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get column")
	}
	if column.ProjectID != projectID {
		return nil, apperror.ErrInvalidColumn
	}

	if req.Priority != "" {
		switch models.TaskPriority(req.Priority) {
		case models.TaskPriorityLOW, models.TaskPriorityMED, models.TaskPriorityHIGH, models.TaskPriorityURGENT:
		default:
			return nil, apperror.ErrInvalidPriority
		}
	}

	if req.StartDate != nil && req.DueDate != nil && req.StartDate.After(*req.DueDate) {
		return nil, apperror.ErrInvalidDateRange
	}

	if len(req.AssigneeIDs) > 0 {
		invalidIDs, err := s.projectMemberRepo.ValidateMembersExist(projectID, req.AssigneeIDs)
		if err != nil {
			return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to validate assignees")
		}
		if len(invalidIDs) > 0 {
			return nil, apperror.NewAppError(400, "INVALID_ASSIGNEE_IDS",
				fmt.Sprintf("Users are not project members: %v", invalidIDs))
		}
	}

	if len(req.LabelIDs) > 0 {
		projectLabels, err := 	s.labelRepo.ListByProjectID(projectID)
		if err != nil {
			return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get labels")
		}
		valid := make(map[string]struct{}, len(projectLabels))
		for _, l := range projectLabels {
			valid[l.ID] = struct{}{}
		}
		var invalidIDs []string
		for _, lid := range req.LabelIDs {
			if _, ok := valid[lid]; !ok {
				invalidIDs = append(invalidIDs, lid)
			}
		}
		if len(invalidIDs) > 0 {
			return nil, apperror.NewAppError(400, "INVALID_LABEL_IDS",
				fmt.Sprintf("Labels do not belong to this project: %v", invalidIDs))
		}
	}

	nextNum, err := s.taskRepo.GetNextTaskNumber(projectID)
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to generate task number")
	}

	priority := models.TaskPriority(req.Priority)
	if req.Priority == "" {
		priority = models.TaskPriorityMED
	}

	task := models.Task{
		ProjectID:   projectID,
		ColumnID:    req.ColumnID,
		CreatorID:   &userID,
		TaskNumber:  nextNum,
		Title:       req.Title,
		Description: req.Description,
		Priority:    priority,
		StartDate:   req.StartDate,
		DueDate:     req.DueDate,
		Position:    float64(nextNum * 1000),
	}
	if err := s.taskRepo.Create(&task); err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to create task")
	}

	if len(req.LabelIDs) > 0 {
		if err := s.taskLabelRepo.BulkTaskLabel(task.ID, req.LabelIDs); err != nil {
			return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to assign labels")
		}
	}
	if len(req.AssigneeIDs) > 0 {
		if err := s.taskAssigneeRepo.BulkTaskAssignee(task.ID, req.AssigneeIDs, userID); err != nil {
			return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to assign members")
		}
	}

	resp, err := s.taskRepo.GetCreateTaskResponse(task.ID)
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get create response")
	}
	resp.Assignees, err = s.taskAssigneeRepo.ListByTaskID(task.ID)
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get assignees")
	}
	resp.Labels, err = s.taskLabelRepo.ListByTaskID(task.ID)
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get labels")
	}
	return resp, nil
}

func (s *taskService) GetTaskById(workspaceID string, userID string, projectID string, taskID string) (*dto.TaskDetailResponse, error) {
	task, err := s.taskRepo.GetByIDWithDetail(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrTaskNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get task")
	}
	if task.Project.ID != projectID {
		return nil, apperror.ErrTaskNotFound
	}

	task.Assignees, err = s.taskAssigneeRepo.ListByTaskID(taskID)
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get assignees")
	}
	task.Labels, err = s.taskLabelRepo.ListByTaskID(taskID)
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get labels")
	}
	return task, nil
}

func (s *taskService) UpdateTask(workspaceID string, userID string, projectID string, taskID string, req *dto.UpdateTaskRequest) (*dto.UpdateTaskResponse, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrTaskNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get task")
	}
	if task.ProjectID != projectID {
		return nil, apperror.ErrTaskNotFound
	}

	project, err := s.projectRepo.GetByID(projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrProjectNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get project")
	}
	if project.IsArchived {
		return nil, apperror.ErrProjectArchived
	}

	var changes []dto.FieldChange

	if req.Title != nil {
		if *req.Title == "" {
			return nil, apperror.NewAppError(400, "VALIDATION_ERROR", "Title cannot be empty")
		}
		if *req.Title != task.Title {
			changes = append(changes, dto.FieldChange{Field: "title", OldValue: task.Title, NewValue: *req.Title})
			task.Title = *req.Title
		}
	}

	if req.Description != nil {
		if (task.Description == nil && *req.Description != "") || (task.Description != nil && *task.Description != *req.Description) {
			changes = append(changes, dto.FieldChange{Field: "description", OldValue: task.Description, NewValue: *req.Description})
		}
		task.Description = req.Description
	}

	if req.Priority != nil {
		switch models.TaskPriority(*req.Priority) {
		case models.TaskPriorityLOW, models.TaskPriorityMED, models.TaskPriorityHIGH, models.TaskPriorityURGENT:
		default:
			return nil, apperror.ErrInvalidPriority
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
		return nil, apperror.NewAppError(400, "VALIDATION_ERROR", "No fields to update")
	}

	task.UpdatedAt = time.Now()
	if err := s.taskRepo.Update(task); err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to update task")
	}

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
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get task")
	}
	if task.ProjectID != projectID {
		return nil, apperror.ErrTaskNotFound
	}

	project, err := s.projectRepo.GetByID(projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrProjectNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get project")
	}
	if project.IsArchived {
		return nil, apperror.ErrProjectArchived
	}

	var subtaskCount int64
	if err := s.taskRepo.CountByParentID(taskID, &subtaskCount); err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to count subtasks")
	}

	if err := s.taskRepo.Delete(taskID); err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to delete task")
	}

	return &dto.TaskDeleteResponse{
		Message:              fmt.Sprintf("Task '%s-%d' has been deleted.", project.Key, task.TaskNumber),
		DeletedTaskID:        taskID,
		DeletedSubtasksCount: int(subtaskCount),
	}, nil
}

func (s *taskService) CreateSubtask(workspaceID string, userID string, projectID string, taskID string, req *dto.CreateSubtaskRequest) (*dto.SubtaskCreateResponse, error) {
	if req.Title == "" {
		return nil, apperror.NewAppError(400, "VALIDATION_ERROR", "Title is required")
	}

	parent, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrTaskNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get parent task")
	}
	if parent.ProjectID != projectID {
		return nil, apperror.ErrTaskNotFound
	}
	if parent.ParentID != nil {
		return nil, apperror.ErrCannotCreateSubtaskOfSubtask
	}

	var subtaskCount int64
	if err := s.taskRepo.CountByParentID(taskID, &subtaskCount); err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to count subtasks")
	}
	if subtaskCount >= 100 {
		return nil, apperror.ErrSubtaskLimitReached
	}

	project, err := s.projectRepo.GetByID(projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrProjectNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get project")
	}
	if project.IsArchived {
		return nil, apperror.ErrProjectArchived
	}

	if req.Priority != "" {
		switch models.TaskPriority(req.Priority) {
		case models.TaskPriorityLOW, models.TaskPriorityMED, models.TaskPriorityHIGH, models.TaskPriorityURGENT:
		default:
			return nil, apperror.ErrInvalidPriority
		}
	}

	if len(req.AssigneeIDs) > 0 {
		invalidIDs, err := s.projectMemberRepo.ValidateMembersExist(projectID, req.AssigneeIDs)
		if err != nil {
			return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to validate assignees")
		}
		if len(invalidIDs) > 0 {
			return nil, apperror.NewAppError(400, "INVALID_ASSIGNEE_IDS",
				fmt.Sprintf("Users are not project members: %v", invalidIDs))
		}
	}

	nextNum, err := s.taskRepo.GetNextTaskNumber(projectID)
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to generate task number")
	}

	priority := models.TaskPriority(req.Priority)
	if req.Priority == "" {
		priority = models.TaskPriorityMED
	}

	subtask := models.Task{
		ProjectID:  projectID,
		ColumnID:   parent.ColumnID,
		CreatorID:  &userID,
		ParentID:   &taskID,
		TaskNumber: nextNum,
		Title:      req.Title,
		Priority:   priority,
		DueDate:    req.DueDate,
		Position:   float64(nextNum * 1000),
	}
	if err := s.taskRepo.Create(&subtask); err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to create subtask")
	}

	if len(req.AssigneeIDs) > 0 {
		if err := s.taskAssigneeRepo.BulkTaskAssignee(subtask.ID, req.AssigneeIDs, userID); err != nil {
			return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to assign members")
		}
	}

	column, err := s.columnRepo.GetByID(parent.ColumnID)
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get column")
	}

	assignees, err := s.taskAssigneeRepo.ListByTaskID(subtask.ID)
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get assignees")
	}

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
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to list subtasks")
	}
	if len(resp.Data) > 0 {
		subtaskIDs := make([]string, len(resp.Data))
		for i := range resp.Data {
			subtaskIDs[i] = resp.Data[i].ID
		}
		assigneeMap, err := s.taskAssigneeRepo.ListByTaskIDs(subtaskIDs)
		if err != nil {
			return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to load assignees")
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

	tasks, _, pagination, err := s.taskAssigneeRepo.ListMyTasks(userID, workspaceID, filters, page, limit, sortBy, sortDir)
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get my tasks")
	}

	for i := range tasks {
		if tasks[i].DueDate != nil && tasks[i].DueDate.Before(time.Now()) {
			tasks[i].IsOverdue = true
		}
	}
	return &dto.MyTaskListResponse{Data: tasks, Pagination: *pagination}, nil
}

func (s *taskService) SearchTasks(workspaceID string, userID string, projectID string, search string, priority string, assigneeID string, labelID string, creatorID string, columnID string, dueDateFrom string, dueDateTo string, hasAssignee *bool, overdue *bool, includeSubtasks bool, sortBy string, sortDir string, page int, limit int) (*dto.TaskSearchResponse, error) {
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
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to search tasks")
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
	if overdue != nil {
		filtersApplied["overdue"] = *overdue
	}

	return &dto.TaskSearchResponse{
		Data:           tasks,
		Pagination:     *pagination,
		FiltersApplied: filtersApplied,
	}, nil
}
