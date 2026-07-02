package _interface

import "TaskFlow-Go/internal/dto"

type ProjectMemberService interface {
	ListMembers(workspaceID string, userID string, projectID string, page int, limit int, search string, roleID string) ([]dto.ProjectMemberInfo, *dto.Pagination, error)
	GetAvailableWorkspaceMembers(workspaceID string, userID string, projectID string, search string, page int, limit int) ([]dto.AvailableWorkspaceMember, *dto.Pagination, error)
	AddMembersToProject(workspaceID string, userID string, projectID string, req *dto.AddMembersRequest) (*dto.AddMembersResponse, error)
	UpdateMemberRole(workspaceID string, userID string, projectID string, targetUserID string, req *dto.UpdateProjectMemberRoleRequest) (*dto.UpdateProjectMemberRoleResponse, error)
	RemoveMemberFromProject(workspaceID string, userID string, projectID string, targetUserID string) (*dto.RemoveProjectMemberResponse, error)
	LeaveProject(workspaceID string, userID string, projectID string, req *dto.LeaveProjectRequest) (*dto.LeaveProjectResponse, error)
}
