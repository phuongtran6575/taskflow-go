package _interface

import (
	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
)

type TaskLabelRepository interface {
	ListTaskLabelsByTaskID(taskID string) ([]models.TaskLabel, error)
	ListByLabelID(labelID string) ([]models.TaskLabel, error)
	BulkTaskLabel(taskID string, labelIDs []string) error
	ListByTaskID(taskID string) ([]dto.LabelRef, error)
	ListByTaskIDs(taskIDs []string) (map[string][]dto.LabelRef, error)
}
