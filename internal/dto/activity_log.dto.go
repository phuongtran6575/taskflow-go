package dto

type ActivityLogActor struct {
	UserID    string  `json:"user_id"`
	FullName  string  `json:"full_name"`
	Username  string  `json:"username"`
	AvatarURL *string `json:"avatar_url"`
}

type ActivityLogWorkspaceRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ActivityLogProjectRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Key  string `json:"key"`
}

type EntitySnapshot map[string]interface{}

type ChangeField struct {
	Field    string `json:"field"`
	OldValue string `json:"old_value"`
	NewValue string `json:"new_value"`
}

type ActivityLogInfo struct {
	ID             string                 `json:"id"`
	Action         string                 `json:"action"`
	EntityType     string                 `json:"entity_type"`
	EntityID       string                 `json:"entity_id"`
	Description    string                 `json:"description"`
	Actor          ActivityLogActor       `json:"actor"`
	Workspace      *ActivityLogWorkspaceRef `json:"workspace,omitempty"`
	Project        *ActivityLogProjectRef `json:"project,omitempty"`
	EntitySnapshot *EntitySnapshot        `json:"entity_snapshot,omitempty"`
	Metadata       map[string]interface{} `json:"metadata"`
	CreatedAt      string                 `json:"created_at"`
}

type ActivityLogListResponse struct {
	Data       []ActivityLogInfo `json:"data"`
	HasMore    bool              `json:"has_more"`
	NextCursor *string           `json:"next_cursor"`
}

type TimelineActivityEntry struct {
	EntryType   string                 `json:"entry_type"`
	ID          string                 `json:"id"`
	Action      string                 `json:"action"`
	EntityType  string                 `json:"entity_type"`
	Description string                 `json:"description"`
	Actor       ActivityLogActor       `json:"actor"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   string                 `json:"created_at"`
}

type TimelineCommentEntry struct {
	EntryType   string            `json:"entry_type"`
	ID          string            `json:"id"`
	Content     *string           `json:"content"`
	ContentHTML *string           `json:"content_html,omitempty"`
	IsDeleted   bool              `json:"is_deleted"`
	IsEdited    bool              `json:"is_edited"`
	Author      *ActivityLogActor `json:"author"`
	Mentions    []MentionUser     `json:"mentions"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
}

type TaskTimelineResponse struct {
	TaskID     string        `json:"task_id"`
	TaskRef    string        `json:"task_ref"`
	Data       []interface{} `json:"data"`
	HasMore    bool          `json:"has_more"`
	NextCursor *string       `json:"next_cursor"`
}
