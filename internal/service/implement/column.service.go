package implement

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/activitylog"
	"TaskFlow-Go/internal/database"
	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/helper"
	"TaskFlow-Go/internal/models"
	"TaskFlow-Go/internal/validator"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
)

type columnService struct {
	tm              *database.TransactionManager
	columnRepo      repoInterface.ColumnRepository
	projectRepo     repoInterface.ProjectRepository
	activityLogRepo repoInterface.ActivityLogRepository
	userRepo        repoInterface.UserRepository
}

func NewColumnService(
	tm *database.TransactionManager,
	columnRepo repoInterface.ColumnRepository,
	projectRepo repoInterface.ProjectRepository,
	activityLogRepo repoInterface.ActivityLogRepository,
	userRepo repoInterface.UserRepository,
) _interface.ColumnService {
	return &columnService{
		tm:              tm,
		columnRepo:      columnRepo,
		projectRepo:     projectRepo,
		activityLogRepo: activityLogRepo,
		userRepo:        userRepo,
	}
}

func (s *columnService) getUserName(userID string) string {
	u, err := s.userRepo.GetByID(userID)
	if err != nil {
		return userID
	}
	return u.FullName
}

func (s *columnService) getProjectOrFail(workspaceID, projectID string) (*models.Project, error) {
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

func (s *columnService) getColumnOrFail(projectID, columnID string) (*models.Column, error) {
	col, err := s.columnRepo.GetByID(columnID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrColumnNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get column")
	}
	if col.ProjectID != projectID {
		return nil, apperror.ErrColumnNotFound
	}
	if col.DeletedAt.Valid {
		return nil, apperror.ErrColumnNotFound
	}
	return col, nil
}

func (s *columnService) logActivity(workspaceID, projectID, userID, entityID string, action models.ActivityAction, metadata map[string]interface{}, description string, entitySnapshot map[string]interface{}) {
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
		EntityType:     models.EntityTypeCOLUMN,
		EntityID:       entityID,
		Description:    descPtr,
		Metadata:       metaStr,
		EntitySnapshot: snapStr,
	})
}

func (s *columnService) logActivityInTx(tx *gorm.DB, workspaceID, projectID, userID, entityID string, action models.ActivityAction, metadata map[string]interface{}, description string, entitySnapshot map[string]interface{}) {
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
		EntityType:     models.EntityTypeCOLUMN,
		EntityID:       entityID,
		Description:    descPtr,
		Metadata:       metaStr,
		EntitySnapshot: snapStr,
	})
}

// validatePositionContext implements BR-COL-03
func (s *columnService) validatePositionContext(colRepo repoInterface.ColumnRepository, projectID string, prevPos, nextPos *float64) error {
	if prevPos == nil && nextPos == nil {
		return apperror.ErrInvalidPositionContext
	}

	if prevPos != nil {
		_, err := colRepo.GetByProjectIDAndPosition(projectID, *prevPos)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperror.ErrInvalidPositionContext
			}
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to validate position")
		}
	}

	if nextPos != nil {
		_, err := colRepo.GetByProjectIDAndPosition(projectID, *nextPos)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperror.ErrInvalidPositionContext
			}
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to validate position")
		}
	}

	if prevPos != nil && nextPos != nil {
		if *prevPos >= *nextPos {
			return apperror.ErrInvalidPositionContext
		}

		allCols, err := colRepo.ListByProjectID(projectID)
		if err != nil {
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to validate adjacency")
		}
		adjacent := false
		for i := 0; i < len(allCols)-1; i++ {
			if allCols[i].Position == *prevPos && allCols[i+1].Position == *nextPos {
				adjacent = true
				break
			}
		}
		if !adjacent {
			return apperror.ErrInvalidPositionContext
		}
	}

	return nil
}

// --- Interface implementation ---

func (s *columnService) ListColumns(workspaceID string, userID string, projectID string) (*dto.ColumnListResponse, error) {
	_, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	columns, err := s.columnRepo.ListByProjectIDWithCount(projectID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list columns")
	}
	return &dto.ColumnListResponse{Data: columns, Total: len(columns)}, nil
}

