package dto

import "time"

type CreateInviteRequest struct {
	Role      string     `json:"role,omitempty"`
	MaxUses   *int       `json:"max_uses,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type InviteCreatorInfo struct {
	UserID   string  `json:"user_id"`
	FullName string  `json:"full_name"`
	AvatarURL *string `json:"avatar_url"`
}

type InviteInfo struct {
	ID        string             `json:"id"`
	Code      string             `json:"code"`
	URL       string             `json:"url"`
	Role      string             `json:"role"`
	MaxUses   *int               `json:"max_uses"`
	UsesCount int                `json:"uses_count"`
	ExpiresAt *time.Time         `json:"expires_at"`
	Status    string             `json:"status"`
	CreatedBy InviteCreatorInfo  `json:"created_by"`
	CreatedAt time.Time          `json:"created_at"`
	DeletedAt *time.Time         `json:"deleted_at"`
}

type InviteCreateResponse struct {
	ID        string             `json:"id"`
	Code      string             `json:"code"`
	URL       string             `json:"url"`
	Role      string             `json:"role"`
	MaxUses   *int               `json:"max_uses"`
	UsesCount int                `json:"uses_count"`
	ExpiresAt *time.Time         `json:"expires_at"`
	Status    string             `json:"status"`
	CreatedBy InviteCreatorInfo  `json:"created_by"`
	CreatedAt time.Time          `json:"created_at"`
}

type InviteWorkspacePreview struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	MemberCount int    `json:"member_count"`
}

type InvitePreviewInfo struct {
	Code          string     `json:"code"`
	Role          string     `json:"role"`
	ExpiresAt     *time.Time `json:"expires_at"`
	RemainingUses *int       `json:"remaining_uses"`
}

type InvitePreviewResponse struct {
	Workspace InviteWorkspacePreview `json:"workspace"`
	Invite    InvitePreviewInfo      `json:"invite"`
	Status    string                 `json:"status"`
}

type JoinWorkspaceResponse struct {
	Message    string `json:"message"`
	Workspace  struct {
		ID     string  `json:"id"`
		Name   string  `json:"name"`
		Domain *string `json:"domain"`
	} `json:"workspace"`
	JoinedAsRole string `json:"joined_as_role"`
}

type RevokeInviteResponse struct {
	Message  string    `json:"message"`
	InviteID string    `json:"invite_id"`
	RevokedAt time.Time `json:"revoked_at"`
}
