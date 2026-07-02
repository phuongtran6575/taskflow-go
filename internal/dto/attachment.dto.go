package dto

type AttachmentInfo struct {
	ID            string         `json:"id"`
	FileName      string         `json:"file_name"`
	FileURL       *string        `json:"file_url"`
	FileType      string         `json:"file_type"`
	FileGroup     string         `json:"file_group"`
	SizeBytes     int64          `json:"size_bytes"`
	SizeDisplay   string         `json:"size_display"`
	ThumbnailURL  *string        `json:"thumbnail_url"`
	Uploader      AttachmentUploader `json:"uploader"`
	CreatedAt     string         `json:"created_at"`
}

type AttachmentUploader struct {
	UserID   string  `json:"user_id"`
	FullName string  `json:"full_name"`
	AvatarURL *string `json:"avatar_url"`
}

type AttachmentListResponse struct {
	TaskID        string           `json:"task_id"`
	TaskRef       string           `json:"task_ref"`
	TotalSizeBytes int64           `json:"total_size_bytes"`
	Data          []AttachmentInfo `json:"data"`
	Pagination    Pagination       `json:"pagination"`
}

type FailedFile struct {
	FileName string `json:"file_name"`
	Reason   string `json:"reason"`
	Message  string `json:"message"`
}

type UploadedFileInfo struct {
	ID           string  `json:"id"`
	FileName     string  `json:"file_name"`
	FileType     string  `json:"file_type"`
	FileGroup    string  `json:"file_group"`
	SizeBytes    int64   `json:"size_bytes"`
	SizeDisplay  string  `json:"size_display"`
	ThumbnailURL *string `json:"thumbnail_url"`
	Uploader     AttachmentUploader `json:"uploader"`
	CreatedAt    string  `json:"created_at"`
}

type UploadAttachmentsResponse struct {
	TaskID        string             `json:"task_id"`
	TaskRef       string             `json:"task_ref"`
	Uploaded      []UploadedFileInfo `json:"uploaded"`
	Failed        []FailedFile       `json:"failed"`
	TotalUploaded int                `json:"total_uploaded"`
}

type DownloadUrlResponse struct {
	AttachmentID    string `json:"attachment_id"`
	FileName        string `json:"file_name"`
	DownloadURL     string `json:"download_url"`
	ExpiresInSeconds int   `json:"expires_in_seconds"`
	ExpiresAt       string `json:"expires_at"`
	Disposition     string `json:"disposition"`
}

type PreviewUrlResponse struct {
	AttachmentID    string  `json:"attachment_id"`
	FileName        string  `json:"file_name"`
	FileType        string  `json:"file_type"`
	PreviewURL      string  `json:"preview_url"`
	ThumbnailURL    string  `json:"thumbnail_url"`
	IsPreviewable   bool    `json:"is_previewable"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

type DeleteAttachmentResponse struct {
	Message                   string `json:"message"`
	DeletedAttachmentID       string `json:"deleted_attachment_id"`
	ScheduledPermanentDeleteAt string `json:"scheduled_permanent_delete_at"`
}
