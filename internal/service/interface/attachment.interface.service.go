package _interface

import (
	"TaskFlow-Go/internal/dto"
	"mime/multipart"
)

type AttachmentService interface {
	ListAttachments(workspaceID string, userID string, projectID string, taskID string, fileType string, page int, limit int) (*dto.AttachmentListResponse, error)
	UploadAttachments(workspaceID string, userID string, projectID string, taskID string, files []*multipart.FileHeader) (*dto.UploadAttachmentsResponse, error)
	GetDownloadUrl(workspaceID string, userID string, projectID string, taskID string, attachmentID string, disposition string) (*dto.DownloadUrlResponse, error)
	GetPreviewUrl(workspaceID string, userID string, projectID string, taskID string, attachmentID string) (*dto.PreviewUrlResponse, error)
	DeleteAttachment(workspaceID string, userID string, projectID string, taskID string, attachmentID string) (*dto.DeleteAttachmentResponse, error)
}
