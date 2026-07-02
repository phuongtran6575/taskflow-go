package implement

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
)

type columnService struct {
	columnRepo  repoInterface.ColumnRepository
	projectRepo repoInterface.ProjectRepository
}

func NewColumnService(
	columnRepo repoInterface.ColumnRepository,
	projectRepo repoInterface.ProjectRepository,
) _interface.ColumnService {
	return &columnService{
		columnRepo:  columnRepo,
		projectRepo: projectRepo,
	}
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
	if len(req.Title) < 1 || len(req.Title) > 50 {
		return nil, apperror.NewAppError(http.StatusBadRequest, "VALIDATION_ERROR", "Column title must be between 1 and 50 characters")
	}

	existingCols, err := 	s.columnRepo.ListByProjectID(projectID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get columns")
	}
	if len(existingCols) >= 20 {
		return nil, apperror.ErrColumnLimitReached
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
		Title:     req.Title,
		Position:  newPos,
		IsDone:    isDone,
	}
	if err := s.columnRepo.Create(col); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create column")
	}

	return &dto.ColumnCreateResponse{
		ID:        col.ID,
		Title:     col.Title,
		Position:  col.Position,
		IsDone:    col.IsDone,
		TaskCount: 0,
		CreatedAt: col.CreatedAt,
		UpdatedAt: col.UpdatedAt,
	}, nil
}

func (s *columnService) UpdateColumnTitle(workspaceID string, userID string, projectID string, columnID string, req *dto.UpdateColumnTitleRequest) (*dto.UpdateColumnTitleResponse, error) {
	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if project.IsArchived {
		return nil, apperror.ErrProjectArchived
	}
	col, err := s.getColumnOrFail(projectID, columnID)
	if err != nil {
		return nil, err
	}
	if len(req.Title) < 1 || len(req.Title) > 50 {
		return nil, apperror.NewAppError(http.StatusBadRequest, "VALIDATION_ERROR", "Column title must be between 1 and 50 characters")
	}
	col.Title = req.Title
	col.UpdatedAt = time.Now()
	if err := s.columnRepo.Update(col); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update column")
	}
	return &dto.UpdateColumnTitleResponse{
		ID:        col.ID,
		Title:     col.Title,
		Position:  col.Position,
		UpdatedAt: col.UpdatedAt,
	}, nil
}

func (s *columnService) UpdateColumnPosition(workspaceID string, userID string, projectID string, columnID string, req *dto.UpdateColumnPositionRequest) (*dto.UpdateColumnPositionResponse, error) {
	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if project.IsArchived {
		return nil, apperror.ErrProjectArchived
	}
	col, err := s.getColumnOrFail(projectID, columnID)
	if err != nil {
		return nil, err
	}

	if req.PreviousPosition == nil && req.NextPosition == nil {
		return nil, apperror.ErrInvalidPositionContext
	}

	var newPos float64
	if req.PreviousPosition == nil {
		newPos = *req.NextPosition / 2
	} else if req.NextPosition == nil {
		newPos = *req.PreviousPosition + 1000
	} else {
		newPos = (*req.PreviousPosition + *req.NextPosition) / 2
	}

	rebalanced := false
	if req.NextPosition != nil && req.PreviousPosition != nil {
		diff := *req.NextPosition - *req.PreviousPosition
		if diff < 2 {
			rebalanced = true
			if err := s.columnRepo.Reorder(projectID, nil); err != nil {
				return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to rebalance columns")
			}
			allCols, err := 	s.columnRepo.ListByProjectID(projectID)
			if err != nil {
				return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get columns")
			}
			for i, c := range allCols {
				if c.ID == columnID {
					newPos = float64((i + 1) * 1000)
					break
				}
			}
		}
	}

	col.Position = newPos
	col.UpdatedAt = time.Now()
	if err := s.columnRepo.Update(col); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update position")
	}

	return &dto.UpdateColumnPositionResponse{
		ID:         col.ID,
		Title:      col.Title,
		Position:   col.Position,
		UpdatedAt:  col.UpdatedAt,
		Rebalanced: rebalanced,
	}, nil
}

func (s *columnService) DeleteColumn(workspaceID string, userID string, projectID string, columnID string, req *dto.DeleteColumnRequest) (*dto.ColumnDeleteResponse, error) {
	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if project.IsArchived {
		return nil, apperror.ErrProjectArchived
	}
	col, err := s.getColumnOrFail(projectID, columnID)
	if err != nil {
		return nil, err
	}

	allCols, err := 	s.columnRepo.ListByProjectID(projectID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get columns")
	}
	if len(allCols) <= 1 {
		return nil, apperror.ErrLastColumn
	}

	taskCount, err := s.columnRepo.CountTasksByColumnID(columnID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to count tasks")
	}

	if taskCount > 0 {
		if req.Strategy == nil || *req.Strategy == "" {
			return nil, apperror.ErrStrategyRequired
		}
		switch *req.Strategy {
		case "move":
			if req.TargetColumnID == nil || *req.TargetColumnID == "" {
				return nil, apperror.ErrTargetColumnRequired
			}
			if *req.TargetColumnID == columnID {
				return nil, apperror.ErrCannotMoveToSelf
			}
			targetCol, err := s.getColumnOrFail(projectID, *req.TargetColumnID)
			if err != nil {
				return nil, apperror.ErrInvalidTargetColumn
			}
			_ = targetCol

			moved, err := s.columnRepo.MoveTasksToColumn(columnID, *req.TargetColumnID)
			if err != nil {
				return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to move tasks")
			}
			movedInt := int(moved)

			if err := s.columnRepo.Delete(columnID); err != nil {
				return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete column")
			}

			tasksMoved := movedInt
			return &dto.ColumnDeleteResponse{
				Message:         "Column '" + col.Title + "' has been deleted. " + strconv.Itoa(tasksMoved) + " tasks moved to '" + targetCol.Title + "'.",
				DeletedColumnID: columnID,
				TasksMoved:      &tasksMoved,
				TargetColumnID:  req.TargetColumnID,
			}, nil

		case "delete":
			if err := s.columnRepo.Delete(columnID); err != nil {
				return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete column")
			}
			tasksDeleted := int(taskCount)
			return &dto.ColumnDeleteResponse{
				Message:         "Column '" + col.Title + "' and all " + strconv.Itoa(tasksDeleted) + " tasks have been deleted.",
				DeletedColumnID: columnID,
				TasksDeleted:    &tasksDeleted,
			}, nil

		default:
			return nil, apperror.NewAppError(http.StatusBadRequest, "VALIDATION_ERROR", "Invalid strategy: must be 'move' or 'delete'")
		}
	}

	if err := s.columnRepo.Delete(columnID); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete column")
	}

	zero := 0
	return &dto.ColumnDeleteResponse{
		Message:         "Column '" + col.Title + "' has been deleted.",
		DeletedColumnID: columnID,
		TasksMoved:      &zero,
	}, nil
}


