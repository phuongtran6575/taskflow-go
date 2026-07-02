package dto

import "time"

type CreateRoleRequest struct {
	Name          string   `json:"name" binding:"required"`
	Description   *string  `json:"description,omitempty"`
	PermissionIDs []string `json:"permission_ids,omitempty"`
}

type UpdateRoleRequest struct {
	Name        *string `json:"name,omitempty" binding:"omitempty,min=1"`
	Description *string `json:"description,omitempty"`
}

type AssignPermissionsRequest struct {
	PermissionIDs []string `json:"permission_ids" binding:"required"`
}

type RemovePermissionsRequest struct {
	PermissionIDs []string `json:"permission_ids" binding:"required"`
}

type RoleSummary struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	PermissionCount int       `json:"permission_count"`
	MemberCount     int       `json:"member_count"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type MemberPreview struct {
	UserID    string  `json:"user_id"`
	FullName  string  `json:"full_name"`
	AvatarURL *string `json:"avatar_url"`
}

type RoleDetailResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Permissions map[string][]struct {
		ID          string `json:"id"`
		Slug        string `json:"slug"`
		Description string `json:"description"`
	} `json:"permissions"`
	PermissionCount int             `json:"permission_count"`
	MemberCount     int             `json:"member_count"`
	MembersPreview  []MemberPreview `json:"members_preview"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type RoleCreateResponse struct {
	ID              string                      `json:"id"`
	Name            string                      `json:"name"`
	Description     string                      `json:"description"`
	Permissions     map[string][]PermissionInfo `json:"permissions"`
	PermissionCount int                         `json:"permission_count"`
	MemberCount     int                         `json:"member_count"`
	UpdatedAt       time.Time                   `json:"updated_at"`
}

type RoleUpdateResponse struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	PermissionCount int       `json:"permission_count"`
	MemberCount     int       `json:"member_count"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type PermissionAssignedInfo struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
}

type AssignPermissionsResponse struct {
	RoleID               string                   `json:"role_id"`
	Added                []PermissionAssignedInfo `json:"added"`
	SkippedAlreadyExists []string                 `json:"skipped_already_exists"`
	PermissionCountAfter int                      `json:"permission_count_after"`
	UpdatedAt            time.Time                `json:"updated_at"`
}

type RemovePermissionsResponse struct {
	RoleID               string                   `json:"role_id"`
	Removed              []PermissionAssignedInfo `json:"removed"`
	SkippedNotExists     []string                 `json:"skipped_not_exists"`
	PermissionCountAfter int                      `json:"permission_count_after"`
	UpdatedAt            time.Time                `json:"updated_at"`
}

type RoleDeleteResponse struct {
	Message       string `json:"message"`
	DeletedRoleID string `json:"deleted_role_id"`
}

type AffectedProject struct {
	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name"`
	MemberCount int    `json:"member_count"`
}

type RoleInUseErrorResponse struct {
	Code                 string            `json:"code"`
	Message              string            `json:"message"`
	AffectedProjects     []AffectedProject `json:"affected_projects"`
	TotalAffectedMembers int               `json:"total_affected_members"`
}
