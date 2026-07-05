package models

import (
	"time"
)

type ActivityAction string

const (
	ActivityActionCREATE ActivityAction = "CREATE"
	ActivityActionUPDATE ActivityAction = "UPDATE"
	ActivityActionDELETE ActivityAction = "DELETE"
)

type EntityType string

const (
	EntityTypeTASK      EntityType = "TASK"
	EntityTypePROJECT   EntityType = "PROJECT"
	EntityTypeLABEL     EntityType = "LABEL"
	EntityTypeCOMMENT   EntityType = "COMMENT"
	EntityTypeCOLUMN    EntityType = "COLUMN"
	EntityTypeWORKSPACE EntityType = "WORKSPACE"
	EntityTypeROLE      EntityType = "ROLE"
)

type ActivityLog struct {
	ID          string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	WorkspaceID *string        `gorm:"type:uuid"`
	ProjectID   *string        `gorm:"type:uuid"`
	UserID      *string        `gorm:"type:uuid"`
	Action      ActivityAction `gorm:"type:varchar(10);not null"`
	EntityType  EntityType     `gorm:"type:varchar(20);not null"`
	EntityID    string         `gorm:"type:uuid;not null"`
	Metadata    *string        `gorm:"type:jsonb"`
	CreatedAt   time.Time      `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`

	Workspace *Workspace `gorm:"foreignKey:WorkspaceID;constraint:OnDelete:SET NULL"`
	Project   *Project   `gorm:"foreignKey:ProjectID;constraint:OnDelete:SET NULL"`
	User      *User      `gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL"`
}

func (ActivityLog) TableName() string { return "activity_logs" }
