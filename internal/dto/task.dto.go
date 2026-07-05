package dto

import "time"

type CreateTaskRequest struct {
	Title       string     `json:"title" binding:"required"`
	ColumnID    string     `json:"column_id" binding:"required"`
	Description *string    `json:"description,omitempty"`
	Priority    string     `json:"priority,omitempty"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	AssigneeIDs []string   `json:"assignee_ids,omitempty"`
	LabelIDs    []string   `json:"label_ids,omitempty"`
}

type UpdateTaskRequest struct {
	Title       *string    `json:"title,omitempty" binding:"omitempty,min=1"`
	Description *string    `json:"description,omitempty"`
	Priority    *string    `json:"priority,omitempty"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
}

type CreateSubtaskRequest struct {
	Title       string     `json:"title" binding:"required"`
	Priority    string     `json:"priority,omitempty"`
	AssigneeIDs []string   `json:"assignee_ids,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	ColumnID    *string    `json:"column_id,omitempty"`
}

type ColumnRef struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type ProjectRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Key  string `json:"key"`
}

type AssigneeInfo struct {
	UserID     string  `json:"user_id"`
	FullName   string  `json:"full_name"`
	AvatarURL  *string `json:"avatar_url"`
	AssignedAt *string `json:"assigned_at,omitempty"`
	AssignedBy *struct {
		UserID   string `json:"user_id"`
		FullName string `json:"full_name"`
	} `json:"assigned_by,omitempty"`
}

type LabelRef struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type TaskParentRef struct {
	ID         string `json:"id"`
	TaskNumber int    `json:"task_number"`
	TaskRef    string `json:"task_ref"`
	Title      string `json:"title"`
}

type CreatorInfo struct {
	UserID    string  `json:"user_id"`
	FullName  string  `json:"full_name"`
	AvatarURL *string `json:"avatar_url"`
}

type TaskSummary struct {
	ID              string         `json:"id"`
	TaskNumber      int            `json:"task_number"`
	TaskRef         string         `json:"task_ref"`
	Title           string         `json:"title"`
	Priority        string         `json:"priority"`
	DueDate         *time.Time     `json:"due_date"`
	IsOverdue       bool           `json:"is_overdue"`
	Column          ColumnRef      `json:"column"`
	Assignees       []AssigneeInfo `json:"assignees"`
	Labels          []LabelRef     `json:"labels"`
	SubtaskCount    int            `json:"subtask_count"`
	CommentCount    int            `json:"comment_count"`
	AttachmentCount int            `json:"attachment_count"`
	Position        float64        `json:"position"`
	CreatedAt       time.Time      `json:"created_at"`
}

type TaskDetailResponse struct {
	ID               string         `json:"id"`
	TaskNumber       int            `json:"task_number"`
	TaskRef          string         `json:"task_ref"`
	Title            string         `json:"title"`
	Description      *string        `json:"description"`
	Priority         string         `json:"priority"`
	StartDate        *time.Time     `json:"start_date"`
	DueDate          *time.Time     `json:"due_date"`
	Column           ColumnRef      `json:"column"`
	Project          ProjectRef     `json:"project"`
	Creator          *CreatorInfo   `json:"creator"`
	Assignees        []AssigneeInfo `json:"assignees"`
	Labels           []LabelRef     `json:"labels"`
	Parent           *TaskParentRef `json:"parent"`
	SubtaskCount     int            `json:"subtask_count"`
	SubtaskDoneCount int            `json:"subtask_done_count"`
	CommentCount     int            `json:"comment_count"`
	AttachmentCount  int            `json:"attachment_count"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

type TaskCreateResponse struct {
	ID          string         `json:"id"`
	TaskNumber  int            `json:"task_number"`
	TaskRef     string         `json:"task_ref"`
	Title       string         `json:"title"`
	Description *string        `json:"description"`
	Priority    string         `json:"priority"`
	StartDate   *time.Time     `json:"start_date"`
	DueDate     *time.Time     `json:"due_date"`
	Column      ColumnRef      `json:"column"`
	Creator     CreatorInfo    `json:"creator"`
	Assignees   []AssigneeInfo `json:"assignees"`
	Labels      []LabelRef     `json:"labels"`
	ParentID    *string        `json:"parent_id"`
	Position    float64        `json:"position"`
	CreatedAt   time.Time      `json:"created_at"`
}

type FieldChange struct {
	Field    string      `json:"field"`
	OldValue interface{} `json:"old_value"`
	NewValue interface{} `json:"new_value"`
}

type UpdateTaskResponse struct {
	ID          string        `json:"id"`
	TaskNumber  int           `json:"task_number"`
	TaskRef     string        `json:"task_ref"`
	Title       string        `json:"title"`
	Description *string       `json:"description"`
	Priority    string        `json:"priority"`
	StartDate   *time.Time    `json:"start_date"`
	DueDate     *time.Time    `json:"due_date"`
	UpdatedAt   time.Time     `json:"updated_at"`
	Changes     []FieldChange `json:"changes"`
}

type TaskDeleteResponse struct {
	Message              string `json:"message"`
	DeletedTaskID        string `json:"deleted_task_id"`
	DeletedSubtasksCount int    `json:"deleted_subtasks_count"`
}

type SubtaskInfo struct {
	ID           string         `json:"id"`
	TaskNumber   int            `json:"task_number"`
	TaskRef      string         `json:"task_ref"`
	Title        string         `json:"title"`
	Priority     string         `json:"priority"`
	DueDate      *time.Time     `json:"due_date"`
	Column       ColumnRef      `json:"column"`
	Assignees    []AssigneeInfo `json:"assignees"`
	CommentCount int            `json:"comment_count"`
	Position     float64        `json:"position"`
}

type SubtaskProgress struct {
	Total      int `json:"total"`
	Done       int `json:"done"`
	Percentage int `json:"percentage"`
}

type SubtaskListResponse struct {
	Parent   TaskParentRef   `json:"parent"`
	Progress SubtaskProgress `json:"progress"`
	Data     []SubtaskInfo   `json:"data"`
}

type SubtaskCreateResponse struct {
	ID         string         `json:"id"`
	TaskNumber int            `json:"task_number"`
	TaskRef    string         `json:"task_ref"`
	Title      string         `json:"title"`
	Priority   string         `json:"priority"`
	DueDate    *time.Time     `json:"due_date"`
	Column     ColumnRef      `json:"column"`
	Parent     TaskParentRef  `json:"parent"`
	Assignees  []AssigneeInfo `json:"assignees"`
	Position   float64        `json:"position"`
	CreatedAt  time.Time      `json:"created_at"`
}

type MyTaskProjectRef struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	Key  string  `json:"key"`
	Icon *string `json:"icon"`
}

type MyTaskInfo struct {
	ID        string           `json:"id"`
	TaskRef   string           `json:"task_ref"`
	Title     string           `json:"title"`
	Priority  string           `json:"priority"`
	DueDate   *time.Time       `json:"due_date"`
	IsOverdue bool             `json:"is_overdue"`
	Column    ColumnRef        `json:"column"`
	Project   MyTaskProjectRef `json:"project"`
	Parent    *struct {
		TaskRef string `json:"task_ref"`
		Title   string `json:"title"`
	} `json:"parent"`
}

type MyTaskSummary struct {
	Total       int `json:"total"`
	Overdue     int `json:"overdue"`
	DueToday    int `json:"due_today"`
	DueThisWeek int `json:"due_this_week"`
	NoDueDate   int `json:"no_due_date"`
}

type MyTaskListResponse struct {
	Data       []MyTaskInfo  `json:"data"`
	Pagination Pagination    `json:"pagination"`
	Summary    *MyTaskSummary `json:"summary,omitempty"`
}

type TaskSearchResponse struct {
	Data           []TaskSummary          `json:"data"`
	Pagination     Pagination             `json:"pagination"`
	FiltersApplied map[string]interface{} `json:"filters_applied"`
}
