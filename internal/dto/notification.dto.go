package dto

type CreateAnnouncementRequest struct {
	Title       string   `json:"title" binding:"required"`
	Content     string   `json:"content" binding:"required"`
	TargetRoles []string `json:"target_roles,omitempty"`
}

type NotificationActor struct {
	UserID   string  `json:"user_id"`
	FullName string  `json:"full_name"`
	AvatarURL *string `json:"avatar_url"`
}

type NotificationInfo struct {
	ID           string            `json:"id"`
	Type         string            `json:"type"`
	Title        string            `json:"title"`
	Content      *string           `json:"content"`
	ReferenceURL *string           `json:"reference_url"`
	Actor        *NotificationActor `json:"actor"`
	IsRead       bool              `json:"is_read"`
	ReadAt       *string           `json:"read_at"`
	CreatedAt    string            `json:"created_at"`
}

type NotificationListResponse struct {
	Data        []NotificationInfo `json:"data"`
	HasMore     bool               `json:"has_more"`
	NextCursor  *string            `json:"next_cursor"`
	UnreadCount int                `json:"unread_count"`
}

type UnreadCountResponse struct {
	UnreadCount int            `json:"unread_count"`
	ByType      map[string]int `json:"by_type"`
}

type MarkAsReadResponse struct {
	NotificationID string  `json:"notification_id"`
	IsRead         bool    `json:"is_read"`
	ReadAt         string  `json:"read_at"`
}

type MarkAllAsReadRequest struct {
	Type *string `json:"type,omitempty"`
}

type MarkAllAsReadResponse struct {
	MarkedReadCount int `json:"marked_read_count"`
	UnreadCountAfter int `json:"unread_count_after"`
}

type CreateAnnouncementResponse struct {
	NotificationID string `json:"notification_id"`
	Title          string `json:"title"`
	Content        string `json:"content"`
	SentToCount    int    `json:"sent_to_count"`
	CreatedAt      string `json:"created_at"`
}
