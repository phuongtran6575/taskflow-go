package dto

import "time"

type CreateColumnRequest struct {
	Title  string `json:"title" binding:"required"`
	IsDone *bool  `json:"is_done,omitempty"`
}

type UpdateColumnTitleRequest struct {
	Title string `json:"title" binding:"required"`
}

type UpdateColumnPositionRequest struct {
	PreviousPosition *float64 `json:"previous_position"`
	NextPosition     *float64 `json:"next_position"`
}

type DeleteColumnRequest struct {
	Strategy       *string `json:"strategy,omitempty"`
	TargetColumnID *string `json:"target_column_id,omitempty"`
}

type ColumnInfo struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Position  float64   `json:"position"`
	IsDone    bool      `json:"is_done"`
	TaskCount int       `json:"task_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ColumnListResponse struct {
	Data  []ColumnInfo `json:"data"`
	Total int          `json:"total"`
}

type ColumnCreateResponse struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Position  float64   `json:"position"`
	IsDone    bool      `json:"is_done"`
	TaskCount int       `json:"task_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UpdateColumnTitleResponse struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Position  float64   `json:"position"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UpdateColumnPositionResponse struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Position  float64   `json:"position"`
	UpdatedAt time.Time `json:"updated_at"`
	Rebalanced bool    `json:"rebalanced"`
}

type ColumnDeleteResponse struct {
	Message        string `json:"message"`
	DeletedColumnID string `json:"deleted_column_id"`
	TasksMoved     *int   `json:"tasks_moved,omitempty"`
	TasksDeleted   *int   `json:"tasks_deleted,omitempty"`
	TargetColumnID *string `json:"target_column_id,omitempty"`
}
