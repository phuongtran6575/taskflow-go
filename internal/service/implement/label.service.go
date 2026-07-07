package implement

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/activitylog"
	"TaskFlow-Go/internal/database"
	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
)

type labelService struct {
	tm              *database.TransactionManager
	labelRepo       repoInterface.LabelRepository
	taskLabelRepo   repoInterface.TaskLabelRepository
	projectRepo     repoInterface.ProjectRepository
	taskRepo        repoInterface.TaskRepository
	activityLogRepo repoInterface.ActivityLogRepository
	userRepo        repoInterface.UserRepository
}

func NewLabelService(
	tm *database.TransactionManager,
	labelRepo repoInterface.LabelRepository,
	taskLabelRepo repoInterface.TaskLabelRepository,
	projectRepo repoInterface.ProjectRepository,
	taskRepo repoInterface.TaskRepository,
	activityLogRepo repoInterface.ActivityLogRepository,
	userRepo repoInterface.UserRepository,
) _interface.LabelService {
	return &labelService{
		tm:              tm,
		labelRepo:       labelRepo,
		taskLabelRepo:   taskLabelRepo,
		projectRepo:     projectRepo,
		taskRepo:        taskRepo,
		activityLogRepo: activityLogRepo,
		userRepo:        userRepo,
	}
}

func normalizeColor(color string) string {
	color = strings.ToUpper(color)
	if len(color) == 4 {
		color = "#" + string(color[1]) + string(color[1]) + string(color[2]) + string(color[2]) + string(color[3]) + string(color[3])
	}
	return color
}

func (s *labelService) logActivityInTx(tx *gorm.DB, workspaceID, projectID, userID, entityID string, action models.ActivityAction, metadata map[string]interface{}, description string, entitySnapshot map[string]interface{}) {
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
		WorkspaceID:    &workspaceID,
		ProjectID:      &projectID,
		UserID:         &userID,
		Action:         action,
		EntityType:     models.EntityTypeLABEL,
		EntityID:       entityID,
		Description:    descPtr,
		Metadata:       metaStr,
		EntitySnapshot: snapStr,
	}).Error
}

func (s *labelService) getUserName(userID string) string {
	u, err := s.userRepo.GetByID(userID)
	if err != nil {
		return userID
	}
	return u.FullName
}

func (s *labelService) logActivity(workspaceID, projectID, userID, entityID string, action models.ActivityAction, metadata map[string]interface{}, description string, entitySnapshot map[string]interface{}) {
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
		WorkspaceID:    &workspaceID,
		ProjectID:      &projectID,
		UserID:         &userID,
		Action:         action,
		EntityType:     models.EntityTypeLABEL,
		EntityID:       entityID,
		Description:    descPtr,
		Metadata:       metaStr,
		EntitySnapshot: snapStr,
	})
}

func (s *labelService) getProjectOrFail(workspaceID, projectID string) (*models.Project, error) {
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

func (s *labelService) getTaskOrFail(projectID, taskID string) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrTaskNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get task")
	}
	if task.ProjectID != projectID || task.DeletedAt.Valid {
		return nil, apperror.ErrTaskNotFound
	}
	return task, nil
}

func (s *labelService) ListProjectLabels(workspaceID string, userID string, projectID string, search string, withTaskCount bool) (*dto.LabelListResponse, error) {
	_, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}

	labels, err := s.labelRepo.ListByProjectIDWithCount(projectID, search, withTaskCount)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list labels")
	}

	return &dto.LabelListResponse{
		Data:  labels,
		Total: len(labels),
	}, nil
}

