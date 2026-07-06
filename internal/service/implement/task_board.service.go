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

type taskBoardService struct {
	tm                *database.TransactionManager
	taskRepo          repoInterface.TaskRepository
	columnRepo        repoInterface.ColumnRepository
	projectRepo       repoInterface.ProjectRepository
	projectMemberRepo repoInterface.ProjectMemberRepository
	taskAssigneeRepo  repoInterface.TaskAssigneeRepository
	taskLabelRepo     repoInterface.TaskLabelRepository
	activityLogRepo   repoInterface.ActivityLogRepository
	notifRepo         repoInterface.NotificationRepository
	userRepo          repoInterface.UserRepository
	dispatcher        *notif.Dispatcher
}

func NewTaskBoardService(
	tm *database.TransactionManager,
	taskRepo repoInterface.TaskRepository,
	columnRepo repoInterface.ColumnRepository,
	projectRepo repoInterface.ProjectRepository,
	projectMemberRepo repoInterface.ProjectMemberRepository,
	taskAssigneeRepo repoInterface.TaskAssigneeRepository,
	taskLabelRepo repoInterface.TaskLabelRepository,
	activityLogRepo repoInterface.ActivityLogRepository,
	notifRepo repoInterface.NotificationRepository,
	userRepo repoInterface.UserRepository,
	dispatcher *notif.Dispatcher,
) _interface.TaskBoardService {
	return &taskBoardService{
		tm:                tm,
		taskRepo:          taskRepo,
		columnRepo:        columnRepo,
		projectRepo:       projectRepo,
		projectMemberRepo: projectMemberRepo,
		taskAssigneeRepo:  taskAssigneeRepo,
		taskLabelRepo:     taskLabelRepo,
		activityLogRepo:   activityLogRepo,
		notifRepo:         notifRepo,
		userRepo:          userRepo,
		dispatcher:        dispatcher,
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

func (s *taskBoardService) buildBoardFilters(priority, assigneeID, labelID []string, dueDateFrom, dueDateTo, creatorID string, hasAssignee, hasLabel, overdue *bool, search string) map[string]interface{} {
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
	if hasLabel != nil {
		filters["has_label"] = *hasLabel
	}
	if overdue != nil {
		filters["overdue"] = *overdue
	}
	if search != "" {
		filters["search"] = search
	}
	return filters
}

func (s *taskBoardService) GetBoardData(workspaceID string, userID string, projectID string, priority []string, assigneeID []string, labelID []string, dueDateFrom string, dueDateTo string, creatorID string, hasAssignee *bool, hasLabel *bool, overdue *bool, search string, tasksPerColumn int) (*dto.BoardResponse, error) {
	_, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}

	filters := s.buildBoardFilters(priority, assigneeID, labelID, dueDateFrom, dueDateTo, creatorID, hasAssignee, hasLabel, overdue, search)
	result, err := s.taskRepo.GetBoardByProjectID(projectID, filters, tasksPerColumn)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrProjectNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get board data")
	}
	return result, nil
}

