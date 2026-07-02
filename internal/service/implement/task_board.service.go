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

type taskBoardService struct {
	taskRepo          repoInterface.TaskRepository
	columnRepo        repoInterface.ColumnRepository
	projectRepo       repoInterface.ProjectRepository
	projectMemberRepo repoInterface.ProjectMemberRepository
	taskAssigneeRepo  repoInterface.TaskAssigneeRepository
	taskLabelRepo     repoInterface.TaskLabelRepository
}

func NewTaskBoardService(
	taskRepo repoInterface.TaskRepository,
	columnRepo repoInterface.ColumnRepository,
	projectRepo repoInterface.ProjectRepository,
	projectMemberRepo repoInterface.ProjectMemberRepository,
	taskAssigneeRepo repoInterface.TaskAssigneeRepository,
	taskLabelRepo repoInterface.TaskLabelRepository,
) _interface.TaskBoardService {
	return &taskBoardService{
		taskRepo:          taskRepo,
		columnRepo:        columnRepo,
		projectRepo:       projectRepo,
		projectMemberRepo: projectMemberRepo,
		taskAssigneeRepo:  taskAssigneeRepo,
		taskLabelRepo:     taskLabelRepo,
	}
}

func (s *taskBoardService) getProjectOrFail(workspaceID, projectID string) (*models.Project, error) {
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

func (s *taskBoardService) getColumnOrFail(projectID, columnID string) (*models.Column, error) {
	col, err := s.columnRepo.GetByID(columnID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewAppError(http.StatusBadRequest, "INVALID_COLUMN", "Column does not exist in this project")
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get column")
	}
	if col.ProjectID != projectID {
		return nil, apperror.NewAppError(http.StatusBadRequest, "INVALID_COLUMN", "Column does not exist in this project")
	}
	return col, nil
}

func (s *taskBoardService) buildBoardFilters(priority, assigneeID, labelID []string, dueDateFrom, dueDateTo, creatorID string, hasAssignee, overdue *bool, search string) map[string]interface{} {
	filters := make(map[string]interface{})
	if len(priority) > 0 {
		filters["priority"] = priority
	}
	if len(assigneeID) > 0 {
		filters["assignee_id"] = assigneeID
	}
	if len(labelID) > 0 {
		filters["label_id"] = labelID
	}
	if dueDateFrom != "" {
		filters["due_date_from"] = dueDateFrom
	}
	if dueDateTo != "" {
		filters["due_date_to"] = dueDateTo
	}
	if creatorID != "" {
		filters["creator_id"] = creatorID
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
	return filters
}

func (s *taskBoardService) GetBoardData(workspaceID string, userID string, projectID string, priority []string, assigneeID []string, labelID []string, dueDateFrom string, dueDateTo string, creatorID string, hasAssignee *bool, overdue *bool, search string, tasksPerColumn int) (*dto.BoardResponse, error) {
	_, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}

	filters := s.buildBoardFilters(priority, assigneeID, labelID, dueDateFrom, dueDateTo, creatorID, hasAssignee, overdue, search)
	result, err := s.taskRepo.GetBoardByProjectID(projectID, filters, tasksPerColumn)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrProjectNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get board data")
	}
	return result, nil
}

func (s *taskBoardService) LoadMoreTasksInColumn(workspaceID string, userID string, projectID string, columnID string, cursor string, limit int, priority []string, assigneeID []string, labelID []string, dueDateFrom string, dueDateTo string, creatorID string, hasAssignee *bool, overdue *bool, search string) (*dto.LoadMoreTasksResponse, error) {
	_, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	_, err = s.getColumnOrFail(projectID, columnID)
	if err != nil {
		return nil, err
	}

	filters := s.buildBoardFilters(priority, assigneeID, labelID, dueDateFrom, dueDateTo, creatorID, hasAssignee, overdue, search)
	result, err := s.taskRepo.LoadMoreTasksInColumn(columnID, cursor, limit, filters)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load more tasks")
	}
	return result, nil
}

func (s *taskBoardService) MoveTask(workspaceID string, userID string, projectID string, taskID string, req *dto.MoveTaskRequest) (*dto.MoveTaskResponse, error) {
	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if project.IsArchived {
		return nil, apperror.ErrProjectArchived
	}

	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewAppError(http.StatusNotFound, "TASK_NOT_FOUND", "Task not found")
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get task")
	}
	if task.ProjectID != projectID {
		return nil, apperror.NewAppError(http.StatusNotFound, "TASK_NOT_FOUND", "Task not found")
	}
	if task.DeletedAt.Valid {
		return nil, apperror.NewAppError(http.StatusBadRequest, "CANNOT_MOVE_DELETED_TASK", "Task has been deleted")
	}

	col, err := s.getColumnOrFail(projectID, req.ColumnID)
	if err != nil {
		return nil, err
	}
	_ = col

	var newPos float64
	needRebalance := false

	existingColTasks, err := 	s.taskRepo.ListByColumnID(req.ColumnID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get column tasks")
	}

	if len(existingColTasks) == 0 {
		newPos = 1000
	} else if req.PreviousPosition == nil && req.NextPosition != nil {
		if *req.NextPosition <= 0 {
			return nil, apperror.NewAppError(http.StatusBadRequest, "INVALID_POSITION_CONTEXT", "Invalid position context")
		}
		newPos = *req.NextPosition / 2
	} else if req.PreviousPosition != nil && req.NextPosition == nil {
		newPos = *req.PreviousPosition + 1000
	} else if req.PreviousPosition != nil && req.NextPosition != nil {
		diff := *req.NextPosition - *req.PreviousPosition
		if diff < 2 {
			needRebalance = true
		}
		newPos = (*req.PreviousPosition + *req.NextPosition) / 2
	} else {
		lastPos := 0.0
		for _, t := range existingColTasks {
			if t.ID != taskID && t.Position > lastPos {
				lastPos = t.Position
			}
		}
		newPos = lastPos + 1000
	}

	if needRebalance {
		if err := s.taskRepo.Reorder(req.ColumnID, nil); err != nil {
			return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to rebalance tasks")
		}
		rebalancedTasks, err := 	s.taskRepo.ListByColumnID(req.ColumnID)
		if err != nil {
			return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get tasks after rebalance")
		}
		for i, t := range rebalancedTasks {
			if t.ID == taskID {
				newPos = float64((i + 1) * 1000)
				break
			}
		}
	}

	previousColumn := task.ColumnID
	task.ColumnID = req.ColumnID
	task.Position = newPos
	task.UpdatedAt = time.Now()

	if err := s.taskRepo.Update(task); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update task position")
	}

	ref := ""
	if project != nil {
		ref = project.Key + "-"
	}
	ref = ref + fmt.Sprintf("%d", task.TaskNumber)

	movedBetween := previousColumn != req.ColumnID

	return &dto.MoveTaskResponse{
		ID:                task.ID,
		TaskRef:           ref,
		ColumnID:          req.ColumnID,
		Position:          newPos,
		PreviousColumnID:  previousColumn,
		MovedBetweenColumns: movedBetween,
		Rebalanced:        needRebalance,
		UpdatedAt:         task.UpdatedAt.Format(time.RFC3339),
	}, nil
}
