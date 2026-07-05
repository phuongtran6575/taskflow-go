package implement

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/database"
	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/helper"
	"TaskFlow-Go/internal/job"
	"TaskFlow-Go/internal/models"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
)

var hexColorRegex = regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)
var projectKeyRegex = regexp.MustCompile(`^[A-Z0-9]{2,10}$`)
var validImageExt = regexp.MustCompile(`(?i)\.(jpg|jpeg|png|webp|gif)$`)
var wordSplitRegex = regexp.MustCompile(`[\s\-_]+`)

type projectService struct {
	tm              *database.TransactionManager
	projectRepo     repoInterface.ProjectRepository
	columnRepo      repoInterface.ColumnRepository
	memberRepo      repoInterface.ProjectMemberRepository
	workspaceRepo   repoInterface.WorkspaceRepository
	roleRepo        repoInterface.RoleRepository
	activityLogRepo repoInterface.ActivityLogRepository
	dispatcher      *job.Dispatcher
}

func NewProjectService(
	tm *database.TransactionManager,
	projectRepo repoInterface.ProjectRepository,
	columnRepo repoInterface.ColumnRepository,
	memberRepo repoInterface.ProjectMemberRepository,
	workspaceRepo repoInterface.WorkspaceRepository,
	roleRepo repoInterface.RoleRepository,
	activityLogRepo repoInterface.ActivityLogRepository,
	dispatcher *job.Dispatcher,
) _interface.ProjectService {
	return &projectService{
		tm:              tm,
		projectRepo:     projectRepo,
		columnRepo:      columnRepo,
		memberRepo:      memberRepo,
		workspaceRepo:   workspaceRepo,
		roleRepo:        roleRepo,
		activityLogRepo: activityLogRepo,
		dispatcher:      dispatcher,
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

// generateKey tự động tạo key từ tên project theo BR-PROJ-01
func (s *projectService) generateKey(name string) string {
	words := wordSplitRegex.Split(strings.TrimSpace(name), -1)
	var filtered []string
	for _, w := range words {
		if w != "" {
			filtered = append(filtered, w)
		}
	}

	var key string
	if len(filtered) >= 2 {
		maxWords := 5
		if len(filtered) < maxWords {
			maxWords = len(filtered)
		}
		for i := 0; i < maxWords; i++ {
			runes := []rune(strings.TrimSpace(filtered[i]))
			if len(runes) > 0 {
				key += string(unicode.ToUpper(runes[0]))
			}
		}
	} else if len(filtered) == 1 {
		clean := strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return r
			}
			return -1
		}, filtered[0])
		if len(clean) > 5 {
			clean = clean[:5]
		}
		key = strings.ToUpper(clean)
	}

	key = strings.Map(func(r rune) rune {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, strings.ToUpper(key))

	if len(key) < 2 {
		key = "PRJ"
	}
	if len(key) > 10 {
		key = key[:10]
	}
	return key
}

// resolveKey xử lý key: auto-generate nếu không nhập, validate, handle duplicate
func (s *projectService) resolveKey(workspaceID string, inputKey *string, projectName string) (string, error) {
	var key string
	if inputKey == nil || *inputKey == "" {
		key = s.generateKey(projectName)
	} else {
		key = strings.ToUpper(*inputKey)
		if !projectKeyRegex.MatchString(key) {
			return "", apperror.ErrInvalidKeyFormat
		}
	}

	existingProjects, err := s.projectRepo.ListByWorkspaceID(workspaceID)
	if err != nil {
		return "", apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list projects")
	}

	existingKeys := make(map[string]bool)
	for _, p := range existingProjects {
		existingKeys[strings.ToUpper(p.Key)] = true
	}

	if !existingKeys[key] {
		return key, nil
	}

	for suffix := 2; suffix <= 99; suffix++ {
		candidate := fmt.Sprintf("%s%d", key, suffix)
		if !existingKeys[candidate] {
			return candidate, nil
		}
	}

	return "", apperror.NewAppError(http.StatusConflict, "KEY_GENERATION_FAILED", "Cannot generate unique key after 99 attempts. Please provide a custom key.")
}

func (s *projectService) isValidBackground(bg string) bool {
	if hexColorRegex.MatchString(bg) {
		return true
	}
	if !strings.HasPrefix(bg, "https://") {
		return false
	}
	return validImageExt.MatchString(bg)
}

