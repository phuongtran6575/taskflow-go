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

	"TaskFlow-Go/internal/activitylog"
	"TaskFlow-Go/internal/database"
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
	tm                *database.TransactionManager
	attachmentRepo    repoInterface.AttachmentRepository
	taskRepo          repoInterface.TaskRepository
	projectRepo       repoInterface.ProjectRepository
	workspaceRepo     repoInterface.WorkspaceRepository
	activityLogRepo   repoInterface.ActivityLogRepository
	projectMemberRepo repoInterface.ProjectMemberRepository
	userRepo          repoInterface.UserRepository
}

func NewAttachmentService(
	tm *database.TransactionManager,
	attachmentRepo repoInterface.AttachmentRepository,
	taskRepo repoInterface.TaskRepository,
	projectRepo repoInterface.ProjectRepository,
	workspaceRepo repoInterface.WorkspaceRepository,
	activityLogRepo repoInterface.ActivityLogRepository,
	projectMemberRepo repoInterface.ProjectMemberRepository,
	userRepo repoInterface.UserRepository,
) _interface.AttachmentService {
	return &attachmentService{
		tm:                tm,
		attachmentRepo:    attachmentRepo,
		taskRepo:          taskRepo,
		projectRepo:       projectRepo,
		workspaceRepo:     workspaceRepo,
		activityLogRepo:   activityLogRepo,
		projectMemberRepo: projectMemberRepo,
		userRepo:          userRepo,
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

	type pendingUpload struct {
		attachment  models.Attachment
		fileGroup   string
		thumbnailURL *string
		origName    string
	}
	var pending []pendingUpload
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
			ID:         attID,
			TaskID:     taskID,
			UploaderID: &userID,
			FileName:   vf.sanitizedName,
			FileURL:    savePath,
			FileType:   vf.ext,
			SizeBytes:  vf.header.Size,
			CreatedAt:  now,
		}

		fileGroup := helper.ClassifyFileType(vf.ext)
		var thumbnailURL *string
		if fileGroup == "image" {
			url := fmt.Sprintf("/api/v1/files/%s/thumbnail", attID)
			thumbnailURL = &url
		}

		pending = append(pending, pendingUpload{
			attachment:   attachment,
			fileGroup:    fileGroup,
			thumbnailURL: thumbnailURL,
			origName:     vf.header.Filename,
		})
	}

	ref := fmt.Sprintf("%s-%d", project.Key, task.TaskNumber)

	if len(pending) > 0 {
		var fileInfos []map[string]interface{}
		for _, p := range pending {
			fileInfos = append(fileInfos, map[string]interface{}{
				"file_name":  p.attachment.FileName,
				"size_bytes": p.attachment.SizeBytes,
				"file_type":  p.attachment.FileType,
			})
		}
		actorName := s.getUserName(userID)
		meta := map[string]interface{}{
			"event":            "attachments_uploaded",
			"files":            fileInfos,
			"total_size_bytes": totalValidSize,
		}
		desc := activitylog.GenerateDescription(actorName, meta)
		snap := activitylog.BuildTaskSnapshot(ref, task.Title, project.Key)

		err = s.tm.Execute(func(tx *gorm.DB) error {
			for _, p := range pending {
				if err := tx.Create(&p.attachment).Error; err != nil {
					return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to save attachment record")
				}
			}
			s.logActivityInTx(tx, workspaceID, projectID, userID, taskID, models.ActivityActionCREATE, meta, desc, snap)
			return nil
		})
		if err != nil {
			for _, p := range pending {
				os.Remove(p.attachment.FileURL)
			}
			return nil, err
		}
	}

	var uploaded []dto.UploadedFileInfo
	for _, p := range pending {
		uploaded = append(uploaded, dto.UploadedFileInfo{
			ID:        p.attachment.ID,
			FileName:  p.attachment.FileName,
			FileType:  p.attachment.FileType,
			FileGroup: p.fileGroup,
			SizeBytes: p.attachment.SizeBytes,
			SizeDisplay: helper.FormatSizeDisplay(p.attachment.SizeBytes),
			ThumbnailURL: p.thumbnailURL,
			Uploader: dto.AttachmentUploader{
				UserID:   userID,
				FullName: "",
			},
			CreatedAt: p.attachment.CreatedAt.Format(time.RFC3339),
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
	actorName := s.getUserName(userID)
	meta := map[string]interface{}{
		"event":       "attachment_deleted",
		"file_name":   att.FileName,
		"size_bytes":  att.SizeBytes,
		"deleted_own": isOwn,
	}
	desc := activitylog.GenerateDescription(actorName, meta)
	taskSnap := activitylog.BuildTaskSnapshot("", "", project.Key)
	s.logActivity(workspaceID, projectID, userID, taskID, models.ActivityActionDELETE, meta, desc, taskSnap)

	return &dto.DeleteAttachmentResponse{
		Message:                   fmt.Sprintf("Attachment '%s' has been deleted.", att.FileName),
		DeletedAttachmentID:       attachmentID,
		ScheduledPermanentDeleteAt: scheduledAt.Format(time.RFC3339),
	}, nil
}

func (s *attachmentService) logActivity(workspaceID, projectID, userID, entityID string, action models.ActivityAction, metadata map[string]interface{}, description string, entitySnapshot map[string]interface{}) {
	wsID := workspaceID
	uID := userID
	var metaStr *string
	if metadata != nil {
		b, _ := json.Marshal(metadata)
		str := string(b)
		metaStr = &str
	}
	var snapStr *string
	if entitySnapshot != nil {
		b, _ := json.Marshal(entitySnapshot)
		str := string(b)
		snapStr = &str
	}
	var descPtr *string
	if description != "" {
		descPtr = &description
	}
	_ = s.activityLogRepo.Create(&models.ActivityLog{
		WorkspaceID:    &wsID,
		ProjectID:      &projectID,
		UserID:         &uID,
		Action:         action,
		EntityType:     models.EntityTypeTASK,
		EntityID:       entityID,
		Description:    descPtr,
		Metadata:       metaStr,
		EntitySnapshot: snapStr,
	})
}

func (s *attachmentService) logActivityInTx(tx *gorm.DB, workspaceID, projectID, userID, entityID string, action models.ActivityAction, metadata map[string]interface{}, description string, entitySnapshot map[string]interface{}) {
	wsID := workspaceID
	uID := userID
	var metaStr *string
	if metadata != nil {
		b, _ := json.Marshal(metadata)
		str := string(b)
		metaStr = &str
	}
	var snapStr *string
	if entitySnapshot != nil {
		b, _ := json.Marshal(entitySnapshot)
		str := string(b)
		snapStr = &str
	}
	var descPtr *string
	if description != "" {
		descPtr = &description
	}
	_ = tx.Create(&models.ActivityLog{
		WorkspaceID:    &wsID,
		ProjectID:      &projectID,
		UserID:         &uID,
		Action:         action,
		EntityType:     models.EntityTypeTASK,
		EntityID:       entityID,
		Description:    descPtr,
		Metadata:       metaStr,
		EntitySnapshot: snapStr,
	}).Error
}

func (s *attachmentService) getUserName(userID string) string {
	u, err := s.userRepo.GetByID(userID)
	if err != nil {
		return userID
	}
	return u.FullName
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
