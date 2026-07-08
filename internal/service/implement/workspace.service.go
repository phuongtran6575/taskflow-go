package implement

import (
	"encoding/json"
	"errors"
	"fmt"
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

	"github.com/google/uuid"
)

type workspaceService struct {
	tm                 *database.TransactionManager
	workspaceRepo      repoInterface.WorkspaceRepository
	roleRepo           repoInterface.RoleRepository
	rolePermissionRepo repoInterface.RolePermissionRepository
	activityLogRepo    repoInterface.ActivityLogRepository
	userRepo           repoInterface.UserRepository
	dispatcher         *job.Dispatcher
}

func NewWorkspaceService(
	tm *database.TransactionManager,
	workspaceRepo repoInterface.WorkspaceRepository,
	roleRepo repoInterface.RoleRepository,
	rolePermissionRepo repoInterface.RolePermissionRepository,
	activityLogRepo repoInterface.ActivityLogRepository,
	userRepo repoInterface.UserRepository,
	dispatcher *job.Dispatcher,
) _interface.WorkspaceService {
	return &workspaceService{
		tm:                 tm,
		workspaceRepo:      workspaceRepo,
		roleRepo:           roleRepo,
		rolePermissionRepo: rolePermissionRepo,
		activityLogRepo:    activityLogRepo,
		userRepo:           userRepo,
		dispatcher:         dispatcher,
	}
}

// da review
func (s *workspaceService) GetWorkspacesByUserId(userID string) (*dto.WorkspaceListResponse, error) {
	workspaces, total, err := s.workspaceRepo.ListByUserIDWithSummary(userID)
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get workspaces")
	}
	return &dto.WorkspaceListResponse{Data: workspaces, Total: total}, nil
}

func (s *workspaceService) CreateWorkspace(userID string, req *dto.CreateWorkspaceRequest) (*dto.WorkspaceCreateResponse, error) {
	if err := validator.ValidateWorkspaceName(req.Name); err != nil {
		return nil, apperror.NewAppError(400, "VALIDATION_ERROR", err.Error())
	}

	if req.Domain != nil {
		if err := validator.ValidateWorkspaceDomain(*req.Domain); err != nil {
			return nil, apperror.NewAppError(400, "VALIDATION_ERROR", err.Error())
		}
		existingWorkspace, err := s.workspaceRepo.GetWorkspaceByDomain(*req.Domain)
		if err != nil {
			return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to check workspace domain")
		}
		if helper.IsDomainTaken(existingWorkspace, "") {
			return nil, apperror.ErrDomainAlreadyTaken
		}
	}

	planCounts, err := s.workspaceRepo.CountWorkspaceByPlan(
		helper.AllWorkspacePlans(),
		models.WorkspaceRoleOWNER,
		userID,
	)
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to count workspaces")
	}

	if err := helper.CheckWorkspacePlanLimit(planCounts, models.WorkspacePlanFREE); err != nil {
		return nil, err
	}

	workspace := &models.Workspace{
		OwnerID: userID,
		Name:    req.Name,
		Domain:  req.Domain,
	}

	actorName := s.getUserName(userID)
	meta := activitylog.WorkspaceCreated(workspace.Name, string(workspace.Plan))
	desc := activitylog.GenerateDescription(actorName, meta)
	snap := activitylog.BuildWorkspaceSnapshot(workspace.Name)

	err = s.tm.Execute(func(tx *gorm.DB) error {
		wsTx := s.workspaceRepo.WithTx(tx)
		if err := wsTx.Create(workspace); err != nil {
			return apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to create workspace")
		}
		if err := wsTx.AddMember(&models.WorkspaceMember{
			WorkspaceID: workspace.ID,
			UserID:      userID,
			Role:        models.WorkspaceRoleOWNER,
			JoinedAt:    time.Now(),
		}); err != nil {
			return apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to add workspace owner")
		}
		s.logActivityInTx(tx, workspace.ID, "", userID, workspace.ID, models.ActivityActionCREATE, meta, desc, snap)
		return nil
	})
	if err != nil {
		return nil, err
	}

	// BR-ROLE-08: Seed default roles (Manager, Developer, Viewer)
	if err := s.seedDefaultRoles(workspace.ID); err != nil {
		return nil, err
	}

	return &dto.WorkspaceCreateResponse{
		ID:        workspace.ID,
		Name:      workspace.Name,
		Domain:    workspace.Domain,
		Plan:      string(workspace.Plan),
		OwnerID:   workspace.OwnerID,
		CreatedAt: workspace.CreatedAt,
	}, nil
}

func (s *workspaceService) GetWorkspaceById(workspaceID string, userID string) (*dto.WorkspaceDetailResponse, error) {
	workspace, err := s.workspaceRepo.GetByIDWithDetail(workspaceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrWorkspaceNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get workspace")
	}

	if _, err := s.workspaceRepo.GetMember(workspaceID, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotAMember
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to verify membership")
	}

	workspace.IsOverLimit = helper.IsWorkspaceOverLimit(
		models.WorkspacePlan(workspace.Plan),
		workspace.MemberCount,
		workspace.ProjectCount,
		0,
	)

	return workspace, nil
}

