package _interface

import "TaskFlow-Go/internal/dto"

type WorkspaceMemberService interface {
	ListMembers(workspaceID string, userID string, page int, limit int, search string, role string) ([]dto.MemberInfo, *dto.Pagination, error)
	GetMemberDetails(workspaceID string, targetUserID string) (*dto.MemberDetailResponse, error)
	UpdateMemberRole(workspaceID string, userID string, targetUserID string, req *dto.UpdateMemberRoleRequest) (*dto.UpdateRoleResponse, error)
	TransferOwnership(workspaceID string, userID string, req *dto.TransferOwnershipRequest) (*dto.TransferOwnershipResponse, error)
	KickMember(workspaceID string, userID string, targetUserID string) (*dto.KickMemberResponse, error)
	LeaveWorkspace(workspaceID string, userID string, req *dto.LeaveWorkspaceRequest) (*dto.LeaveWorkspaceResponse, error)
}
