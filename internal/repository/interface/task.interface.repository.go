package _interface

import (
	"time"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"

	"gorm.io/gorm"
)

type TaskRepository interface {
	WithTx(tx *gorm.DB) TaskRepository

	Create(task *models.Task) error
	GetByID(id string) (*models.Task, error)
	ListByColumnID(columnID string) ([]models.Task, error)
	Update(task *models.Task) error
	Delete(id string) error
	Reorder(columnID string, taskIDs []string) error

	CountByParentID(parentID string, count *int64) error
	GetNextTaskNumber(projectID string) (int, error)
	GetCreateTaskResponse(taskID string) (*dto.TaskCreateResponse, error)
	GetMaxPositionInColumn(projectID, columnID string) (float64, error)
	GetMaxPositionInParent(parentID string) (float64, error)

	ListWithFilters(projectID string, filters map[string]interface{}, page int, limit int) ([]dto.TaskSummary, *dto.Pagination, error)
	GetByIDWithDetail(taskID string) (*dto.TaskDetailResponse, error)
	ListSubtasks(taskID string) (*dto.SubtaskListResponse, error)
	SearchWithFilters(projectID string, filters map[string]interface{}, page int, limit int, sortBy string, sortDir string) ([]dto.TaskSummary, *dto.Pagination, error)

	GetBoardByProjectID(projectID string, filters map[string]interface{}, tasksPerColumn int) (*dto.BoardResponse, error)
	LoadMoreTasksInColumn(columnID string, cursor string, limit int, filters map[string]interface{}) (*dto.LoadMoreTasksResponse, error)

	// BR-BOARD-02: validate position context
	ExistsInColumn(columnID string, position float64) (bool, error)

	// BR-BOARD-02: count tasks between two positions in same column
	CountBetweenPositions(columnID string, prevPos, nextPos float64) (int64, error)

	// BR-BOARD-03: atomic conditional update for optimistic concurrency control
	// Returns rows affected (0 = conflict)
	UpdatePositionAtomic(taskID string, columnID string, position float64, lastKnownUpdatedAt time.Time) (int64, error)

	// BR-BOARD-02: rebalance all tasks in a column, returns updated positions
	RebalanceColumn(columnID string) ([]dto.TaskPositionInfo, error)

	CascadeDelete(taskID string) error
	ListIDsByParentID(parentID string) ([]string, error)
	ListOverdueIDs() ([]string, error)
}
