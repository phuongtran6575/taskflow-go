package _interface

import (
	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"

	"gorm.io/gorm"
)

type ProjectRepository interface {
	WithTx(tx *gorm.DB) ProjectRepository
	Create(project *models.Project) error
	GetByID(id string) (*models.Project, error)
	ListByWorkspaceID(workspaceID string) ([]models.Project, error)
	Update(project *models.Project) error
	Delete(id string) error

	GetCreateProjectResponse(id string) (*dto.ProjectCreateResponse, error)
	GetListMemberProject(workspaceID string, userID string, isArchived *bool, isFavorite *bool, search string, param dto.PaginationParam) ([]dto.ProjectSummary, *dto.Pagination, error)
	GetListWorkspaceProject(workspaceID string, userID string, isArchived *bool, isFavorite *bool, search string, param dto.PaginationParam) ([]dto.ProjectSummary, *dto.Pagination, error)
	GetByIDWithDetail(workspaceID string, userID string, projectID string) (*dto.ProjectDetailResponse, error)
}
