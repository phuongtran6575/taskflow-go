package dto

import "time"

type CreateProjectRequest struct {
	Name       string  `json:"name" binding:"required"`
	Key        *string `json:"key,omitempty"`
	Icon       *string `json:"icon,omitempty"`
	Background *string `json:"background,omitempty"`
}

type UpdateProjectRequest struct {
	Name       *string `json:"name,omitempty" binding:"omitempty,min=1"`
	Icon       *string `json:"icon,omitempty"`
	Background *string `json:"background,omitempty"`
}

type DeleteProjectRequest struct {
	ConfirmationName string `json:"confirmation_name" binding:"required"`
}

type ProjectOwnerInfo struct {
	UserID   string  `json:"user_id"`
	FullName string  `json:"full_name"`
	AvatarURL *string `json:"avatar_url"`
}

type DefaultColumn struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Position float64 `json:"position"`
	IsDone   bool    `json:"is_done"`
}

type ProjectSummary struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Key          string           `json:"key"`
	Icon         *string          `json:"icon"`
	Background   string           `json:"background"`
	IsArchived   bool             `json:"is_archived"`
	IsFavorite   bool             `json:"is_favorite"`
	Owner        ProjectOwnerInfo `json:"owner"`
	MemberCount  int              `json:"member_count"`
	OpenTaskCount int             `json:"open_task_count"`
	MyRole       *RoleRef         `json:"my_role"`
	JoinedAt     time.Time        `json:"joined_at"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

type RoleRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ProjectCreateResponse struct {
	ID             string           `json:"id"`
	Name           string           `json:"name"`
	Key            string           `json:"key"`
	Icon           *string          `json:"icon"`
	Background     string           `json:"background"`
	IsArchived     bool             `json:"is_archived"`
	Owner          ProjectOwnerInfo `json:"owner"`
	DefaultColumns []DefaultColumn  `json:"default_columns"`
	CreatedAt      time.Time        `json:"created_at"`
}

type ColumnTaskSummary struct {
	ColumnID  string `json:"column_id"`
	Title     string `json:"column_title"`
	TaskCount int    `json:"task_count"`
}

type ProjectTaskSummary struct {
	Total    int                 `json:"total"`
	ByColumn []ColumnTaskSummary `json:"by_column"`
}

type MyRolePermissions struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

type ProjectDetailResponse struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Key         string              `json:"key"`
	Icon        *string             `json:"icon"`
	Background  string              `json:"background"`
	IsArchived  bool                `json:"is_archived"`
	IsFavorite  bool                `json:"is_favorite"`
	Owner       ProjectOwnerInfo    `json:"owner"`
	MyRole      MyRolePermissions   `json:"my_role"`
	MemberCount int                 `json:"member_count"`
	TaskSummary ProjectTaskSummary  `json:"task_summary"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

type UpdateProjectResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Key       string    `json:"key"`
	Icon      *string   `json:"icon"`
	Background string   `json:"background"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ArchiveProjectResponse struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	IsArchived bool      `json:"is_archived"`
	ArchivedAt time.Time `json:"archived_at"`
	ArchivedBy string    `json:"archived_by"`
}

type UnarchiveProjectResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	IsArchived   bool      `json:"is_archived"`
	UnarchivedAt time.Time `json:"unarchived_at"`
	UnarchivedBy string   `json:"unarchived_by"`
}

type FavoriteResponse struct {
	ProjectID  string `json:"project_id"`
	IsFavorite bool   `json:"is_favorite"`
}

type ProjectDeleteResponse struct {
	Message         string `json:"message"`
	DeletedProjectID string `json:"deleted_project_id"`
}
