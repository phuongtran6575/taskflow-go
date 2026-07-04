package job

import (
	"gorm.io/gorm"

	"TaskFlow-Go/internal/models"
)

func cascadeSoftDeleteWorkspace(db *gorm.DB, workspaceID string) error {
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Where("workspace_id = ?", workspaceID).Delete(&models.WorkspaceInvite{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Where("workspace_id = ?", workspaceID).Delete(&models.Role{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	var projectIDs []string
	if err := tx.Model(&models.Project{}).
		Where("workspace_id = ?", workspaceID).
		Pluck("id", &projectIDs).Error; err != nil {
		tx.Rollback()
		return err
	}

	if len(projectIDs) > 0 {
		if err := tx.Where("project_id IN ?", projectIDs).Delete(&models.Column{}).Error; err != nil {
			tx.Rollback()
			return err
		}

		if err := tx.Where("project_id IN ?", projectIDs).Delete(&models.Label{}).Error; err != nil {
			tx.Rollback()
			return err
		}

		if err := tx.Where("project_id IN ?", projectIDs).Delete(&models.Project{}).Error; err != nil {
			tx.Rollback()
			return err
		}

		var taskIDs []string
		if err := tx.Model(&models.Task{}).
			Where("project_id IN ?", projectIDs).
			Pluck("id", &taskIDs).Error; err != nil {
			tx.Rollback()
			return err
		}

		if len(taskIDs) > 0 {
			if err := tx.Where("task_id IN ?", taskIDs).Delete(&models.Attachment{}).Error; err != nil {
				tx.Rollback()
				return err
			}

			if err := tx.Where("task_id IN ?", taskIDs).Delete(&models.Comment{}).Error; err != nil {
				tx.Rollback()
				return err
			}

			if err := tx.Where("id IN ?", taskIDs).Delete(&models.Task{}).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit().Error
}
