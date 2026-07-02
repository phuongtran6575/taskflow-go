package _interface

import "TaskFlow-Go/internal/dto"

type ProjectService interface {
	ListProjects(workspaceID string, userID string, isOwner bool, isArchived *bool, isFavorite *bool, search string, param dto.PaginationParam) ([]dto.ProjectSummary, *dto.Pagination, error)
	CreateProject(workspaceID string, userID string, req *dto.CreateProjectRequest) (*dto.ProjectCreateResponse, error)
	GetProjectById(workspaceID string, userID string, projectID string) (*dto.ProjectDetailResponse, error)
	UpdateProject(workspaceID string, userID string, projectID string, req *dto.UpdateProjectRequest) (*dto.UpdateProjectResponse, error)
	ArchiveProject(workspaceID string, userID string, projectID string) (*dto.ArchiveProjectResponse, error)
	UnarchiveProject(workspaceID string, userID string, projectID string) (*dto.UnarchiveProjectResponse, error)
	ToggleFavorite(workspaceID string, userID string, projectID string) (*dto.FavoriteResponse, error)
	DeleteProject(workspaceID string, userID string, projectID string, req *dto.DeleteProjectRequest) (*dto.ProjectDeleteResponse, error)
}
