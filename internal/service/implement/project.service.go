package implement

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/database"
	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/helper"
	"TaskFlow-Go/internal/models"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
)

var hexColorRegex = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
var projectKeyRegex = regexp.MustCompile(`^[A-Z0-9]{2,10}$`)

type projectService struct {
	tm            *database.TransactionManager
	projectRepo   repoInterface.ProjectRepository
	columnRepo    repoInterface.ColumnRepository
	memberRepo    repoInterface.ProjectMemberRepository
	workspaceRepo repoInterface.WorkspaceRepository
	roleRepo      repoInterface.RoleRepository
}

func NewProjectService(
	tm *database.TransactionManager,
	projectRepo repoInterface.ProjectRepository,
	columnRepo repoInterface.ColumnRepository,
	memberRepo repoInterface.ProjectMemberRepository,
	workspaceRepo repoInterface.WorkspaceRepository,
	roleRepo repoInterface.RoleRepository,
) _interface.ProjectService {
	return &projectService{
		tm:            tm,
		projectRepo:   projectRepo,
		columnRepo:    columnRepo,
		memberRepo:    memberRepo,
		workspaceRepo: workspaceRepo,
		roleRepo:      roleRepo,
	}
}

func (s *projectService) getProjectOrFail(workspaceID, projectID string) (*models.Project, error) {
	project, err := s.projectRepo.GetByID(projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrProjectNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get project")
	}
	if project.WorkspaceID != workspaceID {
		return nil, apperror.ErrProjectNotFound
	}
	if project.DeletedAt.Valid {
		return nil, apperror.ErrProjectNotFound
	}
	return project, nil
}

func (s *projectService) isValidBackground(bg string) bool {
	if hexColorRegex.MatchString(bg) {
		return true
	}
	return strings.HasPrefix(bg, "http://") || strings.HasPrefix(bg, "https://")
}

func (s *projectService) ListProjects(workspaceID string, userID string, isOwner bool, isArchived *bool, isFavorite *bool, search string, param dto.PaginationParam) ([]dto.ProjectSummary, *dto.Pagination, error) {
	if isOwner {
		return s.projectRepo.GetListWorkspaceProject(workspaceID, userID, isArchived, isFavorite, search, param)
	}
	return s.projectRepo.GetListMemberProject(workspaceID, userID, isArchived, isFavorite, search, param)
}

