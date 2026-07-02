package _interface

import "TaskFlow-Go/internal/dto"

type LabelService interface {
	ListProjectLabels(workspaceID string, userID string, projectID string, search string, withTaskCount bool) (*dto.LabelListResponse, error)
	CreateLabel(workspaceID string, userID string, projectID string, req *dto.CreateLabelRequest) (*dto.LabelCreateResponse, error)
	UpdateLabel(workspaceID string, userID string, projectID string, labelID string, req *dto.UpdateLabelRequest) (*dto.LabelUpdateResponse, error)
	DeleteLabel(workspaceID string, userID string, projectID string, labelID string) (*dto.LabelDeleteResponse, error)
	ListTaskLabels(workspaceID string, userID string, projectID string, taskID string) (*dto.TaskLabelListResponse, error)
	AssignLabelsToTask(workspaceID string, userID string, projectID string, taskID string, req *dto.AssignLabelsRequest) (*dto.AssignLabelsResponse, error)
	RemoveLabelsFromTask(workspaceID string, userID string, projectID string, taskID string, req *dto.RemoveLabelsRequest) (*dto.RemoveLabelsResponse, error)
}
