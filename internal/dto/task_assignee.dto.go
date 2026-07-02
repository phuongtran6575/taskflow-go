package dto

type AssignMembersRequest struct {
	UserIDs []string `json:"user_ids" binding:"required"`
}

type UnassignMembersRequest struct {
	UserIDs []string `json:"user_ids" binding:"required"`
}

type AssigneeDetail struct {
	UserID    string   `json:"user_id"`
	FullName  string   `json:"full_name"`
	Username  string   `json:"username"`
	AvatarURL *string  `json:"avatar_url"`
	ProjectRole *RoleRef `json:"project_role"`
	AssignedAt string  `json:"assigned_at"`
	AssignedBy *struct {
		UserID   string `json:"user_id"`
		FullName string `json:"full_name"`
	} `json:"assigned_by,omitempty"`
}

type AssigneeListResponse struct {
	TaskID    string           `json:"task_id"`
	TaskRef   string           `json:"task_ref"`
	Data      []AssigneeDetail `json:"data"`
	Total     int              `json:"total"`
}

type AvailableAssigneeInfo struct {
	UserID      string   `json:"user_id"`
	FullName    string   `json:"full_name"`
	Username    string   `json:"username"`
	AvatarURL   *string  `json:"avatar_url"`
	ProjectRole *RoleRef `json:"project_role"`
}

type AvailableAssigneeListResponse struct {
	Data       []AvailableAssigneeInfo `json:"data"`
	Pagination Pagination              `json:"pagination"`
}

type AddedAssigneeInfo struct {
	UserID     string  `json:"user_id"`
	FullName   string  `json:"full_name"`
	AvatarURL  *string `json:"avatar_url"`
	AssignedAt string  `json:"assigned_at"`
}

type AssignMembersResponse struct {
	TaskID               string              `json:"task_id"`
	TaskRef              string              `json:"task_ref"`
	Added                []AddedAssigneeInfo `json:"added"`
	SkippedAlreadyAssigned []string            `json:"skipped_already_assigned"`
	TotalAssigneesAfter  int                 `json:"total_assignees_after"`
}

type SkippedUser struct {
	UserID string `json:"user_id"`
	Reason string `json:"reason"`
}

type UnassignMembersResponse struct {
	TaskID              string          `json:"task_id"`
	TaskRef             string          `json:"task_ref"`
	Removed             []UserRef       `json:"removed"`
	SkippedNotAssigned  []SkippedUser   `json:"skipped_not_assigned"`
	TotalAssigneesAfter int             `json:"total_assignees_after"`
}

type UserRef struct {
	UserID   string `json:"user_id"`
	FullName string `json:"full_name"`
}

type SelfAssignResponse struct {
	TaskID              string `json:"task_id"`
	TaskRef             string `json:"task_ref"`
	Message             string `json:"message"`
	AssignedAt          string `json:"assigned_at"`
	TotalAssigneesAfter int    `json:"total_assignees_after"`
}

type SelfUnassignResponse struct {
	TaskID              string `json:"task_id"`
	TaskRef             string `json:"task_ref"`
	Message             string `json:"message"`
	TotalAssigneesAfter int    `json:"total_assignees_after"`
}
