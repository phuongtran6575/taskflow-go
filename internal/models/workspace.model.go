package models

import (
	"time"

	"gorm.io/gorm"
)

type WorkspacePlan string

const (
	WorkspacePlanFREE       WorkspacePlan = "FREE"
	WorkspacePlanPRO        WorkspacePlan = "PRO"
	WorkspacePlanENTERPRISE WorkspacePlan = "ENTERPRISE"
)

type WorkspaceRole string

const (
	WorkspaceRoleOWNER  WorkspaceRole = "OWNER"
	WorkspaceRoleADMIN  WorkspaceRole = "ADMIN"
	WorkspaceRoleMEMBER WorkspaceRole = "MEMBER"
)

type Workspace struct {
	ID        string       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OwnerID   string       `gorm:"type:uuid;not null"`
	Name      string       `gorm:"type:varchar(100);not null"`
	Domain    *string      `gorm:"type:varchar(100);unique"`
	Plan      WorkspacePlan `gorm:"type:varchar(20);not null;default:'FREE'"`
	CreatedAt time.Time    `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time    `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	DeletedAt gorm.DeletedAt

	Owner     User              `gorm:"foreignKey:OwnerID;constraint:OnDelete:CASCADE"`
	Members   []WorkspaceMember `gorm:"foreignKey:WorkspaceID;constraint:OnDelete:CASCADE"`
	Invites   []WorkspaceInvite `gorm:"foreignKey:WorkspaceID;constraint:OnDelete:CASCADE"`
	Projects  []Project         `gorm:"foreignKey:WorkspaceID;constraint:OnDelete:CASCADE"`
	Roles     []Role            `gorm:"foreignKey:WorkspaceID;constraint:OnDelete:CASCADE"`
}

func (Workspace) TableName() string { return "workspaces" }

type WorkspaceMember struct {
	WorkspaceID  string       `gorm:"type:uuid;primaryKey"`
	UserID       string       `gorm:"type:uuid;primaryKey"`
	Role         WorkspaceRole `gorm:"type:varchar(20);not null;default:'MEMBER'"`
	JoinedAt     time.Time    `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt    time.Time    `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	UpdatedByID  *string      `gorm:"type:uuid"`

	Workspace Workspace `gorm:"foreignKey:WorkspaceID;constraint:OnDelete:CASCADE"`
	User      User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	UpdatedBy *User     `gorm:"foreignKey:UpdatedByID;constraint:OnDelete:SET NULL"`
}

func (WorkspaceMember) TableName() string { return "workspace_members" }

type WorkspaceInvite struct {
	ID          string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	WorkspaceID string     `gorm:"type:uuid;not null"`
	Code        string     `gorm:"type:varchar(50);not null"`
	Role        string     `gorm:"type:varchar(20);not null"`
	MaxUses     *int       `gorm:"type:int"`
	UsesCount   int        `gorm:"type:int;not null;default:0"`
	CreatedBy   *string    `gorm:"type:uuid"`
	ExpiresAt   *time.Time `gorm:"type:timestamp"`
	CreatedAt   time.Time  `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	DeletedAt   gorm.DeletedAt

	Workspace Workspace `gorm:"foreignKey:WorkspaceID;constraint:OnDelete:CASCADE"`
	Creator   *User     `gorm:"foreignKey:CreatedBy;constraint:OnDelete:SET NULL"`
}

func (WorkspaceInvite) TableName() string { return "workspace_invites" }
