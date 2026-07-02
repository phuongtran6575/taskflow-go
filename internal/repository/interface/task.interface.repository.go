package _interface

import (
	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
)

type TaskRepository interface {
	Create(task *models.Task) error
	GetByID(id string) (*models.Task, error)
	ListByColumnID(columnID string) ([]models.Task, error)
	Update(task *models.Task) error
	Delete(id string) error
	Reorder(columnID string, taskIDs []string) error

	CountByParentID(parentID string, count *int64) error
	GetNextTaskNumber(projectID string) (int, error)
	GetCreateTaskResponse(taskID string) (*dto.TaskCreateResponse, error)

	ListWithFilters(projectID string, filters map[string]interface{}, page int, limit int) ([]dto.TaskSummary, *dto.Pagination, error)
	GetByIDWithDetail(taskID string) (*dto.TaskDetailResponse, error)
	ListSubtasks(taskID string) (*dto.SubtaskListResponse, error)
	SearchWithFilters(projectID string, filters map[string]interface{}, page int, limit int, sortBy string, sortDir string) ([]dto.TaskSummary, *dto.Pagination, error)

	GetBoardByProjectID(projectID string, filters map[string]interface{}, tasksPerColumn int) (*dto.BoardResponse, error)
	LoadMoreTasksInColumn(columnID string, cursor string, limit int, filters map[string]interface{}) (*dto.LoadMoreTasksResponse, error)
}
