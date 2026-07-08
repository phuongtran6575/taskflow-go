package implement

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/helper"
	"TaskFlow-Go/internal/models"
	_interface "TaskFlow-Go/internal/repository/interface"
)

type attachmentRepository struct{ db *gorm.DB }

func NewAttachmentRepository(db *gorm.DB) _interface.AttachmentRepository {
	return &attachmentRepository{db: db}
}

func (r *attachmentRepository) WithTx(tx *gorm.DB) _interface.AttachmentRepository {
	return &attachmentRepository{db: tx}
}

func (r *attachmentRepository) Create(attachment *models.Attachment) error {
	return r.db.Create(attachment).Error
}

func (r *attachmentRepository) GetByID(id string) (*models.Attachment, error) {
	var a models.Attachment
	err := r.db.Where("id = ?", id).First(&a).Error
	return &a, err
}

func (r *attachmentRepository) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.Attachment{}).Error
}

func (r *attachmentRepository) SoftDelete(id string, scheduledAt time.Time) error {
	return r.db.Model(&models.Attachment{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"deleted_at":          time.Now(),
			"scheduled_delete_at": scheduledAt,
		}).Error
}

func (r *attachmentRepository) HardDelete(id string) error {
	return r.db.Unscoped().Where("id = ?", id).Delete(&models.Attachment{}).Error
}

func (r *attachmentRepository) ListExpiredForDeletion() ([]models.Attachment, error) {
	var attachments []models.Attachment
	err := r.db.Unscoped().
		Where("deleted_at IS NOT NULL AND scheduled_delete_at IS NOT NULL AND scheduled_delete_at <= NOW()").
		Find(&attachments).Error
	return attachments, err
}

func (r *attachmentRepository) ListByTaskIDWithPagination(taskID string, fileType string, page int, limit int) (*dto.AttachmentListResponse, error) {
	type attachmentRow struct {
		ID             string    `gorm:"column:id"`
		FileName       string    `gorm:"column:file_name"`
		FileType       string    `gorm:"column:file_type"`
		SizeBytes      int64     `gorm:"column:size_bytes"`
		CreatedAt      time.Time `gorm:"column:created_at"`
		UploaderID     string    `gorm:"column:uploader_user_id"`
		UploaderName   string    `gorm:"column:uploader_full_name"`
		UploaderAvatar *string   `gorm:"column:uploader_avatar_url"`
		TaskRef        string    `gorm:"column:task_ref"`
	}
	type taskInfo struct {
		TaskID  string `gorm:"column:task_id"`
		TaskRef string `gorm:"column:task_ref"`
	}
	var task taskInfo
	err := r.db.Table("tasks t").
		Select("t.id as task_id, CONCAT(p.key, '-', t.task_number) as task_ref").
		Joins("JOIN projects p ON p.id = t.project_id").
		Where("t.id = ?", taskID).
		First(&task).Error
	if err != nil {
		return nil, err
	}

	var total int64
	var totalBytes int64
	countQuery := r.db.Model(&models.Attachment{}).Where("task_id = ? AND deleted_at IS NULL", taskID)
	if fileType != "" {
		countQuery = countQuery.Where("file_type = ?", fileType)
	}
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, err
	}
	type sumResult struct {
		Sum int64
	}
	var sum sumResult
	if err := countQuery.Select("COALESCE(SUM(size_bytes), 0) as sum").Scan(&sum).Error; err != nil {
		return nil, err
	}
	totalBytes = sum.Sum

	offset := (page - 1) * limit
	dataQuery := r.db.Table("attachments a").
		Select(`
			a.id, a.file_name, a.file_type, a.size_bytes, a.created_at,
			u.id as uploader_user_id, u.full_name as uploader_full_name,
			u.avatar_url as uploader_avatar_url
		`).
		Joins("JOIN users u ON u.id = a.uploader_id").
		Where("a.task_id = ? AND a.deleted_at IS NULL", taskID)
	if fileType != "" {
		dataQuery = dataQuery.Where("a.file_type = ?", fileType)
	}
	var rows []attachmentRow
	if err := dataQuery.Order("a.created_at DESC").Offset(offset).Limit(limit).Scan(&rows).Error; err != nil {
		return nil, err
	}

	data := make([]dto.AttachmentInfo, len(rows))
	for i, row := range rows {
		fileGroup := helper.ClassifyFileType(row.FileType)
		thumbnailURL := fmt.Sprintf("/api/v1/files/%s/thumbnail", row.ID)
		var thumbPtr *string
		if fileGroup == "image" {
			thumbPtr = &thumbnailURL
		}

		data[i] = dto.AttachmentInfo{
			ID:        row.ID,
			FileName:  row.FileName,
			FileURL:   nil,
			FileType:  row.FileType,
			FileGroup: fileGroup,
			SizeBytes: row.SizeBytes,
			SizeDisplay: helper.FormatSizeDisplay(row.SizeBytes),
			ThumbnailURL: thumbPtr,
			Uploader: dto.AttachmentUploader{
				UserID:    row.UploaderID,
				FullName:  row.UploaderName,
				AvatarURL: row.UploaderAvatar,
			},
			CreatedAt: row.CreatedAt.Format(time.RFC3339),
		}
	}

	return &dto.AttachmentListResponse{
		TaskID:         task.TaskID,
		TaskRef:        task.TaskRef,
		TotalSizeBytes: totalBytes,
		Data:           data,
		Pagination:     *dto.NewPagination(total, dto.PaginationParam{Page: page, Limit: limit}),
	}, nil
}

func (r *attachmentRepository) GetStorageUsageByWorkspace(workspaceID string) (*dto.StorageUsageResponse, error) {
	type usageRow struct {
		ProjectID   string `gorm:"column:project_id"`
		ProjectName string `gorm:"column:project_name"`
		ProjectKey  string `gorm:"column:project_key"`
		UsedBytes   int64  `gorm:"column:used_bytes"`
		FileCount   int    `gorm:"column:file_count"`
	}
	var rows []usageRow
	err := r.db.Table("projects p").
		Select(`
			p.id as project_id, p.name as project_name, p.key as project_key,
			COALESCE(SUM(a.size_bytes), 0) as used_bytes,
			COUNT(a.id) as file_count
		`).
		Joins("LEFT JOIN tasks t ON t.project_id = p.id AND t.deleted_at IS NULL").
		Joins("LEFT JOIN attachments a ON a.task_id = t.id AND a.deleted_at IS NULL").
		Where("p.workspace_id = ? AND p.deleted_at IS NULL", workspaceID).
		Group("p.id, p.name, p.key").
		Order("used_bytes DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	breakdown := make([]dto.ProjectStorageBreakdown, len(rows))
	totalBytes := int64(0)
	for i, row := range rows {
		breakdown[i] = dto.ProjectStorageBreakdown{
			ProjectID:   row.ProjectID,
			ProjectName: row.ProjectName,
			ProjectKey:  row.ProjectKey,
			UsedBytes:   row.UsedBytes,
			UsedDisplay: helper.FormatSizeDisplay(row.UsedBytes),
			FileCount:   row.FileCount,
		}
		totalBytes += row.UsedBytes
	}

	return &dto.StorageUsageResponse{
		WorkspaceID:        workspaceID,
		Plan:               "",
		Storage:            dto.StorageInfo{},
		BreakdownByProject: breakdown,
		Warnings:           nil,
	}, nil
}