func (s *projectService) CreateProject(workspaceID string, userID string, req *dto.CreateProjectRequest) (*dto.ProjectCreateResponse, error) {
	if len(req.Name) < 1 || len(req.Name) > 100 {
		return nil, apperror.NewAppError(http.StatusBadRequest, "VALIDATION_ERROR", "Project name must be between 1 and 100 characters")
	}
	key := strings.ToUpper(req.Key)
	if !projectKeyRegex.MatchString(key) {
		return nil, apperror.ErrInvalidKeyFormat
	}
	if req.Background != nil && *req.Background != "" && !s.isValidBackground(*req.Background) {
		return nil, apperror.ErrInvalidBackground
	}

	workspace, err := s.workspaceRepo.GetByID(workspaceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrWorkspaceNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get workspace")
	}

	existingProjects, err := 	s.projectRepo.ListByWorkspaceID(workspaceID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to count projects")
	}
	if err := helper.CheckProjectLimit(workspace.Plan, len(existingProjects)); err != nil {
		return nil, err
	}

	for _, p := range existingProjects {
		if p.Key == key {
			return nil, apperror.ErrProjectKeyAlreadyExists
		}
	}

	bg := "#ffffff"
	if req.Background != nil && *req.Background != "" {
		bg = *req.Background
	}

	var result *dto.ProjectCreateResponse
	err = s.tm.Execute(func(tx *gorm.DB) error {
		projectRepo := s.projectRepo.WithTx(tx)
		columnRepo := s.columnRepo.WithTx(tx)

		project := &models.Project{
			WorkspaceID: workspaceID,
			OwnerID:     &userID,
			Name:        req.Name,
			Key:         key,
			Icon:        req.Icon,
			Background:  bg,
		}
		if err := projectRepo.Create(project); err != nil {
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create project")
		}

		defaultColumns := []models.Column{
			{ProjectID: project.ID, Title: "To Do", Position: 1000},
			{ProjectID: project.ID, Title: "In Progress", Position: 2000},
			{ProjectID: project.ID, Title: "Done", Position: 3000},
		}
		if err := columnRepo.CreateListColumn(defaultColumns); err != nil {
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create default columns")
		}

		projectResponse, err := projectRepo.GetCreateProjectResponse(project.ID)
		if err != nil {
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get project response")
		}
		result = projectResponse
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *projectService) GetProjectById(workspaceID string, userID string, projectID string) (*dto.ProjectDetailResponse, error) {
	_, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	project, err := s.projectRepo.GetByIDWithDetail(workspaceID, userID, projectID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get project details")
	}
	return project, nil
}

func (s *projectService) UpdateProject(workspaceID string, userID string, projectID string, req *dto.UpdateProjectRequest) (*dto.UpdateProjectResponse, error) {
	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if project.IsArchived {
		return nil, apperror.ErrProjectArchived
	}

	if req.Name != nil {
		if len(*req.Name) < 1 || len(*req.Name) > 100 {
			return nil, apperror.NewAppError(http.StatusBadRequest, "VALIDATION_ERROR", "Project name must be between 1 and 100 characters")
		}
		project.Name = *req.Name
	}
	if req.Background != nil {
		if *req.Background != "" && !s.isValidBackground(*req.Background) {
			return nil, apperror.ErrInvalidBackground
		}
		project.Background = *req.Background
	}
	if req.Icon != nil {
		project.Icon = req.Icon
	}
	project.UpdatedAt = time.Now()
	if err := s.projectRepo.Update(project); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update project")
	}

	return &dto.UpdateProjectResponse{
		ID:         project.ID,
		Name:       project.Name,
		Key:        project.Key,
		Background: project.Background,
		Icon:       project.Icon,
		UpdatedAt:  project.UpdatedAt,
	}, nil
}

func (s *projectService) ArchiveProject(workspaceID string, userID string, projectID string) (*dto.ArchiveProjectResponse, error) {
	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if project.IsArchived {
		return nil, apperror.ErrAlreadyArchived
	}

	now := time.Now()
	project.IsArchived = true
	project.ArchivedAt = &now
	project.ArchivedByID = &userID
	project.UpdatedAt = now
	if err := s.projectRepo.Update(project); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to archive project")
	}

	return &dto.ArchiveProjectResponse{
		ID:         project.ID,
		Name:       project.Name,
		IsArchived: true,
		ArchivedAt: now,
		ArchivedBy: userID,
	}, nil
}

func (s *projectService) UnarchiveProject(workspaceID string, userID string, projectID string) (*dto.UnarchiveProjectResponse, error) {
	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if !project.IsArchived {
		return nil, apperror.ErrNotArchived
	}

	project.IsArchived = false
	project.ArchivedAt = nil
	project.ArchivedByID = nil
	project.UpdatedAt = time.Now()
	if err := s.projectRepo.Update(project); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to unarchive project")
	}

	return &dto.UnarchiveProjectResponse{
		ID:           project.ID,
		Name:         project.Name,
		IsArchived:   false,
		UnarchivedAt: project.UpdatedAt,
		UnarchivedBy: userID,
	}, nil
}

func (s *projectService) ToggleFavorite(workspaceID string, userID string, projectID string) (*dto.FavoriteResponse, error) {
	_, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	member, err := s.memberRepo.GetByID(projectID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotAProjectMember
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get member")
	}

	member.IsFavorite = !member.IsFavorite
	if err := s.memberRepo.Update(member); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to toggle favorite")
	}

	return &dto.FavoriteResponse{
		ProjectID:  member.ProjectID,
		IsFavorite: member.IsFavorite,
	}, nil
}

func (s *projectService) DeleteProject(workspaceID string, userID string, projectID string, req *dto.DeleteProjectRequest) (*dto.ProjectDeleteResponse, error) {
	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}

	if req.ConfirmationName != project.Name {
		return nil, apperror.ErrInvalidConfirmation
	}

	if err := s.projectRepo.Delete(projectID); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete project")
	}

	return &dto.ProjectDeleteResponse{
		Message:         "Project '" + project.Name + "' has been deleted successfully.",
		DeletedProjectID: project.ID,
	}, nil
}