func (s *taskBoardService) LoadMoreTasksInColumn(workspaceID string, userID string, projectID string, columnID string, cursor string, limit int, priority []string, assigneeID []string, labelID []string, dueDateFrom string, dueDateTo string, creatorID string, hasAssignee *bool, hasLabel *bool, overdue *bool, search string) (*dto.LoadMoreTasksResponse, error) {
	_, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	_, err = s.getColumnOrFail(projectID, columnID)
	if err != nil {
		return nil, err
	}

	filters := s.buildBoardFilters(priority, assigneeID, labelID, dueDateFrom, dueDateTo, creatorID, hasAssignee, hasLabel, overdue, search)
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

	targetCol, err := s.getColumnOrFail(projectID, req.ColumnID)
	if err != nil {
		return nil, err
	}
	_ = targetCol

	isCrossColumn := req.ColumnID != task.ColumnID

	if err := s.validatePositionContext(req, isCrossColumn, task.ColumnID); err != nil {
		return nil, err
	}

	newPos, needRebalance, err := s.calculateNewPosition(req, isCrossColumn, task.ColumnID, task.ID)
	if err != nil {
		return nil, err
	}

	var oldColumn *models.Column
	if isCrossColumn {
		oldColumn, err = s.columnRepo.GetByID(task.ColumnID)
		if err != nil {
			return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get source column")
		}
	}

	var updatedTaskID string
	var responsePos float64
	var allPositions *[]dto.TaskPositionInfo

	now := time.Now()

	err = s.tm.Execute(func(tx *gorm.DB) error {
		txTaskRepo := s.taskRepo.WithTx(tx)

		lastKnown := now
		if req.LastKnownUpdatedAt != nil {
			lastKnown = *req.LastKnownUpdatedAt
		}

		affected, err := txTaskRepo.UpdatePositionAtomic(taskID, req.ColumnID, newPos, lastKnown)
		if err != nil {
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update task position")
		}
		if affected == 0 {
			return apperror.ErrPositionConflict
		}

		responsePos = newPos
		updatedTaskID = taskID

		if needRebalance {
			positions, err := txTaskRepo.RebalanceColumn(req.ColumnID)
			if err != nil {
				return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to rebalance column")
			}
			allPositions = &positions
			for _, p := range positions {
				if p.ID == taskID {
					responsePos = p.Position
					break
				}
			}
		}

		if isCrossColumn {
			actorName := s.getUserName(userID)
			taskRef := fmt.Sprintf("%s-%d", project.Key, task.TaskNumber)
			meta := activitylog.ColumnChanged(oldColumn.ID, oldColumn.Title, req.ColumnID, targetCol.Title)
			desc := activitylog.GenerateDescription(actorName, meta)
			snap := activitylog.BuildTaskSnapshot(taskRef, task.Title, project.Key)

			metaBytes, _ := json.Marshal(meta)
			metaStr := string(metaBytes)
			snapBytes, _ := json.Marshal(snap)
			snapStr := string(snapBytes)
			var descPtr *string
			if desc != "" {
				descPtr = &desc
			}
			wsID := workspaceID
			uID := userID
			if err := tx.Create(&models.ActivityLog{
				WorkspaceID:    &wsID,
				ProjectID:      &projectID,
				UserID:         &uID,
				Action:         models.ActivityActionUPDATE,
				EntityType:     models.EntityTypeTASK,
				EntityID:       taskID,
				Description:    descPtr,
				Metadata:       &metaStr,
				EntitySnapshot: &snapStr,
				CreatedAt:      time.Now(),
			}).Error; err != nil {
				return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to log activity")
			}
		}

		return nil
	})
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			return nil, appErr
		}
		return nil, err
	}

	if isCrossColumn && task.ParentID == nil {
		s.sendStatusChangeNotification(taskID, userID, project, task, oldColumn, targetCol)
	}

	ref := project.Key + "-" + fmt.Sprintf("%d", task.TaskNumber)

	prevColID := req.ColumnID
	if isCrossColumn {
		prevColID = oldColumn.ID
	}

	return &dto.MoveTaskResponse{
		ID:                  updatedTaskID,
		TaskRef:             ref,
		ColumnID:            req.ColumnID,
		Position:            responsePos,
		PreviousColumnID:    prevColID,
		MovedBetweenColumns: isCrossColumn,
		Rebalanced:          needRebalance,
		AllPositions:        allPositions,
		UpdatedAt:           now.Format(time.RFC3339),
	}, nil
}

