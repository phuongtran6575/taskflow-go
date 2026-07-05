package dto

import "time"

type BoardProjectInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Key        string `json:"key"`
	IsArchived bool   `json:"is_archived"`
}

type BoardFilters struct {
	Priority    []string `json:"priority"`
	AssigneeID  []string `json:"assignee_id"`
	LabelID     []string `json:"label_id"`
	DueDateFrom *string  `json:"due_date_from"`
	DueDateTo   *string  `json:"due_date_to"`
	CreatorID   *string  `json:"creator_id"`
	HasAssignee *bool    `json:"has_assignee"`
	Overdue     *bool    `json:"overdue"`
	Search      *string  `json:"search"`
}

type BoardTaskInfo struct {
	ID              string         `json:"id"`
	TaskNumber      int            `json:"task_number"`
	TaskRef         string         `json:"task_ref"`
	Title           string         `json:"title"`
	Priority        string         `json:"priority"`
	DueDate         *string        `json:"due_date"`
	IsOverdue       bool           `json:"is_overdue"`
	Position        float64        `json:"position"`
	Assignees       []AssigneeInfo `json:"assignees"`
	Labels          []LabelRef     `json:"labels"`
	SubtaskCount    int            `json:"subtask_count"`
	SubtaskDoneCount int           `json:"subtask_done_count"`
	CommentCount    int            `json:"comment_count"`
	AttachmentCount int            `json:"attachment_count"`
}

type BoardColumn struct {
	ID            string          `json:"id"`
	Title         string          `json:"title"`
	Position      float64         `json:"position"`
	TaskTotal     int             `json:"task_total"`
	TaskFiltered  int             `json:"task_filtered"`
	Tasks         []BoardTaskInfo `json:"tasks"`
	HasMore       bool            `json:"has_more"`
	NextCursor    *string         `json:"next_cursor"`
}

type BoardResponse struct {
	Project        BoardProjectInfo `json:"project"`
	FiltersApplied BoardFilters     `json:"filters_applied"`
	Columns        []BoardColumn    `json:"columns"`
}

type LoadMoreTasksResponse struct {
	ColumnID  string          `json:"column_id"`
	Tasks     []BoardTaskInfo `json:"tasks"`
	HasMore   bool            `json:"has_more"`
	NextCursor *string        `json:"next_cursor"`
}

type MoveTaskRequest struct {
	ColumnID           string     `json:"column_id" binding:"required"`
	PreviousPosition   *float64   `json:"previous_position"`
	NextPosition       *float64   `json:"next_position"`
	LastKnownUpdatedAt *time.Time `json:"last_known_updated_at"`
}

type TaskPositionInfo struct {
	ID       string  `json:"id"`
	Position float64 `json:"position"`
}

type MoveTaskResponse struct {
	ID                string              `json:"id"`
	TaskRef           string              `json:"task_ref"`
	ColumnID          string              `json:"column_id"`
	Position          float64             `json:"position"`
	PreviousColumnID  string              `json:"previous_column_id"`
	MovedBetweenColumns bool              `json:"moved_between_columns"`
	Rebalanced        bool                `json:"rebalanced"`
	AllPositions      *[]TaskPositionInfo `json:"all_positions,omitempty"`
	UpdatedAt         string              `json:"updated_at"`
}
