package _interface

import "TaskFlow-Go/internal/dto"

type TaskAssigneeService interface {
	ListAssignees(workspaceID string, userID string, projectID string, taskID string) (*dto.AssigneeListResponse, error)
	GetAvailableAssignees(workspaceID string, userID string, projectID string, taskID string, search string, page int, limit int) (*dto.AvailableAssigneeListResponse, error)
	AssignMembersToTask(workspaceID string, userID string, projectID string, taskID string, req *dto.AssignMembersRequest) (*dto.AssignMembersResponse, error)
	UnassignMembersFromTask(workspaceID string, userID string, projectID string, taskID string, req *dto.UnassignMembersRequest) (*dto.UnassignMembersResponse, error)
	SelfAssignToTask(workspaceID string, userID string, projectID string, taskID string) (*dto.SelfAssignResponse, error)
	SelfUnassignFromTask(workspaceID string, userID string, projectID string, taskID string) (*dto.SelfUnassignResponse, error)
}
