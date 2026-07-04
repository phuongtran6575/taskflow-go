package _interface

import (
	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
)

type ActivityLogRepository interface {
	Create(log *models.ActivityLog) error
	ListByWorkspace(workspaceID string, filters map[string][]string, limit int, cursor string) (*dto.ActivityLogListResponse, error)
	ListByProject(projectID string, filters map[string][]string, limit int, cursor string) (*dto.ActivityLogListResponse, error)
	ListByTask(taskID string, limit int, cursor string, direction string) (*dto.ActivityLogListResponse, error)
}
