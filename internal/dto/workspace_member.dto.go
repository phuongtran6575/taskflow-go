package dto

type UpdateMemberRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

type TransferOwnershipRequest struct {
	NewOwnerID   string `json:"new_owner_id" binding:"required"`
	Confirmation string `json:"confirmation" binding:"required"`
}

type LeaveWorkspaceRequest struct {
	Confirmation bool `json:"confirmation" binding:"required"`
}

type MemberInfo struct {
	UserID    string  `json:"user_id"`
	FullName  string  `json:"full_name"`
	Username  string  `json:"username"`
	Email     string  `json:"email"`
	AvatarURL *string `json:"avatar_url"`
	Role      string  `json:"role"`
	JoinedAt  string  `json:"joined_at"`
}

type ProjectRoleSummary struct {
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
	Key       string `json:"key"`
	RoleName  string `json:"role_name"`
}

type MemberDetailResponse struct {
	UserID    string               `json:"user_id"`
	FullName  string               `json:"full_name"`
	Username  string               `json:"username"`
	Email     string               `json:"email"`
	AvatarURL *string              `json:"avatar_url"`
	Role      string               `json:"role"`
	JoinedAt  string               `json:"joined_at"`
	Projects  []ProjectRoleSummary `json:"projects" gorm:"-"`
}

type UpdateRoleResponse struct {
	UserID       string `json:"user_id"`
	FullName     string `json:"full_name"`
	PreviousRole string `json:"previous_role"`
	CurrentRole  string `json:"current_role"`
	UpdatedBy    string `json:"updated_by"`
	UpdatedAt    string `json:"updated_at"`
}

type TransferOwnershipResponse struct {
	Message        string         `json:"message"`
	PreviousOwner  OwnerRoleInfo  `json:"previous_owner"`
	NewOwner       OwnerRoleInfo  `json:"new_owner"`
}

type OwnerRoleInfo struct {
	UserID  string `json:"user_id"`
	NewRole string `json:"new_role"`
}

type KickMemberResponse struct {
	Message          string   `json:"message"`
	RemovedUserID    string   `json:"removed_user_id"`
	RemovedFromProjects []string `json:"removed_from_projects"`
}

type LeaveWorkspaceResponse struct {
	Message         string `json:"message"`
	LeftWorkspaceID string `json:"left_workspace_id"`
}
