package dto

type AddMembersRequest struct {
	Members []MemberRolePair `json:"members" binding:"required"`
}

type MemberRolePair struct {
	UserID string `json:"user_id" binding:"required"`
	RoleID string `json:"role_id" binding:"required"`
}

type UpdateProjectMemberRoleRequest struct {
	RoleID string `json:"role_id" binding:"required"`
}

type LeaveProjectRequest struct {
	Confirmation bool `json:"confirmation" binding:"required"`
}

type ProjectMemberInfo struct {
	UserID         string   `json:"user_id"`
	FullName       string   `json:"full_name"`
	Username       string   `json:"username"`
	Email          string   `json:"email"`
	AvatarURL      *string  `json:"avatar_url"`
	WorkspaceRole  string   `json:"workspace_role"`
	ProjectRole    *RoleRef `json:"project_role"`
	IsFavorite     bool     `json:"is_favorite"`
	JoinedAt       string   `json:"joined_at"`
}

type AvailableWorkspaceMember struct {
	UserID        string  `json:"user_id"`
	FullName      string  `json:"full_name"`
	Username      string  `json:"username"`
	Email         string  `json:"email"`
	AvatarURL     *string `json:"avatar_url"`
	WorkspaceRole string  `json:"workspace_role"`
}

type AddedMemberInfo struct {
	UserID    string   `json:"user_id"`
	FullName  string   `json:"full_name"`
	ProjectRole *RoleRef `json:"project_role"`
	JoinedAt  string   `json:"joined_at"`
}

type AddMembersResponse struct {
	Added              []AddedMemberInfo `json:"added"`
	SkippedAlreadyMember []string         `json:"skipped_already_member"`
	TotalAdded         int                `json:"total_added"`
}

type UpdateProjectMemberRoleResponse struct {
	UserID       string   `json:"user_id"`
	FullName     string   `json:"full_name"`
	PreviousRole *RoleRef `json:"previous_role"`
	CurrentRole  *RoleRef `json:"current_role"`
	UpdatedBy    string   `json:"updated_by"`
	UpdatedAt    string   `json:"updated_at"`
}

type RemoveProjectMemberResponse struct {
	Message        string `json:"message"`
	RemovedUserID  string `json:"removed_user_id"`
}

type LeaveProjectResponse struct {
	Message       string `json:"message"`
	LeftProjectID string `json:"left_project_id"`
}
