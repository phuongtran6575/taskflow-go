package _interface

import "TaskFlow-Go/internal/dto"

type WorkspaceService interface {
	GetWorkspacesByUserId(userID string) (*dto.WorkspaceListResponse, error)
	CreateWorkspace(userID string, req *dto.CreateWorkspaceRequest) (*dto.WorkspaceCreateResponse, error)
	GetWorkspaceById(workspaceID string, userID string) (*dto.WorkspaceDetailResponse, error)
	UpdateWorkspace(workspaceID string, userID string, req *dto.UpdateWorkspaceRequest) (*dto.UpdateWorkspaceResponse, error)
	UpgradePlan(workspaceID string, userID string, req *dto.UpgradePlanRequest) (*dto.PlanUpgradeResponse, error)
	DeleteWorkspace(workspaceID string, userID string, req *dto.DeleteWorkspaceRequest) (*dto.DeleteWorkspaceResponse, error)
}