func (s *columnService) CreateColumn(workspaceID string, userID string, projectID string, req *dto.CreateColumnRequest) (*dto.ColumnCreateResponse, error) {
	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if project.IsArchived {
		return nil, apperror.ErrProjectArchived
	}
	if err := validator.ValidateColumnTitle(req.Title); err != nil {
		return nil, err
	}

	var result *dto.ColumnCreateResponse
	err = s.tm.Execute(func(tx *gorm.DB) error {
		colRepo := s.columnRepo.WithTx(tx)

		existingCols, err := colRepo.ListByProjectID(projectID)
		if err != nil {
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get columns")
		}
		if len(existingCols) >= 20 {
			return apperror.ErrColumnLimitReached
		}

		maxPos := 0.0
		for _, c := range existingCols {
			if c.Position > maxPos {
				maxPos = c.Position
			}
		}
		newPos := maxPos + 1000

		isDone := false
		if req.IsDone != nil {
			isDone = *req.IsDone
		}

		col := &models.Column{
			ProjectID: projectID,
			Title:     strings.TrimSpace(req.Title),
			Position:  newPos,
			IsDone:    isDone,
		}
		if err := colRepo.Create(col); err != nil {
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create column")
		}

		result = &dto.ColumnCreateResponse{
			ID:        col.ID,
			Title:     col.Title,
			Position:  col.Position,
			IsDone:    col.IsDone,
			TaskCount: 0,
			CreatedAt: col.CreatedAt,
			UpdatedAt: col.UpdatedAt,
		}

		actorName := s.getUserName(userID)
		meta := activitylog.ColumnCreated(col.Title, int(col.Position))
		desc := activitylog.GenerateDescription(actorName, meta)
		snap := activitylog.BuildColumnSnapshot(col.Title, project.Key)
		s.logActivityInTx(tx, workspaceID, projectID, userID, col.ID, models.ActivityActionCREATE, meta, desc, snap)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *columnService) UpdateColumnTitle(workspaceID string, userID string, projectID string, columnID string, req *dto.UpdateColumnTitleRequest) (*dto.UpdateColumnTitleResponse, error) {
	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if project.IsArchived {
		return nil, apperror.ErrProjectArchived
	}
	if err := validator.ValidateColumnTitle(req.Title); err != nil {
		return nil, err
	}

	var result *dto.UpdateColumnTitleResponse
	err = s.tm.Execute(func(tx *gorm.DB) error {
		colRepo := s.columnRepo.WithTx(tx)

		col, err := colRepo.GetByID(columnID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperror.ErrColumnNotFound
			}
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get column")
		}
		if col.ProjectID != projectID || col.DeletedAt.Valid {
			return apperror.ErrColumnNotFound
		}

		oldTitle := col.Title
		col.Title = strings.TrimSpace(req.Title)
		col.UpdatedAt = time.Now()
		if err := colRepo.Update(col); err != nil {
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update column")
		}

		result = &dto.UpdateColumnTitleResponse{
			ID:        col.ID,
			Title:     col.Title,
			Position:  col.Position,
			UpdatedAt: col.UpdatedAt,
		}

		actorName := s.getUserName(userID)
		meta := activitylog.ColumnUpdated(oldTitle, col.Title)
		desc := activitylog.GenerateDescription(actorName, meta)
		snap := activitylog.BuildColumnSnapshot(col.Title, project.Key)
		s.logActivityInTx(tx, workspaceID, projectID, userID, col.ID, models.ActivityActionUPDATE, meta, desc, snap)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *columnService) UpdateColumnPosition(workspaceID string, userID string, projectID string, columnID string, req *dto.UpdateColumnPositionRequest) (*dto.UpdateColumnPositionResponse, error) {
	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if project.IsArchived {
		return nil, apperror.ErrProjectArchived
	}

	var result *dto.UpdateColumnPositionResponse
	err = s.tm.Execute(func(tx *gorm.DB) error {
		colRepo := s.columnRepo.WithTx(tx)

		col, err := colRepo.GetByID(columnID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperror.ErrColumnNotFound
			}
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get column")
		}
		if col.ProjectID != projectID || col.DeletedAt.Valid {
			return apperror.ErrColumnNotFound
		}

		if err := s.validatePositionContext(colRepo, projectID, req.PreviousPosition, req.NextPosition); err != nil {
			return err
		}

		var newPos float64
		if req.PreviousPosition == nil {
			newPos = *req.NextPosition / 2
		} else if req.NextPosition == nil {
			newPos = *req.PreviousPosition + 1000
		} else {
			newPos = (*req.PreviousPosition + *req.NextPosition) / 2
		}

		if req.NextPosition != nil && req.PreviousPosition != nil {
			diff := *req.NextPosition - *req.PreviousPosition
			if diff < 0.001 {
				if err := colRepo.Reorder(projectID, nil); err != nil {
					return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to rebalance columns")
				}

				rebalancedCols, err := colRepo.ListByProjectID(projectID)
				if err != nil {
					return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get columns after rebalance")
				}

				for i, c := range rebalancedCols {
					if c.ID == columnID {
						newPos = float64((i + 1) * 1000)
						break
					}
				}

				allColsInfo := helper.ColumnsToInfo(rebalancedCols)

				col.Position = newPos
				col.UpdatedAt = time.Now()
				if err := colRepo.Update(col); err != nil {
					return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update position")
				}

				result = &dto.UpdateColumnPositionResponse{
					ID:         col.ID,
					Title:      col.Title,
					Position:   col.Position,
					UpdatedAt:  col.UpdatedAt,
					Rebalanced: true,
					AllColumns: &allColsInfo,
				}

			return nil
			}
		}

		col.Position = newPos
		col.UpdatedAt = time.Now()
		if err := colRepo.Update(col); err != nil {
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update position")
		}

		result = &dto.UpdateColumnPositionResponse{
			ID:         col.ID,
			Title:      col.Title,
			Position:   col.Position,
			UpdatedAt:  col.UpdatedAt,
			Rebalanced: false,
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *columnService) DeleteColumn(workspaceID string, userID string, projectID string, columnID string, req *dto.DeleteColumnRequest) (*dto.ColumnDeleteResponse, error) {
	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if project.IsArchived {
		return nil, apperror.ErrProjectArchived
	}

	var result *dto.ColumnDeleteResponse
	err = s.tm.Execute(func(tx *gorm.DB) error {
		colRepo := s.columnRepo.WithTx(tx)

		col, err := colRepo.GetByID(columnID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperror.ErrColumnNotFound
			}
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get column")
		}
		if col.ProjectID != projectID || col.DeletedAt.Valid {
			return apperror.ErrColumnNotFound
		}

		allCols, err := colRepo.ListByProjectID(projectID)
		if err != nil {
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get columns")
		}
		if len(allCols) <= 1 {
			return apperror.ErrLastColumn
		}

		taskCount, err := colRepo.CountTasksByColumnID(columnID)
		if err != nil {
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to count tasks")
		}

		if taskCount > 0 {
			if req.Strategy == nil || *req.Strategy == "" {
				return apperror.ErrStrategyRequired
			}
			switch *req.Strategy {
			case "move":
				if req.TargetColumnID == nil || *req.TargetColumnID == "" {
					return apperror.ErrTargetColumnRequired
				}
				if *req.TargetColumnID == columnID {
					return apperror.ErrCannotMoveToSelf
				}

				targetCol, err := colRepo.GetByID(*req.TargetColumnID)
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						return apperror.ErrInvalidTargetColumn
					}
					return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get target column")
				}
				if targetCol.ProjectID != projectID || targetCol.DeletedAt.Valid {
					return apperror.ErrInvalidTargetColumn
				}

				minPos, err := colRepo.GetMinTaskPosition(*req.TargetColumnID)
				if err != nil {
					return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get target min position")
				}

				var tasksMoved int64
				if minPos > 0 {
					newBase := minPos / 2
					tasksMoved, err = colRepo.MoveTasksWithReposition(columnID, *req.TargetColumnID, newBase)
				} else {
					tasksMoved, err = colRepo.MoveTasksToColumn(columnID, *req.TargetColumnID)
				}
				if err != nil {
					return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to move tasks")
				}
				movedInt := int(tasksMoved)

				if err := colRepo.Delete(columnID); err != nil {
					return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete column")
				}

				result = &dto.ColumnDeleteResponse{
					Message:         "Column '" + col.Title + "' has been deleted. " + strconv.Itoa(movedInt) + " tasks moved to '" + targetCol.Title + "'.",
					DeletedColumnID: columnID,
					TasksMoved:      &movedInt,
					TargetColumnID:  req.TargetColumnID,
				}

				actorName := s.getUserName(userID)
				meta := activitylog.ColumnDeletedMoveTasks(col.Title, *req.TargetColumnID, targetCol.Title, movedInt)
				desc := activitylog.GenerateDescription(actorName, meta)
				snap := activitylog.BuildColumnSnapshot(col.Title, project.Key)
				s.logActivityInTx(tx, workspaceID, projectID, userID, col.ID, models.ActivityActionDELETE, meta, desc, snap)
				return nil

			case "delete":
				deleted, err := colRepo.SoftDeleteTasksByColumnID(columnID)
				if err != nil {
					return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete tasks")
				}
				if err := colRepo.SoftDeleteAttachmentsByColumnID(columnID); err != nil {
					return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete attachments")
				}
				if err := colRepo.SoftDeleteCommentsByColumnID(columnID); err != nil {
					return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete comments")
				}
				if err := colRepo.Delete(columnID); err != nil {
					return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete column")
				}
				tasksDeleted := int(deleted)

				result = &dto.ColumnDeleteResponse{
					Message:         "Column '" + col.Title + "' and all " + strconv.Itoa(tasksDeleted) + " tasks have been deleted.",
					DeletedColumnID: columnID,
					TasksDeleted:    &tasksDeleted,
				}

				actorName := s.getUserName(userID)
				meta := activitylog.ColumnDeletedDeleteTasks(col.Title, tasksDeleted)
				desc := activitylog.GenerateDescription(actorName, meta)
				snap := activitylog.BuildColumnSnapshot(col.Title, project.Key)
				s.logActivityInTx(tx, workspaceID, projectID, userID, col.ID, models.ActivityActionDELETE, meta, desc, snap)
				return nil

			default:
				return apperror.NewAppError(http.StatusBadRequest, "VALIDATION_ERROR", "Invalid strategy: must be 'move' or 'delete'")
			}
		}

		if err := colRepo.Delete(columnID); err != nil {
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete column")
		}

		zero := 0
		result = &dto.ColumnDeleteResponse{
			Message:         "Column '" + col.Title + "' has been deleted.",
			DeletedColumnID: columnID,
			TasksMoved:      &zero,
		}

		actorName := s.getUserName(userID)
		meta := activitylog.ColumnDeleted(col.Title)
		desc := activitylog.GenerateDescription(actorName, meta)
		snap := activitylog.BuildColumnSnapshot(col.Title, project.Key)
		s.logActivityInTx(tx, workspaceID, projectID, userID, col.ID, models.ActivityActionDELETE, meta, desc, snap)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}


