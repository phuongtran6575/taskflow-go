package implement

import (
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
	maxFileSize    = 50 * 1024 * 1024
	maxBatchSize   = 10
)

var blockedExtensions = map[string]bool{
	"exe": true, "sh": true, "bat": true, "cmd": true,
	"dmg": true, "apk": true, "msix": true, "scr": true, "com": true, "vbs": true,
}

type attachmentService struct {
	attachmentRepo repoInterface.AttachmentRepository
	taskRepo       repoInterface.TaskRepository
	projectRepo    repoInterface.ProjectRepository
	workspaceRepo  repoInterface.WorkspaceRepository
}

func NewAttachmentService(
	attachmentRepo repoInterface.AttachmentRepository,
	taskRepo repoInterface.TaskRepository,
	projectRepo repoInterface.ProjectRepository,
	workspaceRepo repoInterface.WorkspaceRepository,
) _interface.AttachmentService {
	return &attachmentService{
		attachmentRepo: attachmentRepo,
		taskRepo:       taskRepo,
		projectRepo:    projectRepo,
		workspaceRepo:  workspaceRepo,
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
	if project.WorkspaceID != workspaceID {
		return nil, apperror.ErrProjectNotFound
	}
	if project.DeletedAt.Valid {
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

func (s *attachmentService) ListAttachments(workspaceID string, userID string, projectID string, taskID string, fileType string, page int, pageSize int) (*dto.AttachmentListResponse, error) {
	page = max(page, 1)
	if pageSize <= 0 {
		pageSize = 20
	}

	_, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	_, err = s.getTaskOrFail(projectID, taskID)
	if err != nil {
		return nil, err
	}

	result, err := s.attachmentRepo.ListByTaskIDWithPagination(taskID, fileType, page, pageSize)
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

	storageUsage, err := s.attachmentRepo.GetStorageUsageByWorkspace(workspaceID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check storage")
	}
	usedBytes := int64(0)
	for _, p := range storageUsage.BreakdownByProject {
		usedBytes += p.UsedBytes
	}
	if err := helper.CheckStorageLimit(workspace.Plan, usedBytes, 0); err != nil {
		return nil, err
	}

	var uploaded []dto.UploadedFileInfo
	var failed []dto.FailedFile
	totalSizeNew := int64(0)

	for _, fh := range files {
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fh.Filename), "."))
		if blockedExtensions[ext] {
			failed = append(failed, dto.FailedFile{
				FileName: fh.Filename,
				Reason:   "BLOCKED_FILE_TYPE",
				Message:  fmt.Sprintf("File type .%s is not allowed.", ext),
			})
			continue
		}

		if fh.Size > maxFileSize {
			failed = append(failed, dto.FailedFile{
				FileName: fh.Filename,
				Reason:   "FILE_TOO_LARGE",
				Message:  "File exceeds 50MB limit.",
			})
			continue
		}

		if err := helper.CheckStorageLimit(workspace.Plan, usedBytes, totalSizeNew+fh.Size); err != nil {
			failed = append(failed, dto.FailedFile{
				FileName: fh.Filename,
				Reason:   "STORAGE_QUOTA_EXCEEDED",
				Message:  "Workspace storage quota exceeded.",
			})
			continue
		}

		attachment := models.Attachment{
			TaskID:     taskID,
			UploaderID: &userID,
			FileName:   fh.Filename,
			FileType:   ext,
			SizeBytes:  fh.Size,
		}
		if err := s.attachmentRepo.Create(&attachment); err != nil {
			failed = append(failed, dto.FailedFile{
				FileName: fh.Filename,
				Reason:   "UPLOAD_FAILED",
				Message:  "Failed to save attachment.",
			})
			continue
		}

		savePath := filepath.Join(".", "uploads", workspaceID, taskID, attachment.ID+"."+ext)
		if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
			failed = append(failed, dto.FailedFile{
				FileName: fh.Filename,
				Reason:   "UPLOAD_FAILED",
				Message:  "Failed to create upload directory.",
			})
			continue
		}

		src, err := fh.Open()
		if err != nil {
			failed = append(failed, dto.FailedFile{
				FileName: fh.Filename,
				Reason:   "UPLOAD_FAILED",
				Message:  "Failed to read file.",
			})
			continue
		}

		dst, err := os.Create(savePath)
		if err != nil {
			src.Close()
			failed = append(failed, dto.FailedFile{
				FileName: fh.Filename,
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
				FileName: fh.Filename,
				Reason:   "UPLOAD_FAILED",
				Message:  "Failed to write file.",
			})
			continue
		}

		attachment.FileURL = savePath
		_ = s.attachmentRepo.Create(&attachment)

		fileGroup := classifyFileType(ext)
		var thumbnailURL *string
		if fileGroup == "image" {
			url := fmt.Sprintf("/thumbnails/%s.webp", attachment.ID)
			thumbnailURL = &url
		}

		uploaded = append(uploaded, dto.UploadedFileInfo{
			ID:        attachment.ID,
			FileName:  fh.Filename,
			FileType:  ext,
			FileGroup: fileGroup,
			SizeBytes: fh.Size,
			SizeDisplay: formatSizeDisplay(fh.Size),
			ThumbnailURL: thumbnailURL,
			Uploader: dto.AttachmentUploader{
				UserID:   userID,
				FullName: "",
			},
			CreatedAt: attachment.CreatedAt.Format(time.RFC3339),
		})
		totalSizeNew += fh.Size
	}

	_ = uuid.New()
	ref := fmt.Sprintf("%s-%d", project.Key, task.TaskNumber)

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
	fileGroup := classifyFileType(att.FileType)
	isPreviewable := fileGroup == "image" || att.FileType == "pdf"

	previewURL := fmt.Sprintf("/api/v1/files/%s?disposition=inline&expires=%d", attachmentID, time.Now().Add(time.Duration(expiresIn)*time.Second).Unix())

	thumbnailURL := ""
	if fileGroup == "image" {
		thumbnailURL = fmt.Sprintf("/thumbnails/%s.webp", attachmentID)
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

	if att.UploaderID != nil && *att.UploaderID != userID {
		return nil, apperror.ErrForbidden
	}

	if err := s.attachmentRepo.Delete(attachmentID); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete attachment")
	}

	scheduledAt := time.Now().Add(30 * 24 * time.Hour)

	return &dto.DeleteAttachmentResponse{
		Message:                   fmt.Sprintf("Attachment '%s' has been deleted.", att.FileName),
		DeletedAttachmentID:       attachmentID,
		ScheduledPermanentDeleteAt: scheduledAt.Format(time.RFC3339),
	}, nil
}

func classifyFileType(ext string) string {
	imageTypes := map[string]bool{"jpg": true, "jpeg": true, "png": true, "gif": true, "webp": true, "svg": true}
	documentTypes := map[string]bool{"pdf": true, "doc": true, "docx": true, "xls": true, "xlsx": true, "txt": true, "csv": true, "ppt": true, "pptx": true}
	videoTypes := map[string]bool{"mp4": true, "mov": true, "avi": true, "mkv": true, "webm": true}
	if imageTypes[ext] {
		return "image"
	}
	if documentTypes[ext] {
		return "document"
	}
	if videoTypes[ext] {
		return "video"
	}
	return "other"
}

func formatSizeDisplay(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.0f KB", float64(bytes)/1024)
	}
	if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.0f MB", float64(bytes)/(1024*1024))
	}
	return fmt.Sprintf("%.0f GB", float64(bytes)/(1024*1024*1024))
}


