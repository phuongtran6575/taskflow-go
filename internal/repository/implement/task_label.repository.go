package implement

import (
	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
	_interface "TaskFlow-Go/internal/repository/interface"
)

type taskLabelRepository struct{ db *gorm.DB }

// ListByTaskIDWithDetail implements [_interface.TaskLabelRepository].

func NewTaskLabelRepository(db *gorm.DB) _interface.TaskLabelRepository {
	return &taskLabelRepository{db: db}
}

func (r *taskLabelRepository) ListByTaskIDs(taskIDs []string) (map[string][]dto.LabelRef, error) {
	if len(taskIDs) == 0 {
		return map[string][]dto.LabelRef{}, nil
	}
	var rows []struct {
		TaskID string
		dto.LabelRef
	}
	err := r.db.Table("task_labels tl").
		Select("tl.task_id, l.id, l.name, l.color").
		Joins("JOIN labels l ON tl.label_id = l.id").
		Where("tl.task_id IN ?", taskIDs).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[string][]dto.LabelRef, len(taskIDs))
	for _, row := range rows {
		result[row.TaskID] = append(result[row.TaskID], row.LabelRef)
	}
	return result, nil
}

func (r *taskLabelRepository) ListTaskLabelsByTaskID(taskID string) ([]models.TaskLabel, error) {
	var tls []models.TaskLabel
	err := r.db.Where("task_id = ?", taskID).Find(&tls).Error
	return tls, err
}

func (r *taskLabelRepository) ListByTaskID(taskID string) ([]dto.LabelRef, error) {
	var tls []dto.LabelRef
	err := r.db.Table("labels l").Select("l.id, l.name, l.color").Joins("JOIN task_labels tl ON tl.label_id = l.id").Where("tl.task_id = ?", taskID).Scan(&tls).Error
	return tls, err
}

// BulkTaskLabel implements [_interface.TaskLabelRepository].
func (r *taskLabelRepository) BulkTaskLabel(taskID string, labelIDs []string) error {
	var tls []models.TaskLabel
	for _, labelID := range labelIDs {
		tls = append(tls, models.TaskLabel{
			TaskID:  taskID,
			LabelID: labelID,
		})
	}
	return r.db.Create(&tls).Error
}

func (r *taskLabelRepository) ListByLabelID(labelID string) ([]models.TaskLabel, error) {
	var tls []models.TaskLabel
	err := r.db.Where("label_id = ?", labelID).Find(&tls).Error
	return tls, err
}
