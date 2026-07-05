package implement

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/helper"
	"TaskFlow-Go/internal/models"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
)

const (
	maxFileSize  = 50 * 1024 * 1024
	maxBatchSize = 10
)

type attachmentService struct {
	attachmentRepo    repoInterface.AttachmentRepository
	taskRepo          repoInterface.TaskRepository
	projectRepo       repoInterface.ProjectRepository
	workspaceRepo     repoInterface.WorkspaceRepository
	activityLogRepo   repoInterface.ActivityLogRepository
	projectMemberRepo repoInterface.ProjectMemberRepository
}

func NewAttachmentService(
	attachmentRepo repoInterface.AttachmentRepository,
	taskRepo repoInterface.TaskRepository,
	projectRepo repoInterface.ProjectRepository,
	workspaceRepo repoInterface.WorkspaceRepository,
	activityLogRepo repoInterface.ActivityLogRepository,
	projectMemberRepo repoInterface.ProjectMemberRepository,
) _interface.AttachmentService {
	return &attachmentService{
		attachmentRepo:    attachmentRepo,
		taskRepo:          taskRepo,
		projectRepo:       projectRepo,
		workspaceRepo:     workspaceRepo,
		activityLogRepo:   activityLogRepo,
		projectMemberRepo: projectMemberRepo,
	}
}

func (s *attachmentService) getProjectOrFail(workspaceID, projectID string) (*models.Project, error) {
	project, err := s.projectRepo.GetByID(projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrProjectNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get project")
	}
	if project.WorkspaceID != workspaceID || project.DeletedAt.Valid {
		return nil, apperror.ErrProjectNotFound
	}
	return project, nil
}

func (s *attachmentService) getTaskOrFail(projectID, taskID string) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrTaskNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get task")
	}
	if task.ProjectID != projectID || task.DeletedAt.Valid {
		return nil, apperror.ErrTaskNotFound
	}
	return task, nil
}

func (s *attachmentService) getAttachmentOrFail(taskID, attachmentID string) (*models.Attachment, error) {
	att, err := s.attachmentRepo.GetByID(attachmentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrAttachmentNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get attachment")
	}
	if att.TaskID != taskID || att.DeletedAt.Valid {
		return nil, apperror.ErrAttachmentNotFound
	}
	return att, nil
}

func (s *attachmentService) ListAttachments(workspaceID string, userID string, projectID string, taskID string, fileType string, page int, limit int) (*dto.AttachmentListResponse, error) {
	page = max(page, 1)
	if limit <= 0 {
		limit = 20
	}

	_, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	_, err = s.getTaskOrFail(projectID, taskID)
	if err != nil {
		return nil, err
	}

	result, err := s.attachmentRepo.ListByTaskIDWithPagination(taskID, fileType, page, limit)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrTaskNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list attachments")
	}
	return result, nil
}

