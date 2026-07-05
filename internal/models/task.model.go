package models

import (
	"time"

	"gorm.io/gorm"
)

type TaskPriority string

const (
	TaskPriorityLOW    TaskPriority = "LOW"
	TaskPriorityMED    TaskPriority = "MED"
	TaskPriorityHIGH   TaskPriority = "HIGH"
	TaskPriorityURGENT TaskPriority = "URGENT"
)

type Task struct {
	ID          string       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ColumnID    string       `gorm:"type:uuid;not null"`
	ProjectID   string       `gorm:"type:uuid;not null"`
	CreatorID   *string      `gorm:"type:uuid"`
	ParentID    *string      `gorm:"type:uuid"`
	TaskNumber  int          `gorm:"type:int;not null"`
	Title       string       `gorm:"type:varchar(255);not null"`
	Description *string      `gorm:"type:text"`
	Priority    TaskPriority `gorm:"type:varchar(10);not null;default:'MED'"`
	StartDate   *time.Time   `gorm:"type:timestamp"`
	DueDate     *time.Time   `gorm:"type:timestamp"`
	Position    float64      `gorm:"type:float;not null"`
	CreatedAt   time.Time    `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt   time.Time    `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	DeletedAt   gorm.DeletedAt

	Column      Column           `gorm:"foreignKey:ColumnID;constraint:OnDelete:CASCADE"`
	Project     Project          `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
	Creator     User             `gorm:"foreignKey:CreatorID;constraint:OnDelete:SET NULL"`
	Parent      *Task            `gorm:"foreignKey:ParentID;constraint:OnDelete:CASCADE"`
	Assignees   []TaskAssignee   `gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE"`
	Attachments []Attachment     `gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE"`
	Comments    []Comment        `gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE"`
	TaskLabels  []TaskLabel      `gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE"`
	Subtasks    []Task           `gorm:"foreignKey:ParentID;constraint:OnDelete:CASCADE"`
}

func (Task) TableName() string { return "tasks" }

type TaskAssignee struct {
	TaskID       string    `gorm:"type:uuid;primaryKey"`
	UserID       string    `gorm:"type:uuid;primaryKey"`
	AssignedAt   time.Time `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	AssignedByID *string   `gorm:"type:uuid"`

	Task       Task  `gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE"`
	User       User  `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	AssignedBy *User `gorm:"foreignKey:AssignedByID;constraint:OnDelete:SET NULL"`
}

func (TaskAssignee) TableName() string { return "task_assignees" }

type Attachment struct {
	ID                string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TaskID            string         `gorm:"type:uuid;not null"`
	UploaderID        *string        `gorm:"type:uuid"`
	FileName          string         `gorm:"type:varchar(255);not null"`
	FileURL           string         `gorm:"type:text;not null"`
	FileType          string         `gorm:"type:varchar(50);not null"`
	SizeBytes         int64          `gorm:"type:bigint;not null"`
	CreatedAt         time.Time      `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	DeletedAt         gorm.DeletedAt
	ScheduledDeleteAt *time.Time    `gorm:"type:timestamp"`

	Task     Task  `gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE"`
	Uploader *User `gorm:"foreignKey:UploaderID;constraint:OnDelete:SET NULL"`
}

func (Attachment) TableName() string { return "attachments" }

type Comment struct {
	ID        string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TaskID    string         `gorm:"type:uuid;not null"`
	UserID    *string        `gorm:"type:uuid"`
	Content   string         `gorm:"type:text;not null"`
	CreatedAt time.Time      `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time      `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	DeletedAt gorm.DeletedAt

	Task Task  `gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE"`
	User *User `gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL"`
}

func (Comment) TableName() string { return "comments" }

type Label struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ProjectID string    `gorm:"type:uuid;not null"`
	Name      string    `gorm:"type:varchar(50);not null"`
	Color     string    `gorm:"type:varchar(7);not null"`
	CreatedAt time.Time `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`

	Project    Project    `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
	TaskLabels []TaskLabel `gorm:"foreignKey:LabelID;constraint:OnDelete:CASCADE"`
}

func (Label) TableName() string { return "labels" }

type TaskLabel struct {
	TaskID  string `gorm:"type:uuid;primaryKey"`
	LabelID string `gorm:"type:uuid;primaryKey"`

	Task  Task  `gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE"`
	Label Label `gorm:"foreignKey:LabelID;constraint:OnDelete:CASCADE"`
}

func (TaskLabel) TableName() string { return "task_labels" }

type CommentMention struct {
	CommentID string `gorm:"type:uuid;primaryKey"`
	UserID    string `gorm:"type:uuid;primaryKey"`

	Comment Comment `gorm:"foreignKey:CommentID;constraint:OnDelete:CASCADE"`
	User    User    `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

func (CommentMention) TableName() string { return "comment_mentions" }