func (s *workspaceService) UpdateWorkspace(workspaceID string, userID string, req *dto.UpdateWorkspaceRequest) (*dto.UpdateWorkspaceResponse, error) {
	workspace, err := s.workspaceRepo.GetByID(workspaceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrWorkspaceNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get workspace")
	}

	member, err := s.workspaceRepo.GetMember(workspaceID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrWorkspaceNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to verify membership")
	}
	if member.Role != models.WorkspaceRoleOWNER && member.Role != models.WorkspaceRoleADMIN {
		return nil, apperror.ErrForbidden
	}

	var changes []activitylog.ChangeField

	if req.Name != nil {
		if err := validator.ValidateWorkspaceName(*req.Name); err != nil {
			return nil, apperror.NewAppError(400, "VALIDATION_ERROR", err.Error())
		}
		if workspace.Name != *req.Name {
			changes = append(changes, activitylog.BuildChangeField("name", workspace.Name, *req.Name))
		}
		workspace.Name = *req.Name
	}

	if req.Domain != nil {
		if *req.Domain != "" {
			if err := validator.ValidateWorkspaceDomain(*req.Domain); err != nil {
				return nil, apperror.NewAppError(400, "VALIDATION_ERROR", err.Error())
			}
			existingWorkspace, err := s.workspaceRepo.GetWorkspaceByDomain(*req.Domain)
			if err != nil {
				return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to check workspace domain")
			}
			if helper.IsDomainTaken(existingWorkspace, workspaceID) {
				return nil, apperror.ErrDomainAlreadyTaken
			}
			oldDomain := ""
			if workspace.Domain != nil {
				oldDomain = *workspace.Domain
			}
			changes = append(changes, activitylog.BuildChangeField("domain", oldDomain, *req.Domain))
			workspace.Domain = req.Domain
		} else {
			if workspace.Domain != nil {
				changes = append(changes, activitylog.BuildChangeField("domain", *workspace.Domain, ""))
			}
			workspace.Domain = nil
		}
	}

	workspace.UpdatedAt = time.Now()

	var logMeta map[string]interface{}
	var logDesc string
	var logSnap map[string]interface{}
	if len(changes) > 0 {
		actorName := s.getUserName(userID)
		logMeta = activitylog.WorkspaceUpdated(changes)
		logDesc = activitylog.GenerateDescription(actorName, logMeta)
		logSnap = activitylog.BuildWorkspaceSnapshot(workspace.Name)
	}

	if err := s.tm.Execute(func(tx *gorm.DB) error {
		if err := s.workspaceRepo.WithTx(tx).Update(workspace); err != nil {
			return apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to update workspace")
		}
		if len(changes) > 0 {
			s.logActivityInTx(tx, workspaceID, "", userID, workspaceID, models.ActivityActionUPDATE, logMeta, logDesc, logSnap)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &dto.UpdateWorkspaceResponse{
		ID:        workspace.ID,
		Name:      workspace.Name,
		Domain:    workspace.Domain,
		Plan:      string(workspace.Plan),
		UpdatedAt: workspace.UpdatedAt,
	}, nil
}

func (s *workspaceService) UpgradePlan(workspaceID string, userID string, req *dto.UpgradePlanRequest) (*dto.PlanUpgradeResponse, error) {
	workspace, err := s.workspaceRepo.GetByID(workspaceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrWorkspaceNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get workspace")
	}

	member, err := s.workspaceRepo.GetMember(workspaceID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrWorkspaceNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to verify membership")
	}
	if member.Role != models.WorkspaceRoleOWNER {
		return nil, apperror.ErrForbidden
	}

	newPlan := models.WorkspacePlan(req.NewPlan)
	switch newPlan {
	case models.WorkspacePlanFREE, models.WorkspacePlanPRO, models.WorkspacePlanENTERPRISE:
	default:
		return nil, apperror.ErrInvalidPlan
	}

	if workspace.Plan == newPlan {
		return nil, apperror.ErrAlreadyOnThisPlan
	}

	planRank := map[models.WorkspacePlan]int{
		models.WorkspacePlanFREE:       0,
		models.WorkspacePlanPRO:        1,
		models.WorkspacePlanENTERPRISE: 2,
	}
	if planRank[newPlan] < planRank[workspace.Plan] {
		return nil, apperror.ErrPlanDowngradeNotAllowed
	}

	prevPlan := workspace.Plan
	workspace.Plan = newPlan
	workspace.UpdatedAt = time.Now()

	actorName := s.getUserName(userID)
	logMeta := activitylog.PlanUpgraded(string(prevPlan), string(newPlan))
	logDesc := activitylog.GenerateDescription(actorName, logMeta)
	logSnap := activitylog.BuildWorkspaceSnapshot(workspace.Name)

	if err := s.tm.Execute(func(tx *gorm.DB) error {
		if err := s.workspaceRepo.WithTx(tx).Update(workspace); err != nil {
			return apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to upgrade plan")
		}
		s.logActivityInTx(tx, workspaceID, "", userID, workspaceID, models.ActivityActionUPDATE, logMeta, logDesc, logSnap)
		return nil
	}); err != nil {
		return nil, err
	}

	return &dto.PlanUpgradeResponse{
		ID:           workspace.ID,
		PreviousPlan: string(prevPlan),
		CurrentPlan:  string(workspace.Plan),
		UpgradedAt:   workspace.UpdatedAt,
	}, nil
}

func (s *workspaceService) DeleteWorkspace(workspaceID string, userID string, req *dto.DeleteWorkspaceRequest) (*dto.DeleteWorkspaceResponse, error) {
	workspace, err := s.workspaceRepo.GetByID(workspaceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrWorkspaceNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get workspace")
	}

	member, err := s.workspaceRepo.GetMember(workspaceID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrWorkspaceNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to verify membership")
	}
	if member.Role != models.WorkspaceRoleOWNER {
		return nil, apperror.ErrForbidden
	}

	if strings.TrimSpace(req.ConfirmationName) != strings.TrimSpace(workspace.Name) {
		return nil, apperror.ErrInvalidConfirmation
	}

	actorName := s.getUserName(userID)
	logMeta := activitylog.WorkspaceDeleted(workspace.Name)
	logDesc := activitylog.GenerateDescription(actorName, logMeta)
	logSnap := activitylog.BuildWorkspaceSnapshot(workspace.Name)

	if err := s.tm.Execute(func(tx *gorm.DB) error {
		if err := s.workspaceRepo.WithTx(tx).Delete(workspaceID); err != nil {
			return apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to delete workspace")
		}
		s.logActivityInTx(tx, workspaceID, "", userID, workspaceID, models.ActivityActionDELETE, logMeta, logDesc, logSnap)
		return nil
	}); err != nil {
		return nil, err
	}

	s.dispatcher.CascadeSoftDeleteWorkspace(workspaceID)

	return &dto.DeleteWorkspaceResponse{
		Message: fmt.Sprintf("Workspace '%s' has been deleted successfully.", workspace.Name),
	}, nil
}

// BR-ROLE-08: Seed 3 default roles when workspace is created
func (s *workspaceService) seedDefaultRoles(workspaceID string) error {
	type seedRole struct {
		Name        string
		Description string
		Permissions []string
	}

	allPerms := []string{
		"task:view", "task:create", "task:update", "task:delete", "task:assign", "task:move", "task:set_priority",
		"project:view", "project:update", "project:delete", "project:manage_members", "project:archive",
		"column:create", "column:update", "column:delete",
		"comment:create", "comment:update_own", "comment:delete_own", "comment:delete_any",
		"label:create", "label:update", "label:delete", "label:assign",
		"attachment:upload", "attachment:delete_own", "attachment:delete_any",
	}

	roles := []seedRole{
		{
			Name:        "Manager",
			Description: "Quản lý toàn diện project và thành viên",
			Permissions: allPerms,
		},
		{
			Name:        "Developer",
			Description: "Thành viên thực thi, quản lý task và công việc",
			Permissions: []string{
				"task:view", "task:create", "task:update", "task:delete", "task:assign", "task:move", "task:set_priority",
				"comment:create", "comment:update_own", "comment:delete_own",
				"attachment:upload", "attachment:delete_own",
				"label:assign",
			},
		},
		{
			Name:        "Viewer",
			Description: "Chỉ xem nội dung, không chỉnh sửa",
			Permissions: []string{
				"task:view",
				"comment:create", "comment:update_own", "comment:delete_own",
			},
		},
	}

	for _, r := range roles {
		desc := r.Description
		role := &models.Role{
			ID:          uuid.New().String(),
			WorkspaceID: workspaceID,
			Name:        r.Name,
			Description: &desc,
		}
		if _, err := s.roleRepo.Create(role); err != nil {
			return apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to create default role: "+r.Name)
		}
		if err := s.rolePermissionRepo.BulkCreate(role.ID, r.Permissions); err != nil {
			return apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to assign permissions to default role: "+r.Name)
		}
	}

	return nil
}

func (s *workspaceService) logActivity(workspaceID, projectID, userID, entityID string, action models.ActivityAction, metadata map[string]interface{}, description string, entitySnapshot map[string]interface{}) {
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
		UserID:         &uID,
		Action:         action,
		EntityType:     models.EntityTypeWORKSPACE,
		EntityID:       entityID,
		Description:    descPtr,
		Metadata:       metaStr,
		EntitySnapshot: snapStr,
	})
}

func (s *workspaceService) logActivityInTx(tx *gorm.DB, workspaceID, projectID, userID, entityID string, action models.ActivityAction, metadata map[string]interface{}, description string, entitySnapshot map[string]interface{}) {
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
		UserID:         &uID,
		Action:         action,
		EntityType:     models.EntityTypeWORKSPACE,
		EntityID:       entityID,
		Description:    descPtr,
		Metadata:       metaStr,
		EntitySnapshot: snapStr,
	}).Error
}

func (s *workspaceService) getUserName(userID string) string {
	u, err := s.userRepo.GetByID(userID)
	if err != nil {
		return userID
	}
	return u.FullName
}