func (s *attachmentService) UploadAttachments(workspaceID string, userID string, projectID string, taskID string, files []*multipart.FileHeader) (*dto.UploadAttachmentsResponse, error) {
	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if project.IsArchived {
		return nil, apperror.ErrProjectArchived
	}

	task, err := s.getTaskOrFail(projectID, taskID)
	if err != nil {
		return nil, err
	}

	if len(files) > maxBatchSize {
		return nil, apperror.ErrBatchSizeExceeded
	}

	workspace, err := s.workspaceRepo.GetByID(workspaceID)
	if err != nil {
		return nil, apperror.ErrWorkspaceNotFound
	}

	var validFiles []struct {
		header *multipart.FileHeader
		ext    string
		sanitizedName string
	}
	var totalValidSize int64
	var preFailed []dto.FailedFile

	for _, fh := range files {
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fh.Filename), "."))

		if helper.IsBlockedExtension(ext) {
			preFailed = append(preFailed, dto.FailedFile{
				FileName: fh.Filename,
				Reason:   "BLOCKED_FILE_TYPE",
				Message:  fmt.Sprintf("File type .%s is not allowed.", ext),
			})
			continue
		}

		if fh.Size > maxFileSize {
			preFailed = append(preFailed, dto.FailedFile{
				FileName: fh.Filename,
				Reason:   "FILE_TOO_LARGE",
				Message:  "File exceeds 50MB limit.",
			})
			continue
		}

		sanitized := helper.SanitizeFileName(fh.Filename)

		validFiles = append(validFiles, struct {
			header        *multipart.FileHeader
			ext           string
			sanitizedName string
		}{header: fh, ext: ext, sanitizedName: sanitized})
		totalValidSize += fh.Size
	}

	storageUsage, err := s.attachmentRepo.GetStorageUsageByWorkspace(workspaceID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check storage")
	}
	usedBytes := int64(0)
	for _, p := range storageUsage.BreakdownByProject {
		usedBytes += p.UsedBytes
	}

	if len(validFiles) > 0 {
		if err := helper.CheckStorageLimit(workspace.Plan, usedBytes, totalValidSize); err != nil {
			return nil, apperror.NewAppError(
				http.StatusInsufficientStorage,
				"STORAGE_QUOTA_EXCEEDED",
				fmt.Sprintf("Workspace storage quota exceeded. Need %s more.",
					helper.FormatSizeDisplay((usedBytes+totalValidSize)-helper.GetPlanLimits(workspace.Plan).MaxStorageBytes)),
			)
		}
	}

	var uploaded []dto.UploadedFileInfo
	var failed []dto.FailedFile
	failed = append(failed, preFailed...)

	for _, vf := range validFiles {
		attID := uuid.New().String()
		savePath := filepath.Join(".", "attachments", workspaceID, projectID, taskID, attID+"."+vf.ext)

		if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
			failed = append(failed, dto.FailedFile{
				FileName: vf.header.Filename,
				Reason:   "UPLOAD_FAILED",
				Message:  "Failed to create upload directory.",
			})
			continue
		}

		src, err := vf.header.Open()
		if err != nil {
			failed = append(failed, dto.FailedFile{
				FileName: vf.header.Filename,
				Reason:   "UPLOAD_FAILED",
				Message:  "Failed to read file.",
			})
			continue
		}

		dst, err := os.Create(savePath)
		if err != nil {
			src.Close()
			failed = append(failed, dto.FailedFile{
				FileName: vf.header.Filename,
				Reason:   "UPLOAD_FAILED",
				Message:  "Failed to save file.",
			})
			continue
		}

		_, copyErr := io.Copy(dst, src)
		src.Close()
		dst.Close()

		if copyErr != nil {
			failed = append(failed, dto.FailedFile{
				FileName: vf.header.Filename,
				Reason:   "UPLOAD_FAILED",
				Message:  "Failed to write file.",
			})
			os.Remove(savePath)
			continue
		}

		now := time.Now()
		attachment := models.Attachment{
			ID:        attID,
			TaskID:    taskID,
			UploaderID: &userID,
			FileName:  vf.sanitizedName,
			FileURL:   savePath,
			FileType:  vf.ext,
			SizeBytes: vf.header.Size,
			CreatedAt: now,
		}
		if err := s.attachmentRepo.Create(&attachment); err != nil {
			failed = append(failed, dto.FailedFile{
				FileName: vf.header.Filename,
				Reason:   "UPLOAD_FAILED",
				Message:  "Failed to save attachment record.",
			})
			os.Remove(savePath)
			continue
		}

		fileGroup := helper.ClassifyFileType(vf.ext)
		var thumbnailURL *string
		if fileGroup == "image" {
			url := fmt.Sprintf("/api/v1/files/%s/thumbnail", attID)
			thumbnailURL = &url
		}

		uploaded = append(uploaded, dto.UploadedFileInfo{
			ID:        attID,
			FileName:  vf.sanitizedName,
			FileType:  vf.ext,
			FileGroup: fileGroup,
			SizeBytes: vf.header.Size,
			SizeDisplay: helper.FormatSizeDisplay(vf.header.Size),
			ThumbnailURL: thumbnailURL,
			Uploader: dto.AttachmentUploader{
				UserID:   userID,
				FullName: "",
			},
			CreatedAt: now.Format(time.RFC3339),
		})
	}

	ref := fmt.Sprintf("%s-%d", project.Key, task.TaskNumber)

	if len(uploaded) > 0 {
		var fileInfos []map[string]interface{}
		for _, u := range uploaded {
			fileInfos = append(fileInfos, map[string]interface{}{
				"file_name":  u.FileName,
				"size_bytes": u.SizeBytes,
				"file_type":  u.FileType,
			})
		}
		meta := map[string]interface{}{
			"event":           "attachments_uploaded",
			"files":           fileInfos,
			"total_size_bytes": totalValidSize,
		}
		metaBytes, _ := json.Marshal(meta)
		metaStr := string(metaBytes)
		s.activityLogRepo.Create(&models.ActivityLog{
			WorkspaceID: &workspaceID,
			ProjectID:   &projectID,
			UserID:      &userID,
			Action:      models.ActivityActionCREATE,
			EntityType:  models.EntityTypeTASK,
			EntityID:    taskID,
			Metadata:    &metaStr,
		})
	}

	return &dto.UploadAttachmentsResponse{
		TaskID:        taskID,
		TaskRef:       ref,
		Uploaded:      uploaded,
		Failed:        failed,
		TotalUploaded: len(uploaded),
	}, nil
}

