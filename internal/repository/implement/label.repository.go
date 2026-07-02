package implement

import (
	"fmt"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
	_interface "TaskFlow-Go/internal/repository/interface"
)

type labelRepository struct{ db *gorm.DB }

func NewLabelRepository(db *gorm.DB) _interface.LabelRepository {
	return &labelRepository{db: db}
}

func (r *labelRepository) Create(label *models.Label) error {
	return r.db.Create(label).Error
}

func (r *labelRepository) GetByID(id string) (*models.Label, error) {
	var label models.Label
	err := r.db.Where("id = ?", id).First(&label).Error
	return &label, err
}

func (r *labelRepository) ListByProjectID(projectID string) ([]models.Label, error) {
	var labels []models.Label
	err := r.db.Where("project_id = ?", projectID).Find(&labels).Error
	return labels, err
}

func (r *labelRepository) Update(label *models.Label) error {
	return r.db.Save(label).Error
}

func (r *labelRepository) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.Label{}).Error
}

func (r *labelRepository) AddToTask(taskID, labelID string) error {
	return r.db.Create(&models.TaskLabel{TaskID: taskID, LabelID: labelID}).Error
}

func (r *labelRepository) RemoveFromTask(taskID, labelID string) error {
	return r.db.Where("task_id = ? AND label_id = ?", taskID, labelID).
		Delete(&models.TaskLabel{}).Error
}

func (r *labelRepository) ListByProjectIDWithCount(projectID string, search string, withTaskCount bool) ([]dto.LabelInfo, error) {
	type labelRow struct {
		ID        string    `gorm:"column:id"`
		Name      string    `gorm:"column:name"`
		Color     string    `gorm:"column:color"`
		TaskCount int       `gorm:"column:task_count"`
	}

	query := r.db.Table("labels l").
		Select("l.id, l.name, l.color, COUNT(tl.task_id) as task_count").
		Joins("LEFT JOIN task_labels tl ON tl.label_id = l.id").
		Where("l.project_id = ? AND l.deleted_at IS NULL", projectID).
		Group("l.id, l.name, l.color")

	if search != "" {
		pattern := "%" + search + "%"
		query = query.Where("l.name ILIKE ?", pattern)
	}

	var rows []labelRow
	if err := query.Order("l.name ASC").Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]dto.LabelInfo, len(rows))
	for i, row := range rows {
		result[i] = dto.LabelInfo{
			ID:    row.ID,
			Name:  row.Name,
			Color: row.Color,
		}
		if withTaskCount {
			result[i].TaskCount = row.TaskCount
		}
	}
	return result, nil
}

func (r *labelRepository) ListByTaskIDWithRef(taskID string) (*dto.TaskLabelListResponse, error) {
	type taskInfo struct {
		TaskID     string `gorm:"column:task_id"`
		TaskNumber int    `gorm:"column:task_number"`
		ProjectKey string `gorm:"column:project_key"`
	}
	var info taskInfo
	err := r.db.Table("tasks t").
		Select("t.id as task_id, t.task_number, p.key as project_key").
		Joins("JOIN projects p ON p.id = t.project_id").
		Where("t.id = ?", taskID).
		First(&info).Error
	if err != nil {
		return nil, err
	}

	var labels []dto.TaskLabelInfo
	err = r.db.Table("labels l").
		Select("l.id, l.name, l.color").
		Joins("JOIN task_labels tl ON tl.label_id = l.id").
		Where("tl.task_id = ?", taskID).
		Scan(&labels).Error
	if err != nil {
		return nil, err
	}

	taskRef := fmt.Sprintf("%s-%d", info.ProjectKey, info.TaskNumber)
	return &dto.TaskLabelListResponse{
		TaskID:  info.TaskID,
		TaskRef: taskRef,
		Data:    labels,
		Total:   len(labels),
	}, nil
}

func (r *labelRepository) ExistsByNameInProject(projectID string, name string, excludeID string) (bool, error) {
	var count int64
	query := r.db.Model(&models.Label{}).
		Where("project_id = ? AND name = ? AND deleted_at IS NULL", projectID, name)
	if excludeID != "" {
		query = query.Where("id != ?", excludeID)
	}
	err := query.Count(&count).Error
	return count > 0, err
}

func (r *labelRepository) CountByProjectID(projectID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Label{}).
		Where("project_id = ? AND deleted_at IS NULL", projectID).
		Count(&count).Error
	return count, err
}
