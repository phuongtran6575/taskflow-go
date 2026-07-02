package dto

type CreateCommentRequest struct {
	Content string `json:"content" binding:"required"`
}

type UpdateCommentRequest struct {
	Content string `json:"content" binding:"required"`
}

type CommentAuthor struct {
	UserID   string  `json:"user_id"`
	FullName string  `json:"full_name"`
	Username string  `json:"username"`
	AvatarURL *string `json:"avatar_url"`
}

type MentionUser struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	FullName string `json:"full_name"`
}

type CommentInfo struct {
	ID          string        `json:"id"`
	Content     *string       `json:"content"`
	ContentHTML *string       `json:"content_html"`
	IsDeleted   bool          `json:"is_deleted"`
	Author      *CommentAuthor `json:"author"`
	Mentions    []MentionUser `json:"mentions"`
	IsEdited    bool          `json:"is_edited"`
	CreatedAt   string        `json:"created_at"`
	UpdatedAt   string        `json:"updated_at"`
}

type CommentListResponse struct {
	TaskID    string        `json:"task_id"`
	TaskRef   string        `json:"task_ref"`
	Total     int           `json:"total"`
	Data      []CommentInfo `json:"data"`
	HasMore   bool          `json:"has_more"`
	NextCursor *string      `json:"next_cursor"`
}

type CommentCreateResponse struct {
	ID          string           `json:"id"`
	Content     string           `json:"content"`
	ContentHTML string           `json:"content_html"`
	IsDeleted   bool             `json:"is_deleted"`
	Author      CommentAuthor    `json:"author"`
	Mentions    []MentionUser    `json:"mentions"`
	IsEdited    bool             `json:"is_edited"`
	CreatedAt   string           `json:"created_at"`
	UpdatedAt   string           `json:"updated_at"`
	NotificationsSent *NotificationsSent `json:"notifications_sent"`
}

type NotificationsSent struct {
	Mentioned []string `json:"mentioned"`
	Commented []string `json:"commented"`
}

type CommentUpdateResponse struct {
	ID                   string           `json:"id"`
	Content              string           `json:"content"`
	ContentHTML          string           `json:"content_html"`
	IsDeleted            bool             `json:"is_deleted"`
	Author               CommentAuthor    `json:"author"`
	Mentions             []MentionUser    `json:"mentions"`
	IsEdited             bool             `json:"is_edited"`
	CreatedAt            string           `json:"created_at"`
	UpdatedAt            string           `json:"updated_at"`
	NewMentionsNotified  []string         `json:"new_mentions_notified"`
}

type CommentDeleteResponse struct {
	Message          string `json:"message"`
	DeletedCommentID string `json:"deleted_comment_id"`
	DeletedByOwner   bool   `json:"deleted_by_owner"`
}

type MentionableUserInfo struct {
	UserID      string   `json:"user_id"`
	Username    string   `json:"username"`
	FullName    string   `json:"full_name"`
	AvatarURL   *string  `json:"avatar_url"`
	ProjectRole *RoleRef `json:"project_role"`
}

type MentionableUsersResponse struct {
	Data  []MentionableUserInfo `json:"data"`
	Total int                   `json:"total"`
}
