package implement

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/activitylog"
	"TaskFlow-Go/internal/database"
	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/helper"
	"TaskFlow-Go/internal/job"
	"TaskFlow-Go/internal/models"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
	"TaskFlow-Go/internal/validator"
)

var projectKeyRegex = regexp.MustCompile(`^[A-Z0-9]{2,10}$`)

type projectService struct {
	tm              *database.TransactionManager
	projectRepo     repoInterface.ProjectRepository
	columnRepo      repoInterface.ColumnRepository
	memberRepo      repoInterface.ProjectMemberRepository
	workspaceRepo   repoInterface.WorkspaceRepository
	roleRepo        repoInterface.RoleRepository
	activityLogRepo repoInterface.ActivityLogRepository
	userRepo        repoInterface.UserRepository
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
	userRepo repoInterface.UserRepository,
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
		userRepo:        userRepo,
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

// resolveKey xử lý key: auto-generate nếu không nhập, validate, handle duplicate
func (s *projectService) resolveKey(workspaceID string, inputKey *string, projectName string) (string, error) {
	var key string
	if inputKey == nil || *inputKey == "" {
		key = helper.GenerateProjectKey(projectName)
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

func (s *projectService) logActivity(workspaceID, projectID, userID, entityID string, action models.ActivityAction, metadata map[string]interface{}, description string, entitySnapshot map[string]interface{}) {
	wsID := workspaceID
	uID := userID
	var metaStr *string
	if metadata != nil {
		b, _ := json.Marshal(metadata)
		str := string(b)
		metaStr = &str
	}
	var snapStr *string
	if entitySnapshot != nil {
		b, _ := json.Marshal(entitySnapshot)
		str := string(b)
		snapStr = &str
	}
	var descPtr *string
	if description != "" {
		descPtr = &description
	}
	_ = s.activityLogRepo.Create(&models.ActivityLog{
		WorkspaceID:    &wsID,
		ProjectID:      &projectID,
		UserID:         &uID,
		Action:         action,
		EntityType:     models.EntityTypePROJECT,
		EntityID:       entityID,
		Description:    descPtr,
		Metadata:       metaStr,
		EntitySnapshot: snapStr,
	})
}

func (s *projectService) logActivityInTx(tx *gorm.DB, workspaceID, projectID, userID, entityID string, action models.ActivityAction, metadata map[string]interface{}, description string, entitySnapshot map[string]interface{}) {
	wsID := workspaceID
	uID := userID
	var metaStr *string
	if metadata != nil {
		b, _ := json.Marshal(metadata)
		str := string(b)
		metaStr = &str
	}
	var snapStr *string
	if entitySnapshot != nil {
		b, _ := json.Marshal(entitySnapshot)
		str := string(b)
		snapStr = &str
	}
	var descPtr *string
	if description != "" {
		descPtr = &description
	}
	_ = tx.Create(&models.ActivityLog{
		WorkspaceID:    &wsID,
		ProjectID:      &projectID,
		UserID:         &uID,
		Action:         action,
		EntityType:     models.EntityTypePROJECT,
		EntityID:       entityID,
		Description:    descPtr,
		Metadata:       metaStr,
		EntitySnapshot: snapStr,
	}).Error
}

func (s *projectService) getUserName(userID string) string {
	u, err := s.userRepo.GetByID(userID)
	if err != nil {
		return userID
	}
	return u.FullName
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

	if req.Background != nil && *req.Background != "" && !validator.IsValidProjectBackground(*req.Background) {
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

		actorName := s.getUserName(userID)
		meta := activitylog.ProjectCreated(req.Name, key)
		desc := activitylog.GenerateDescription(actorName, meta)
		snap := activitylog.BuildProjectSnapshot(req.Name, key)
		s.logActivityInTx(tx, workspaceID, project.ID, userID, project.ID, models.ActivityActionCREATE, meta, desc, snap)
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

	var changes []activitylog.ChangeField

	if req.Name != nil {
		if len(*req.Name) < 1 || len(*req.Name) > 100 {
			return nil, apperror.NewAppError(http.StatusBadRequest, "VALIDATION_ERROR", "Project name must be between 1 and 100 characters")
		}
		if project.Name != *req.Name {
			changes = append(changes, activitylog.BuildChangeField("name", project.Name, *req.Name))
		}
		project.Name = *req.Name
	}
	if req.Background != nil {
		if *req.Background != "" && !validator.IsValidProjectBackground(*req.Background) {
			return nil, apperror.ErrInvalidBackground
		}
		if project.Background != *req.Background {
			changes = append(changes, activitylog.BuildChangeField("background", project.Background, *req.Background))
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
			changes = append(changes, activitylog.BuildChangeField("icon", oldIcon, newIcon))
		}
		project.Icon = req.Icon
	}

	actorName := s.getUserName(userID)
	meta := activitylog.ProjectUpdated(changes)
	desc := activitylog.GenerateDescription(actorName, meta)
	snap := activitylog.BuildProjectSnapshot(project.Name, project.Key)

	err = s.tm.Execute(func(tx *gorm.DB) error {
		project.UpdatedAt = time.Now()
		if err := s.projectRepo.WithTx(tx).Update(project); err != nil {
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update project")
		}
		if len(changes) > 0 {
			s.logActivityInTx(tx, workspaceID, project.ID, userID, project.ID, models.ActivityActionUPDATE, meta, desc, snap)
		}
		return nil
	})
	if err != nil {
		return nil, err
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

	actorName := s.getUserName(userID)
	meta := activitylog.ProjectArchived()
	desc := activitylog.GenerateDescription(actorName, meta)
	snap := activitylog.BuildProjectSnapshot(project.Name, project.Key)

	var archivedAt time.Time
	err = s.tm.Execute(func(tx *gorm.DB) error {
		now := time.Now()
		archivedAt = now
		project.IsArchived = true
		project.ArchivedAt = &now
		project.ArchivedByID = &userID
		project.UpdatedAt = now
		if err := s.projectRepo.WithTx(tx).Update(project); err != nil {
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to archive project")
		}
		s.logActivityInTx(tx, workspaceID, project.ID, userID, project.ID, models.ActivityActionUPDATE, meta, desc, snap)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &dto.ArchiveProjectResponse{
		ID:         project.ID,
		Name:       project.Name,
		IsArchived: true,
		ArchivedAt: archivedAt,
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

	actorName := s.getUserName(userID)
	meta := activitylog.ProjectUnarchived()
	desc := activitylog.GenerateDescription(actorName, meta)
	snap := activitylog.BuildProjectSnapshot(project.Name, project.Key)

	var unarchivedAt time.Time
	err = s.tm.Execute(func(tx *gorm.DB) error {
		now := time.Now()
		unarchivedAt = now
		project.IsArchived = false
		project.ArchivedAt = nil
		project.ArchivedByID = nil
		project.UpdatedAt = now
		if err := s.projectRepo.WithTx(tx).Update(project); err != nil {
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to unarchive project")
		}
		s.logActivityInTx(tx, workspaceID, project.ID, userID, project.ID, models.ActivityActionUPDATE, meta, desc, snap)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &dto.UnarchiveProjectResponse{
		ID:           project.ID,
		Name:         project.Name,
		IsArchived:   false,
		UnarchivedAt: unarchivedAt,
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

	actorName := s.getUserName(userID)
	meta := activitylog.ProjectDeleted(project.Name, project.Key, int(taskCount))
	desc := activitylog.GenerateDescription(actorName, meta)
	snap := activitylog.BuildProjectSnapshot(project.Name, project.Key)

	err = s.tm.Execute(func(tx *gorm.DB) error {
		if err := s.projectRepo.WithTx(tx).Delete(projectID); err != nil {
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete project")
		}
		s.logActivityInTx(tx, workspaceID, project.ID, userID, project.ID, models.ActivityActionDELETE, meta, desc, snap)
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.dispatcher.CascadeSoftDeleteProject(projectID)

	return &dto.ProjectDeleteResponse{
		Message:          "Project '" + project.Name + "' has been deleted successfully.",
		DeletedProjectID: project.ID,
	}, nil
}
