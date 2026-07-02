package implement

import (
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
	_interface "TaskFlow-Go/internal/repository/interface"
)

type columnRepository struct{ db *gorm.DB }

func (r *columnRepository) WithTx(tx *gorm.DB) _interface.ColumnRepository {
	return &columnRepository{db: tx}
}

// CreateListColumn implements [_interface.ColumnRepository].

func NewColumnRepository(db *gorm.DB) _interface.ColumnRepository {
	return &columnRepository{db: db}
}

func (r *columnRepository) CreateListColumn(columns []models.Column) error {
	return r.db.Create(&columns).Error
}

func (r *columnRepository) Create(column *models.Column) error {
	return r.db.Create(column).Error
}

func (r *columnRepository) GetByID(id string) (*models.Column, error) {
	var col models.Column
	err := r.db.Where("id = ?", id).First(&col).Error
	return &col, err
}

func (r *columnRepository) ListByProjectID(projectID string) ([]models.Column, error) {
	var columns []models.Column
	err := r.db.Where("project_id = ?", projectID).Order("position ASC").Find(&columns).Error
	return columns, err
}

func (r *columnRepository) Update(column *models.Column) error {
	return r.db.Save(column).Error
}

func (r *columnRepository) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.Column{}).Error
}

func (r *columnRepository) Reorder(projectID string, columnIDs []string) error {
	if columnIDs == nil {
		var cols []models.Column
		if err := r.db.Where("project_id = ?", projectID).Order("position ASC").Find(&cols).Error; err != nil {
			return err
		}
		return r.db.Transaction(func(tx *gorm.DB) error {
			for i, col := range cols {
				newPos := float64((i + 1) * 1000)
				if err := tx.Model(&models.Column{}).
					Where("id = ?", col.ID).
					Update("position", newPos).Error; err != nil {
					return err
				}
			}
			return nil
		})
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		for i, id := range columnIDs {
			if err := tx.Model(&models.Column{}).
				Where("id = ? AND project_id = ?", id, projectID).
				Update("position", float64((i+1)*1000)).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *columnRepository) MoveTasksToColumn(sourceColumnID string, targetColumnID string) (int64, error) {
	result := r.db.Model(&models.Task{}).
		Where("column_id = ?", sourceColumnID).
		Update("column_id", targetColumnID)
	return result.RowsAffected, result.Error
}

func (r *columnRepository) CountTasksByColumnID(columnID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Task{}).Where("column_id = ?", columnID).Count(&count).Error
	return count, err
}

func (r *columnRepository) ListByProjectIDWithCount(projectID string) ([]dto.ColumnInfo, error) {
	type columnWithCount struct {
		ID        string    `gorm:"column:id"`
		Title     string    `gorm:"column:title"`
		Position  float64   `gorm:"column:position"`
		IsDone    bool      `gorm:"column:is_done"`
		TaskCount int       `gorm:"column:task_count"`
		CreatedAt time.Time `gorm:"column:created_at"`
		UpdatedAt time.Time `gorm:"column:updated_at"`
	}
	var rows []columnWithCount
	err := r.db.Table("columns c").
		Select("c.id, c.title, c.position, c.is_done, c.created_at, c.updated_at, "+
			"(SELECT COUNT(*) FROM tasks t WHERE t.column_id = c.id AND t.deleted_at IS NULL) as task_count").
		Where("c.project_id = ?", projectID).
		Order("c.position ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	info := make([]dto.ColumnInfo, len(rows))
	for i, row := range rows {
		info[i] = dto.ColumnInfo{
			ID:        row.ID,
			Title:     row.Title,
			Position:  row.Position,
			IsDone:    row.IsDone,
			TaskCount: row.TaskCount,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}
	}
	return info, nil
}
