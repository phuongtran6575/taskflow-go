package projection

import "time"

type TaskSummaryRow struct {
	ID              string     `gorm:"column:id"`
	TaskNumber      int        `gorm:"column:task_number"`
	TaskRef         string     `gorm:"column:task_ref"`
	Title           string     `gorm:"column:title"`
	Priority        string     `gorm:"column:priority"`
	DueDate         *time.Time `gorm:"column:due_date"`
	ColumnID        string     `gorm:"column:column_id"`
	ColumnTitle     string     `gorm:"column:column_title"`
	SubtaskCount    int        `gorm:"column:subtask_count"`
	CommentCount    int        `gorm:"column:comment_count"`
	AttachmentCount int        `gorm:"column:attachment_count"`
	Position        float64    `gorm:"column:position"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	ProjectID       string     `gorm:"column:project_id"`
	ProjectName     string     `gorm:"column:project_name"`
	ProjectKey      string     `gorm:"column:project_key"`
	ProjectIcon     *string    `gorm:"column:project_icon"`
}

type SubtaskCountRow struct {
	SubtaskCount int `gorm:"column:subtask_count"`
	DoneCount    int `gorm:"column:done_count"`
}

type TaskDetailRow struct {
	ID          string  `gorm:"column:id"`
	TaskNumber  int     `gorm:"column:task_number"`
	TaskRef     string  `gorm:"column:task_ref"`
	Title       string  `gorm:"column:title"`
	Description *string `gorm:"column:description"`
	Priority    string  `gorm:"column:priority"`
	StartDate   *time.Time `gorm:"column:start_date"`
	DueDate     *time.Time `gorm:"column:due_date"`
	Position    float64    `gorm:"column:position"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at"`

	ColumnID    string `gorm:"column:column_id"`
	ColumnTitle string `gorm:"column:column_title"`

	ProjectID   string `gorm:"column:project_id"`
	ProjectName string `gorm:"column:project_name"`
	ProjectKey  string `gorm:"column:project_key"`

	CreatorID     *string `gorm:"column:creator_id"`
	CreatorName   *string `gorm:"column:creator_name"`
	CreatorAvatar *string `gorm:"column:creator_avatar"`

	ParentID         *string `gorm:"column:parent_id"`
	ParentTaskNumber *int    `gorm:"column:parent_task_number"`
	ParentTaskRef    *string `gorm:"column:parent_task_ref"`
	ParentTitle      *string `gorm:"column:parent_title"`

	SubtaskCount     int `gorm:"column:subtask_count"`
	SubtaskDoneCount int `gorm:"column:subtask_done_count"`
	CommentCount     int `gorm:"column:comment_count"`
	AttachmentCount  int `gorm:"column:attachment_count"`
}

type SubtaskDetailRow struct {
	ID          string     `gorm:"column:id"`
	TaskNumber  int        `gorm:"column:task_number"`
	TaskRef     string     `gorm:"column:task_ref"`
	Title       string     `gorm:"column:title"`
	Priority    string     `gorm:"column:priority"`
	DueDate     *time.Time `gorm:"column:due_date"`
	ColumnID    string     `gorm:"column:column_id"`
	ColumnTitle string     `gorm:"column:column_title"`
	Position    float64    `gorm:"column:position"`
	CommentCount int       `gorm:"column:comment_count"`
}

type BoardTaskRow struct {
	ID               string     `gorm:"column:id"`
	TaskNumber       int        `gorm:"column:task_number"`
	TaskRef          string     `gorm:"column:task_ref"`
	Title            string     `gorm:"column:title"`
	Priority         string     `gorm:"column:priority"`
	DueDate          *time.Time `gorm:"column:due_date"`
	Position         float64    `gorm:"column:position"`
	ColumnID         string     `gorm:"column:column_id"`
	ColumnTitle      string     `gorm:"column:column_title"`
	SubtaskCount     int        `gorm:"column:subtask_count"`
	SubtaskDoneCount int        `gorm:"column:subtask_done_count"`
	CommentCount     int        `gorm:"column:comment_count"`
	AttachmentCount  int        `gorm:"column:attachment_count"`
}

type MyTaskRow struct {
	ID          string     `gorm:"column:id"`
	TaskNumber  int        `gorm:"column:task_number"`
	TaskRef     string     `gorm:"column:task_ref"`
	Title       string     `gorm:"column:title"`
	Priority    string     `gorm:"column:priority"`
	DueDate     *time.Time `gorm:"column:due_date"`
	ColumnID    string     `gorm:"column:column_id"`
	ColumnTitle string     `gorm:"column:column_title"`
	ProjectID   string     `gorm:"column:project_id"`
	ProjectName string     `gorm:"column:project_name"`
	ProjectKey  string     `gorm:"column:project_key"`
	ProjectIcon *string    `gorm:"column:project_icon"`
	ParentID    *string    `gorm:"column:parent_id"`
	ParentRef   *string    `gorm:"column:parent_ref"`
	ParentTitle *string    `gorm:"column:parent_title"`
}