func (s *projectService) logActivity(workspaceID, projectID, userID string, action models.ActivityAction, metadata map[string]interface{}) {
	wsID := workspaceID
	uID := userID
	var metaStr *string
	if metadata != nil {
		b, _ := json.Marshal(metadata)
		s := string(b)
		metaStr = &s
	}
	_ = s.activityLogRepo.Create(&models.ActivityLog{
		WorkspaceID: &wsID,
		ProjectID:   &projectID,
		UserID:      &uID,
		Action:      action,
		EntityType:  models.EntityTypePROJECT,
		EntityID:    projectID,
		Metadata:    metaStr,
	})
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

	key, err := s.resolveKey(workspaceID, req.Key, req.Name)
	if err != nil {
		return nil, err
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

	existingProjects, err := s.projectRepo.ListByWorkspaceID(workspaceID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to count projects")
	}
	if err := helper.CheckProjectLimit(workspace.Plan, len(existingProjects)); err != nil {
		return nil, err
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

		wsMember, wsErr := s.workspaceRepo.GetMember(workspaceID, userID)
		if wsErr != nil {
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get workspace member")
		}

		// BR-PROJ-03: chọn role đầy đủ quyền nhất cho MEMBER
		var bestRole *models.Role
		if wsMember.Role == models.WorkspaceRoleMEMBER {
			bestRole = s.findBestRole(tx, workspaceID)
		}

		now := time.Now()
		projectMember := &models.ProjectMember{
			ProjectID:  project.ID,
			UserID:     userID,
			IsFavorite: false,
			JoinedAt:   now,
		}
		if bestRole != nil {
			projectMember.RoleID = &bestRole.ID
		}
		if err := tx.Create(projectMember).Error; err != nil {
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to add creator to project")
		}

		projectResponse, err := projectRepo.GetCreateProjectResponse(project.ID)
		if err != nil {
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get project response")
		}
		result = projectResponse

		s.logActivity(workspaceID, project.ID, userID, models.ActivityActionCREATE, map[string]interface{}{
			"project_name": req.Name,
			"project_key":  key,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// findBestRole tìm role tốt nhất: ưu tiên "Manager", nếu không có thì lấy role có nhiều permission nhất
func (s *projectService) findBestRole(tx *gorm.DB, workspaceID string) *models.Role {
	type roleWithCount struct {
		models.Role
		PermissionCount int
	}
	var roles []roleWithCount
	tx.Model(&models.Role{}).
		Select("roles.*, COUNT(rp.permission_id) as permission_count").
		Joins("LEFT JOIN role_permissions rp ON rp.role_id = roles.id").
		Where("roles.workspace_id = ?", workspaceID).
		Group("roles.id").
		Order("permission_count DESC").
		Scan(&roles)

	if len(roles) == 0 {
		return nil
	}
	for _, r := range roles {
		if r.Name == "Manager" {
			return &r.Role
		}
	}
	return &roles[0].Role
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

	var metadata map[string]interface{}

	if req.Name != nil {
		if len(*req.Name) < 1 || len(*req.Name) > 100 {
			return nil, apperror.NewAppError(http.StatusBadRequest, "VALIDATION_ERROR", "Project name must be between 1 and 100 characters")
		}
		if project.Name != *req.Name {
			metadata = map[string]interface{}{
				"old_name": project.Name,
				"new_name": *req.Name,
			}
		}
		project.Name = *req.Name
	}
	if req.Background != nil {
		if *req.Background != "" && !s.isValidBackground(*req.Background) {
			return nil, apperror.ErrInvalidBackground
		}
		if project.Background != *req.Background {
			metadata = map[string]interface{}{
				"field": "background",
			}
		}
		project.Background = *req.Background
	}
	if req.Icon != nil {
		oldIcon := ""
		if project.Icon != nil {
			oldIcon = *project.Icon
		}
		newIcon := ""
		if req.Icon != nil {
			newIcon = *req.Icon
		}
		if oldIcon != newIcon {
			metadata = map[string]interface{}{
				"old_icon": oldIcon,
				"new_icon": newIcon,
			}
		}
		project.Icon = req.Icon
	}

	project.UpdatedAt = time.Now()
	if err := s.projectRepo.Update(project); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update project")
	}

	if metadata != nil {
		s.logActivity(workspaceID, project.ID, userID, models.ActivityActionUPDATE, metadata)
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

	s.logActivity(workspaceID, project.ID, userID, models.ActivityActionUPDATE, map[string]interface{}{
		"is_archived": true,
	})

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

	s.logActivity(workspaceID, project.ID, userID, models.ActivityActionUPDATE, map[string]interface{}{
		"is_archived": false,
	})

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

	taskCount, _ := s.projectRepo.CountTasksByProject(projectID)

	if err := s.projectRepo.Delete(projectID); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete project")
	}

	s.dispatcher.CascadeSoftDeleteProject(projectID)

	s.logActivity(workspaceID, project.ID, userID, models.ActivityActionDELETE, map[string]interface{}{
		"name":                project.Name,
		"key":                 project.Key,
		"total_tasks_deleted": taskCount,
	})

	return &dto.ProjectDeleteResponse{
		Message:          "Project '" + project.Name + "' has been deleted successfully.",
		DeletedProjectID: project.ID,
	}, nil
}