func (s *labelService) CreateLabel(workspaceID string, userID string, projectID string, req *dto.CreateLabelRequest) (*dto.LabelCreateResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" || req.Color == "" {
		return nil, apperror.ErrValidation
	}

	if !hexColorRegex.MatchString(req.Color) {
		return nil, apperror.ErrInvalidColor
	}

	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if project.IsArchived {
		return nil, apperror.ErrProjectArchived
	}

	exists, err := s.labelRepo.ExistsByNameInProject(projectID, name, "")
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check label name")
	}
	if exists {
		return nil, apperror.ErrLabelNameAlreadyExists
	}

	count, err := s.labelRepo.CountByProjectID(projectID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to count labels")
	}
	if count >= 50 {
		return nil, apperror.ErrLabelLimitReached
	}

	color := normalizeColor(req.Color)
	label := models.Label{
		Name:      name,
		Color:     color,
		ProjectID: projectID,
	}

	actorName := s.getUserName(userID)
	meta := map[string]interface{}{
		"event":       "label_created",
		"label_name":  label.Name,
		"label_color": label.Color,
	}
	desc := activitylog.GenerateDescription(actorName, meta)

	err = s.tm.Execute(func(tx *gorm.DB) error {
		if err := tx.Create(&label).Error; err != nil {
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create label")
		}
		s.logActivityInTx(tx, workspaceID, projectID, userID, label.ID, models.ActivityActionCREATE, meta, desc, nil)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &dto.LabelCreateResponse{
		ID:        label.ID,
		Name:      label.Name,
		Color:     label.Color,
		TaskCount: 0,
		CreatedAt: label.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *labelService) UpdateLabel(workspaceID string, userID string, projectID string, labelID string, req *dto.UpdateLabelRequest) (*dto.LabelUpdateResponse, error) {
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		return nil, apperror.ErrValidation
	}
	if req.Color != nil && !hexColorRegex.MatchString(*req.Color) {
		return nil, apperror.ErrInvalidColor
	}

	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if project.IsArchived {
		return nil, apperror.ErrProjectArchived
	}

	label, err := s.labelRepo.GetByID(labelID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrLabelNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get label")
	}
	if label.ProjectID != projectID {
		return nil, apperror.ErrLabelNotFound
	}

	var changes []map[string]interface{}

	if req.Name != nil {
		newName := strings.TrimSpace(*req.Name)
		if newName != label.Name {
			exists, err := s.labelRepo.ExistsByNameInProject(projectID, newName, labelID)
			if err != nil {
				return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check label name")
			}
			if exists {
				return nil, apperror.ErrLabelNameAlreadyExists
			}
			changes = append(changes, map[string]interface{}{
				"field": "name",
				"old":   label.Name,
				"new":   newName,
			})
			label.Name = newName
		}
	}
	if req.Color != nil {
		newColor := normalizeColor(*req.Color)
		if newColor != label.Color {
			changes = append(changes, map[string]interface{}{
				"field": "color",
				"old":   label.Color,
				"new":   newColor,
			})
			label.Color = newColor
		}
	}

	if len(changes) == 0 {
		taskLabels, _ := s.taskLabelRepo.ListByLabelID(labelID)
		return &dto.LabelUpdateResponse{
			ID:        label.ID,
			Name:      label.Name,
			Color:     label.Color,
			TaskCount: len(taskLabels),
			UpdatedAt: label.UpdatedAt.Format(time.RFC3339),
		}, nil
	}

	actorName := s.getUserName(userID)
	meta := map[string]interface{}{
		"event":    "label_updated",
		"label_id": labelID,
		"changes":  changes,
	}
	desc := activitylog.GenerateDescription(actorName, meta)

	err = s.tm.Execute(func(tx *gorm.DB) error {
		label.UpdatedAt = time.Now()
		if err := tx.Save(&label).Error; err != nil {
			return apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update label")
		}
		s.logActivityInTx(tx, workspaceID, projectID, userID, labelID, models.ActivityActionUPDATE, meta, desc, nil)
		return nil
	})
	if err != nil {
		return nil, err
	}

	taskLabels, _ := s.taskLabelRepo.ListByLabelID(labelID)
	taskCount := len(taskLabels)

	return &dto.LabelUpdateResponse{
		ID:        label.ID,
		Name:      label.Name,
		Color:     label.Color,
		TaskCount: taskCount,
		UpdatedAt: label.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (s *labelService) DeleteLabel(workspaceID string, userID string, projectID string, labelID string) (*dto.LabelDeleteResponse, error) {
	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if project.IsArchived {
		return nil, apperror.ErrProjectArchived
	}

	label, err := s.labelRepo.GetByID(labelID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrLabelNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get label")
	}
	if label.ProjectID != projectID {
		return nil, apperror.ErrLabelNotFound
	}

	var affectedCount int64
	actorName := s.getUserName(userID)
	meta := map[string]interface{}{
		"event":                "label_deleted",
		"label_name":           label.Name,
		"label_color":          label.Color,
		"affected_tasks_count": affectedCount,
	}
	desc := activitylog.GenerateDescription(actorName, meta)

	err = s.tm.Execute(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.TaskLabel{}).Where("label_id = ?", labelID).Count(&count).Error; err != nil {
			return err
		}
		affectedCount = count

		if err := tx.Delete(&models.Label{}, "id = ?", labelID).Error; err != nil {
			return err
		}
		s.logActivityInTx(tx, workspaceID, projectID, userID, labelID, models.ActivityActionDELETE, meta, desc, nil)
		return nil
	})
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete label")
	}

	return &dto.LabelDeleteResponse{
		Message:            fmt.Sprintf("Label '%s' has been deleted.", label.Name),
		DeletedLabelID:     labelID,
		AffectedTasksCount: int(affectedCount),
	}, nil
}

