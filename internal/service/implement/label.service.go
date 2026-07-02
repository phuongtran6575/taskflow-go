package implement

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
)

type labelService struct {
	labelRepo     repoInterface.LabelRepository
	taskLabelRepo repoInterface.TaskLabelRepository
	projectRepo   repoInterface.ProjectRepository
	taskRepo      repoInterface.TaskRepository
}

func NewLabelService(
	labelRepo repoInterface.LabelRepository,
	taskLabelRepo repoInterface.TaskLabelRepository,
	projectRepo repoInterface.ProjectRepository,
	taskRepo repoInterface.TaskRepository,
) _interface.LabelService {
	return &labelService{
		labelRepo:     labelRepo,
		taskLabelRepo: taskLabelRepo,
		projectRepo:   projectRepo,
		taskRepo:      taskRepo,
	}
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
	if req.Name == "" || req.Color == "" {
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

	exists, err := s.labelRepo.ExistsByNameInProject(projectID, req.Name, "")
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

	label := models.Label{
		Name:      req.Name,
		Color:     req.Color,
		ProjectID: projectID,
	}
	if err := s.labelRepo.Create(&label); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create label")
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
	if req.Name != nil && *req.Name == "" {
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

	if req.Name != nil && *req.Name != label.Name {
		exists, err := s.labelRepo.ExistsByNameInProject(projectID, *req.Name, labelID)
		if err != nil {
			return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check label name")
		}
		if exists {
			return nil, apperror.ErrLabelNameAlreadyExists
		}
		label.Name = *req.Name
	}
	if req.Color != nil {
		label.Color = *req.Color
	}

	label.UpdatedAt = time.Now()
	if err := s.labelRepo.Update(label); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update label")
	}

	taskLabels, err := 	s.taskLabelRepo.ListByLabelID(labelID)
	taskCount := 0
	if err == nil {
		taskCount = len(taskLabels)
	}

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

	taskLabels, err := 	s.taskLabelRepo.ListByLabelID(labelID)
	affectedCount := 0
	if err == nil {
		affectedCount = len(taskLabels)
	}

	if err := s.labelRepo.Delete(labelID); err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete label")
	}

	return &dto.LabelDeleteResponse{
		Message:            fmt.Sprintf("Label '%s' has been deleted.", label.Name),
		DeletedLabelID:     labelID,
		AffectedTasksCount: affectedCount,
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

	projectLabels, err := 	s.labelRepo.ListByProjectID(projectID)
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
		return nil, apperror.NewAppError(http.StatusBadRequest, "INVALID_LABEL_IDS",
			fmt.Sprintf("Labels do not belong to this project: %v", invalidIDs))
	}

	existingLabels, err := 	s.taskLabelRepo.ListTaskLabelsByTaskID(taskID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get existing labels")
	}
	existingSet := make(map[string]struct{}, len(existingLabels))
	for _, tl := range existingLabels {
		existingSet[tl.LabelID] = struct{}{}
	}
	if len(existingLabels)+len(req.LabelIDs) > 10 {
		return nil, apperror.ErrTaskLabelLimitReached
	}

	var addedIDs []string
	var skippedIDs []string
	for _, lid := range req.LabelIDs {
		if _, ok := existingSet[lid]; ok {
			skippedIDs = append(skippedIDs, lid)
			continue
		}
		if err := s.labelRepo.AddToTask(taskID, lid); err != nil {
			return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to assign label")
		}
		addedIDs = append(addedIDs, lid)
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

	allAfter, _ := 	s.taskLabelRepo.ListTaskLabelsByTaskID(taskID)
	ref := fmt.Sprintf("%s-%d", project.Key, task.TaskNumber)

	return &dto.AssignLabelsResponse{
		TaskID:               taskID,
		TaskRef:              ref,
		Added:                added,
		SkippedAlreadyAssigned: skippedIDs,
		TotalLabelsAfter:     len(allAfter),
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

	existingLabels, err := 	s.taskLabelRepo.ListTaskLabelsByTaskID(taskID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get existing labels")
	}
	existingSet := make(map[string]struct{}, len(existingLabels))
	for _, tl := range existingLabels {
		existingSet[tl.LabelID] = struct{}{}
	}

	projectLabels, err := 	s.labelRepo.ListByProjectID(projectID)
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

	allAfter, _ := 	s.taskLabelRepo.ListTaskLabelsByTaskID(taskID)
	ref := fmt.Sprintf("%s-%d", project.Key, task.TaskNumber)

	return &dto.RemoveLabelsResponse{
		TaskID:             taskID,
		TaskRef:            ref,
		Removed:            removed,
		SkippedNotAssigned: skippedIDs,
		TotalLabelsAfter:   len(allAfter),
	}, nil
}