func (s *taskBoardService) validatePositionContext(req *dto.MoveTaskRequest, isCrossColumn bool, currentColumnID string) error {
	if req.PreviousPosition == nil && req.NextPosition == nil {
		return nil
	}

	posColumnID := req.ColumnID
	if !isCrossColumn {
		posColumnID = currentColumnID
	}

	if req.PreviousPosition != nil {
		exists, err := s.taskRepo.ExistsInColumn(posColumnID, *req.PreviousPosition)
		if err != nil {
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to validate position")
		}
		if !exists {
			return apperror.NewAppError(http.StatusBadRequest, "INVALID_POSITION_CONTEXT", "Previous position does not exist in target column")
		}
	}

	if req.NextPosition != nil {
		exists, err := s.taskRepo.ExistsInColumn(posColumnID, *req.NextPosition)
		if err != nil {
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to validate position")
		}
		if !exists {
			return apperror.NewAppError(http.StatusBadRequest, "INVALID_POSITION_CONTEXT", "Next position does not exist in target column")
		}
	}

	if req.PreviousPosition != nil && req.NextPosition != nil {
		if *req.PreviousPosition >= *req.NextPosition {
			return apperror.NewAppError(http.StatusBadRequest, "INVALID_POSITION_CONTEXT", "Previous position must be less than next position")
		}
		count, err := s.taskRepo.CountBetweenPositions(posColumnID, *req.PreviousPosition, *req.NextPosition)
		if err != nil {
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to validate position adjacency")
		}
		if count > 0 {
			return apperror.NewAppError(http.StatusBadRequest, "INVALID_POSITION_CONTEXT", "Tasks exist between previous and next position")
		}
	}

	return nil
}

func (s *taskBoardService) calculateNewPosition(req *dto.MoveTaskRequest, isCrossColumn bool, currentColumnID string, taskID string) (float64, bool, error) {
	posColumnID := req.ColumnID
	if !isCrossColumn {
		posColumnID = currentColumnID
	}

	if req.PreviousPosition == nil && req.NextPosition == nil {
		tasks, err := s.taskRepo.ListByColumnID(posColumnID)
		if err != nil {
			return 0, false, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list column tasks")
		}

		if len(tasks) == 0 || (len(tasks) == 1 && tasks[0].ID == taskID) {
			return 1000, false, nil
		}
		lastPos := 0.0
		for _, t := range tasks {
			if t.ID != taskID && t.Position > lastPos {
				lastPos = t.Position
			}
		}
		return lastPos + 1000, false, nil
	}

	if req.PreviousPosition == nil && req.NextPosition != nil {
		return *req.NextPosition / 2, false, nil
	}

	if req.PreviousPosition != nil && req.NextPosition == nil {
		return *req.PreviousPosition + 1000, false, nil
	}

	gap := *req.NextPosition - *req.PreviousPosition
	if gap < 0.001 {
		return (*req.PreviousPosition + *req.NextPosition) / 2, true, nil
	}
	return (*req.PreviousPosition + *req.NextPosition) / 2, false, nil
}

func (s *taskBoardService) sendStatusChangeNotification(taskID, actorID string, project *models.Project, task *models.Task, oldCol, newCol *models.Column) {
	assignees, err := s.taskAssigneeRepo.ListByTaskID(taskID)
	if err != nil || len(assignees) == 0 {
		return
	}

	actor, err := s.userRepo.GetByID(actorID)
	if err != nil {
		return
	}

	recipientIDs := make([]string, 0, len(assignees))
	for _, a := range assignees {
		recipientIDs = append(recipientIDs, a.UserID)
	}

	taskRef := fmt.Sprintf("%s-%d", project.Key, task.TaskNumber)
	s.dispatcher.DispatchSTATUSCHANGED(&notif.STATUSCHANGEDInput{
		ActorID:      actorID,
		ActorName:    actor.FullName,
		RecipientIDs: recipientIDs,
		TaskRef:      taskRef,
		TaskTitle:    task.Title,
		OldColumn:    oldCol.Title,
		NewColumn:    newCol.Title,
		WorkspaceID:  project.WorkspaceID,
		ProjectID:    project.ID,
		TaskID:       taskID,
	})
}

func (s *taskBoardService) getUserName(userID string) string {
	u, err := s.userRepo.GetByID(userID)
	if err != nil {
		return userID
	}
	return u.FullName
}