func (s *labelService) ListTaskLabels(workspaceID string, userID string, projectID string, taskID string) (*dto.TaskLabelListResponse, error) {
	_, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}

	_, err = s.getTaskOrFail(projectID, taskID)
	if err != nil {
		return nil, err
	}

	result, err := s.labelRepo.ListByTaskIDWithRef(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrTaskNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list task labels")
	}
	return result, nil
}

func (s *labelService) AssignLabelsToTask(workspaceID string, userID string, projectID string, taskID string, req *dto.AssignLabelsRequest) (*dto.AssignLabelsResponse, error) {
	if len(req.LabelIDs) == 0 {
		return nil, apperror.ErrLabelIDsRequired
	}

	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if project.IsArchived {
		return nil, apperror.ErrProjectArchived
	}

	task, err := s.getTaskOrFail(projectID, taskID)
	if err != nil {
		return nil, err
	}

	projectLabels, err := s.labelRepo.ListByProjectID(projectID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get project labels")
	}
	validLabelIDs := make(map[string]struct{}, len(projectLabels))
	for _, l := range projectLabels {
		validLabelIDs[l.ID] = struct{}{}
	}
	var invalidIDs []string
	for _, lid := range req.LabelIDs {
		if _, ok := validLabelIDs[lid]; !ok {
			invalidIDs = append(invalidIDs, lid)
		}
	}
	if len(invalidIDs) > 0 {
		return nil, &apperror.InvalidLabelIDsError{
			AppError:    apperror.NewAppError(http.StatusBadRequest, "INVALID_LABEL_IDS", "One or more labels do not belong to this project"),
			InvalidIDs: invalidIDs,
		}
	}

	var addedIDs []string
	var skippedIDs []string
	err = s.tm.Execute(func(tx *gorm.DB) error {
		var existingLabelIDs []string
		tx.Model(&models.TaskLabel{}).Where("task_id = ?", taskID).Pluck("label_id", &existingLabelIDs)

		existingSet := make(map[string]struct{}, len(existingLabelIDs))
		for _, lid := range existingLabelIDs {
			existingSet[lid] = struct{}{}
		}

		current := len(existingLabelIDs)
		toAdd := 0
		for _, lid := range req.LabelIDs {
			if _, ok := existingSet[lid]; !ok {
				toAdd++
			}
		}
		if current+toAdd > 10 {
			return &apperror.TaskLabelLimitError{
				AppError: apperror.NewAppError(http.StatusBadRequest, "LABEL_LIMIT_REACHED", "Maximum 10 labels per task"),
				Current:  current,
				Limit:    10,
				CanAdd:   10 - current,
			}
		}

		for _, lid := range req.LabelIDs {
			if _, ok := existingSet[lid]; ok {
				skippedIDs = append(skippedIDs, lid)
				continue
			}
			if err := tx.Create(&models.TaskLabel{TaskID: taskID, LabelID: lid}).Error; err != nil {
				return err
			}
			addedIDs = append(addedIDs, lid)
		}
		return nil
	})
	if err != nil {
		var taskLabelLimitErr *apperror.TaskLabelLimitError
		if errors.As(err, &taskLabelLimitErr) {
			return nil, taskLabelLimitErr
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to assign labels")
	}

	labelMap := make(map[string]models.Label, len(projectLabels))
	for _, l := range projectLabels {
		labelMap[l.ID] = l
	}
	added := make([]dto.AddedLabelInfo, 0, len(addedIDs))
	for _, id := range addedIDs {
		if l, ok := labelMap[id]; ok {
			added = append(added, dto.AddedLabelInfo{
				ID: l.ID, Name: l.Name, Color: l.Color,
			})
		}
	}

	allAfter, _ := s.taskLabelRepo.ListTaskLabelsByTaskID(taskID)
	ref := fmt.Sprintf("%s-%d", project.Key, task.TaskNumber)

	return &dto.AssignLabelsResponse{
		TaskID:                taskID,
		TaskRef:               ref,
		Added:                 added,
		SkippedAlreadyAssigned: skippedIDs,
		TotalLabelsAfter:      len(allAfter),
	}, nil
}

func (s *labelService) RemoveLabelsFromTask(workspaceID string, userID string, projectID string, taskID string, req *dto.RemoveLabelsRequest) (*dto.RemoveLabelsResponse, error) {
	if len(req.LabelIDs) == 0 {
		return nil, apperror.ErrLabelIDsRequired
	}

	project, err := s.getProjectOrFail(workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	if project.IsArchived {
		return nil, apperror.ErrProjectArchived
	}

	task, err := s.getTaskOrFail(projectID, taskID)
	if err != nil {
		return nil, err
	}

	existingLabels, err := s.taskLabelRepo.ListTaskLabelsByTaskID(taskID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get existing labels")
	}
	existingSet := make(map[string]struct{}, len(existingLabels))
	for _, tl := range existingLabels {
		existingSet[tl.LabelID] = struct{}{}
	}

	projectLabels, err := s.labelRepo.ListByProjectID(projectID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get project labels")
	}
	labelInfoMap := make(map[string]models.Label, len(projectLabels))
	for _, l := range projectLabels {
		labelInfoMap[l.ID] = l
	}

	var removed []dto.RemovedLabelInfo
	var skippedIDs []string
	for _, lid := range req.LabelIDs {
		if _, ok := existingSet[lid]; !ok {
			skippedIDs = append(skippedIDs, lid)
			continue
		}
		if err := s.labelRepo.RemoveFromTask(taskID, lid); err != nil {
			return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to remove label")
		}
		if l, ok := labelInfoMap[lid]; ok {
			removed = append(removed, dto.RemovedLabelInfo{
				ID: l.ID, Name: l.Name, Color: l.Color,
			})
		}
	}

	allAfter, _ := s.taskLabelRepo.ListTaskLabelsByTaskID(taskID)
	ref := fmt.Sprintf("%s-%d", project.Key, task.TaskNumber)

	return &dto.RemoveLabelsResponse{
		TaskID:             taskID,
		TaskRef:            ref,
		Removed:            removed,
		SkippedNotAssigned: skippedIDs,
		TotalLabelsAfter:   len(allAfter),
	}, nil
}
