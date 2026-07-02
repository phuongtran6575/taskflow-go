package _interface

import "TaskFlow-Go/internal/dto"

type ColumnService interface {
	ListColumns(workspaceID string, userID string, projectID string) (*dto.ColumnListResponse, error)
	CreateColumn(workspaceID string, userID string, projectID string, req *dto.CreateColumnRequest) (*dto.ColumnCreateResponse, error)
	UpdateColumnTitle(workspaceID string, userID string, projectID string, columnID string, req *dto.UpdateColumnTitleRequest) (*dto.UpdateColumnTitleResponse, error)
	UpdateColumnPosition(workspaceID string, userID string, projectID string, columnID string, req *dto.UpdateColumnPositionRequest) (*dto.UpdateColumnPositionResponse, error)
	DeleteColumn(workspaceID string, userID string, projectID string, columnID string, req *dto.DeleteColumnRequest) (*dto.ColumnDeleteResponse, error)
}
