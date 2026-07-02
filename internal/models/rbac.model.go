package models

import (
	"time"

	"gorm.io/gorm"
)

type Permission struct {
	ID          string  `gorm:"type:varchar(100);primaryKey"`
	Slug        string  `gorm:"type:varchar(50);not null"`
	Module      string  `gorm:"type:varchar(50);not null"`
	Description *string `gorm:"type:text"`
	IsSystem    bool    `gorm:"not null;default:true"`
}

func (Permission) TableName() string { return "permissions" }

type Role struct {
	ID          string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	WorkspaceID string    `gorm:"type:uuid;not null"`
	Name        string    `gorm:"type:varchar(50);not null"`
	Description *string   `gorm:"type:text"`
	UpdatedAt   time.Time `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	DeletedAt   gorm.DeletedAt

	Workspace   Workspace        `gorm:"foreignKey:WorkspaceID;constraint:OnDelete:CASCADE"`
	Permissions []RolePermission `gorm:"foreignKey:RoleID;constraint:OnDelete:CASCADE"`
}

func (Role) TableName() string { return "roles" }

type RolePermission struct {
	RoleID       string `gorm:"type:uuid;primaryKey"`
	PermissionID string `gorm:"type:varchar(100);primaryKey"`

	Role       Role       `gorm:"foreignKey:RoleID;constraint:OnDelete:CASCADE"`
	Permission Permission `gorm:"foreignKey:PermissionID;constraint:OnDelete:CASCADE"`
}

func (RolePermission) TableName() string { return "role_permissions" }
