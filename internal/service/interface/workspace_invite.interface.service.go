package _interface

import "TaskFlow-Go/internal/dto"

type WorkspaceInviteService interface {
	ListInvites(workspaceID string, userID string, status string, page int, limit int) ([]dto.InviteInfo, *dto.Pagination, error)
	CreateInvite(workspaceID string, userID string, req *dto.CreateInviteRequest) (*dto.InviteCreateResponse, error)
	GetInvitePreview(code string) (*dto.InvitePreviewResponse, error)
	JoinWorkspaceByCode(code string, userID string) (*dto.JoinWorkspaceResponse, error)
	RevokeInvite(workspaceID string, userID string, inviteID string) (*dto.RevokeInviteResponse, error)
}