func (s *attachmentService) GetDownloadUrl(workspaceID string, userID string, projectID string, taskID string, attachmentID string, disposition string) (*dto.DownloadUrlResponse, error) {
	_, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}

	att, err := s.getAttachmentOrFail(taskID, attachmentID)
	if err != nil {
		return nil, err
	}

	expiresIn := 900
	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)
	downloadURL := fmt.Sprintf("/api/v1/files/%s?disposition=%s&expires=%d", attachmentID, disposition, expiresAt.Unix())

	return &dto.DownloadUrlResponse{
		AttachmentID:     attachmentID,
		FileName:         att.FileName,
		DownloadURL:      downloadURL,
		ExpiresInSeconds: expiresIn,
		ExpiresAt:        expiresAt.Format(time.RFC3339),
		Disposition:      disposition,
	}, nil
}

func (s *attachmentService) GetPreviewUrl(workspaceID string, userID string, projectID string, taskID string, attachmentID string) (*dto.PreviewUrlResponse, error) {
	_, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}

	att, err := s.getAttachmentOrFail(taskID, attachmentID)
	if err != nil {
		return nil, err
	}

	expiresIn := 900
	fileGroup := helper.ClassifyFileType(att.FileType)
	isPreviewable := fileGroup == "image" || att.FileType == "pdf"

	previewURL := fmt.Sprintf("/api/v1/files/%s?disposition=inline&expires=%d", attachmentID, time.Now().Add(time.Duration(expiresIn)*time.Second).Unix())

	thumbnailURL := ""
	if fileGroup == "image" {
		thumbnailURL = fmt.Sprintf("/api/v1/files/%s/thumbnail", attachmentID)
	}

	return &dto.PreviewUrlResponse{
		AttachmentID:     attachmentID,
		FileName:         att.FileName,
		FileType:         att.FileType,
		PreviewURL:       previewURL,
		ThumbnailURL:     thumbnailURL,
		IsPreviewable:    isPreviewable,
		ExpiresInSeconds: expiresIn,
	}, nil
}

func (s *attachmentService) DeleteAttachment(workspaceID string, userID string, projectID string, taskID string, attachmentID string) (*dto.DeleteAttachmentResponse, error) {
	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if project.IsArchived {
		return nil, apperror.ErrProjectArchived
	}

	att, err := s.getAttachmentOrFail(taskID, attachmentID)
	if err != nil {
		return nil, err
	}

	if err := s.canDeleteAttachment(workspaceID, projectID, userID, att); err != nil {
		return nil, err
	}

	scheduledAt := time.Now().Add(30 * 24 * time.Hour)
	if err := s.attachmentRepo.SoftDelete(attachmentID, scheduledAt); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete attachment")
	}

	isOwn := att.UploaderID != nil && *att.UploaderID == userID
	meta := map[string]interface{}{
		"event":       "attachment_deleted",
		"file_name":   att.FileName,
		"size_bytes":  att.SizeBytes,
		"deleted_own": isOwn,
	}
	metaBytes, _ := json.Marshal(meta)
	metaStr := string(metaBytes)
	_ = s.activityLogRepo.Create(&models.ActivityLog{
		WorkspaceID: &workspaceID,
		ProjectID:   &projectID,
		UserID:      &userID,
		Action:      models.ActivityActionDELETE,
		EntityType:  models.EntityTypeTASK,
		EntityID:    taskID,
		Metadata:    &metaStr,
	})

	return &dto.DeleteAttachmentResponse{
		Message:                   fmt.Sprintf("Attachment '%s' has been deleted.", att.FileName),
		DeletedAttachmentID:       attachmentID,
		ScheduledPermanentDeleteAt: scheduledAt.Format(time.RFC3339),
	}, nil
}

func (s *attachmentService) canDeleteAttachment(workspaceID, projectID, userID string, att *models.Attachment) error {
	ws, err := s.workspaceRepo.GetByID(workspaceID)
	if err == nil && ws.OwnerID == userID {
		return nil
	}

	isUploader := att.UploaderID != nil && *att.UploaderID == userID

	if isUploader {
		hasPerm, err := s.projectMemberRepo.HasPermission(projectID, userID, "attachment:delete_own")
		if err != nil {
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check permission")
		}
		if hasPerm {
			return nil
		}
	}

	hasAny, err := s.projectMemberRepo.HasPermission(projectID, userID, "attachment:delete_any")
	if err != nil {
		return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check permission")
	}
	if hasAny {
		return nil
	}

	return apperror.ErrForbidden
}
