package _interface

import (
	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
)

type TaskAssigneeRepository interface {
	WithTx(tx *gorm.DB) TaskAssigneeRepository
	Create(assignee *models.TaskAssignee) error
	GetByID(taskID, userID string) (*models.TaskAssignee, error)
	ListTaskAssigneesByTaskID(taskID string) ([]models.TaskAssignee, error)
	Delete(taskID, userID string) error

	ListByTaskIDWithDetail(taskID string) ([]dto.AssigneeDetail, error)
	ListByTaskID(taskID string) ([]dto.AssigneeInfo, error)
	ListByTaskIDs(taskIDs []string) (map[string][]dto.AssigneeInfo, error)
	BulkTaskAssignee(taskID string, assigneeIDs []string, creatorID string) error
	ListAvailableForTask(taskID string, projectID string, search string, page int, limit int) ([]dto.AvailableAssigneeInfo, *dto.Pagination, error)
	ListMyTasks(userID string, workspaceID string, filters map[string]interface{}, page int, limit int, sortBy string, sortDir string) ([]dto.MyTaskInfo, *dto.MyTaskSummary, *dto.Pagination, error)
}
