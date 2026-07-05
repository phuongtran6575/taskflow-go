package job

import (
	"gorm.io/gorm"

	"TaskFlow-Go/internal/models"
)

func cascadeSoftDeleteProject(db *gorm.DB, projectID string) error {
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Soft delete columns
	if err := tx.Where("project_id = ?", projectID).Delete(&models.Column{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Soft delete labels
	if err := tx.Where("project_id = ?", projectID).Delete(&models.Label{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Get task IDs for cascade
	var taskIDs []string
	if err := tx.Model(&models.Task{}).
		Where("project_id = ?", projectID).
		Pluck("id", &taskIDs).Error; err != nil {
		tx.Rollback()
		return err
	}

	if len(taskIDs) > 0 {
		// Soft delete attachments (file records, physical files kept for 30 days)
		if err := tx.Where("task_id IN ?", taskIDs).Delete(&models.Attachment{}).Error; err != nil {
			tx.Rollback()
			return err
		}

		// Soft delete comments
		if err := tx.Where("task_id IN ?", taskIDs).Delete(&models.Comment{}).Error; err != nil {
			tx.Rollback()
			return err
		}

		// Soft delete tasks
		if err := tx.Where("id IN ?", taskIDs).Delete(&models.Task{}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// Note: project_members, task_assignees, activity_logs, notifications are PRESERVED
	// (not soft-deleted) as per BR-PROJ-06

	return tx.Commit().Error
}
