package _interface

import (
	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
)

type LabelRepository interface {
	Create(label *models.Label) error
	GetByID(id string) (*models.Label, error)
	ListByProjectID(projectID string) ([]models.Label, error)
	Update(label *models.Label) error
	Delete(id string) error

	AddToTask(taskID, labelID string) error
	RemoveFromTask(taskID, labelID string) error

	ListByProjectIDWithCount(projectID string, search string, withTaskCount bool) ([]dto.LabelInfo, error)
	ListByTaskIDWithRef(taskID string) (*dto.TaskLabelListResponse, error)
	ExistsByNameInProject(projectID string, name string, excludeID string) (bool, error)
	CountByProjectID(projectID string) (int64, error)
}
