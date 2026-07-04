package dto

import (
	"TaskFlow-Go/internal/models"
	"time"
)

type WorkspacePlanCount struct {
	Plan  models.WorkspacePlan `gorm:"column:plan"`
	Count int64                `gorm:"column:count"`
}

type CreateWorkspaceRequest struct {
	Name   string  `json:"name" binding:"required"`
	Domain *string `json:"domain,omitempty"`
}

type UpdateWorkspaceRequest struct {
	Name   *string `json:"name,omitempty" binding:"omitempty,min=1"`
	Domain *string `json:"domain,omitempty"`
}

type UpgradePlanRequest struct {
	NewPlan string `json:"new_plan" binding:"required,oneof=FREE PRO ENTERPRISE"`
}

type DeleteWorkspaceRequest struct {
	ConfirmationName string `json:"confirmation_name" binding:"required"`
}

type WorkspaceSummary struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Domain       *string   `json:"domain"`
	Plan         string    `json:"plan"`
	MyRole       string    `json:"my_role"`
	MemberCount  int       `json:"member_count"`
	ProjectCount int       `json:"project_count"`
	JoinedAt     time.Time `json:"joined_at"`
}

type WorkspaceListResponse struct {
	Data  []WorkspaceSummary `json:"data"`
	Total int                `json:"total"`
}

type WorkspaceOwnerInfo struct {
	ID        string  `json:"id"`
	FullName  string  `json:"full_name"`
	AvatarURL *string `json:"avatar_url"`
}

type WorkspaceDetailResponse struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	Domain       *string            `json:"domain"`
	Plan         string             `json:"plan"`
	Owner        WorkspaceOwnerInfo `json:"owner"`
	MemberCount  int                `json:"member_count"`
	ProjectCount int                `json:"project_count"`
	IsOverLimit  bool               `json:"is_over_limit"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

type WorkspaceCreateResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Domain    *string   `json:"domain"`
	Plan      string    `json:"plan"`
	OwnerID   string    `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
}

type UpdateWorkspaceResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Domain    *string   `json:"domain"`
	Plan      string    `json:"plan"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PlanUpgradeResponse struct {
	ID           string    `json:"id"`
	PreviousPlan string    `json:"previous_plan"`
	CurrentPlan  string    `json:"current_plan"`
	UpgradedAt   time.Time `json:"upgraded_at"`
}

type DeleteWorkspaceResponse struct {
	Message string `json:"message"`
}
