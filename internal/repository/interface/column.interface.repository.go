package _interface

import (
	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"

	"gorm.io/gorm"
)

type ColumnRepository interface {
	WithTx(tx *gorm.DB) ColumnRepository
	Create(column *models.Column) error
	CreateListColumn(columns []models.Column) error
	GetByID(id string) (*models.Column, error)
	ListByProjectID(projectID string) ([]models.Column, error)
	Update(column *models.Column) error
	Delete(id string) error
	Reorder(projectID string, columnIDs []string) error
	MoveTasksToColumn(sourceColumnID string, targetColumnID string) (int64, error)
	CountTasksByColumnID(columnID string) (int64, error)

	ListByProjectIDWithCount(projectID string) ([]dto.ColumnInfo, error)

	// BR-COL-03: validate position context
	GetByProjectIDAndPosition(projectID string, position float64) (*models.Column, error)

	// BR-COL-04: move with reposition
	GetMinTaskPosition(columnID string) (float64, error)
	MoveTasksWithReposition(sourceColumnID, targetColumnID string, newBase float64) (int64, error)

	// BR-COL-04: cascade soft delete
	SoftDeleteTasksByColumnID(columnID string) (int64, error)
	SoftDeleteAttachmentsByColumnID(columnID string) error
	SoftDeleteCommentsByColumnID(columnID string) error
}
