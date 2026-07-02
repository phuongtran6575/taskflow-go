package models

import (
	"time"

	"gorm.io/gorm"
)

type Project struct {
	ID           string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	WorkspaceID  string         `gorm:"type:uuid;not null"`
	OwnerID      *string        `gorm:"type:uuid"`
	Name         string         `gorm:"type:varchar(100);not null"`
	Key          string         `gorm:"type:varchar(10);not null"`
	Icon         *string        `gorm:"type:varchar(50)"`
	Background   string         `gorm:"type:varchar(255);not null;default:#ffffff"`
	IsArchived   bool           `gorm:"not null;default:false"`
	ArchivedAt   *time.Time     `gorm:"type:timestamp"`
	ArchivedByID *string        `gorm:"type:uuid"`
	CreatedAt    time.Time      `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt    time.Time      `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	DeletedAt    gorm.DeletedAt

	Workspace Workspace        `gorm:"foreignKey:WorkspaceID;constraint:OnDelete:CASCADE"`
	Owner     *User            `gorm:"foreignKey:OwnerID;constraint:OnDelete:SET NULL"`
	ArchivedBy *User           `gorm:"foreignKey:ArchivedByID;constraint:OnDelete:SET NULL"`
	Members   []ProjectMember  `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
	Columns   []Column         `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
	Tasks     []Task           `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
	Labels    []Label          `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
}

func (Project) TableName() string { return "projects" }

type ProjectMember struct {
	ProjectID  string    `gorm:"type:uuid;primaryKey"`
	UserID     string    `gorm:"type:uuid;primaryKey"`
	RoleID     *string   `gorm:"type:uuid"`
	IsFavorite bool      `gorm:"not null;default:false"`
	JoinedAt   time.Time `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`

	Project Project `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
	User    User    `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Role    *Role   `gorm:"foreignKey:RoleID;constraint:OnDelete:SET NULL"`
}

func (ProjectMember) TableName() string { return "project_members" }

type Column struct {
	ID        string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ProjectID string         `gorm:"type:uuid;not null"`
	Title     string         `gorm:"type:varchar(50);not null"`
	Position  float64        `gorm:"type:float;not null"`
	IsDone    bool           `gorm:"not null;default:false"`
	CreatedAt time.Time      `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time      `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	DeletedAt gorm.DeletedAt

	Project Project `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
	Tasks   []Task  `gorm:"foreignKey:ColumnID;constraint:OnDelete:CASCADE"`
}

func (Column) TableName() string { return "columns" }
