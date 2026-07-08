package implement

import (
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/helper"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
)

type activityLogService struct {
	activityLogRepo repoInterface.ActivityLogRepository
	projectRepo     repoInterface.ProjectRepository
	taskRepo        repoInterface.TaskRepository
	commentRepo     repoInterface.CommentRepository
	workspaceRepo   repoInterface.WorkspaceRepository
}

func NewActivityLogService(
	activityLogRepo repoInterface.ActivityLogRepository,
	projectRepo repoInterface.ProjectRepository,
	taskRepo repoInterface.TaskRepository,
	commentRepo repoInterface.CommentRepository,
	workspaceRepo repoInterface.WorkspaceRepository,
) _interface.ActivityLogService {
	return &activityLogService{
		activityLogRepo: activityLogRepo,
		projectRepo:     projectRepo,
		taskRepo:        taskRepo,
		commentRepo:     commentRepo,
		workspaceRepo:   workspaceRepo,
	}
}

func (s *activityLogService) ListWorkspaceActivity(workspaceID string, userID string, filters map[string][]string, limit int, cursor string) (*dto.ActivityLogListResponse, error) {
	return s.activityLogRepo.ListByWorkspace(workspaceID, filters, limit, cursor)
}

func (s *activityLogService) ListProjectActivity(workspaceID string, userID string, projectID string, filters map[string][]string, limit int, cursor string) (*dto.ActivityLogListResponse, error) {
	_, err := s.projectRepo.GetByID(projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrProjectNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get project")
	}

	return s.activityLogRepo.ListByProject(projectID, filters, limit, cursor)
}

func (s *activityLogService) ListTaskTimeline(workspaceID string, userID string, projectID string, taskID string, includeComments bool, limit int, cursor string, direction string) (*dto.TaskTimelineResponse, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrTaskNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get task")
	}
	if task.ProjectID != projectID {
		return nil, apperror.ErrTaskNotFound
	}

	taskRef := fmt.Sprintf("%s-%d", "KEY", task.TaskNumber)
	if proj, err := s.projectRepo.GetByID(projectID); err == nil {
		taskRef = fmt.Sprintf("%s-%d", proj.Key, task.TaskNumber)
	}

	activityResp, err := s.activityLogRepo.ListByTask(taskID, limit, cursor, direction)
	if err != nil {
		return nil, err
	}

	if !includeComments || activityResp == nil {
		entries := make([]interface{}, len(activityResp.Data))
		for i, item := range activityResp.Data {
			entry := dto.TimelineActivityEntry{
				EntryType:   "activity",
				ID:          item.ID,
				Action:      item.Action,
				EntityType:  item.EntityType,
				Description: item.Description,
				Actor:       item.Actor,
				Metadata:    item.Metadata,
				CreatedAt:   item.CreatedAt,
			}
			entries[i] = entry
		}
		return &dto.TaskTimelineResponse{
			TaskID:     taskID,
			TaskRef:    taskRef,
			Data:       entries,
			HasMore:    activityResp.HasMore,
			NextCursor: activityResp.NextCursor,
		}, nil
	}

	commentResp, err := s.commentRepo.ListByTaskIDWithCursor(taskID, limit, cursor, direction)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list comments")
	}

	entries := helper.MergeTimeline(activityResp.Data, commentResp.Data, direction)
	hasMore := activityResp.HasMore || commentResp.HasMore
	nextCursor := activityResp.NextCursor
	if commentResp.NextCursor != nil {
		nextCursor = commentResp.NextCursor
	}

	interfaceEntries := make([]interface{}, len(entries))
	copy(interfaceEntries, entries)

	return &dto.TaskTimelineResponse{
		TaskID:     taskID,
		TaskRef:    taskRef,
		Data:       interfaceEntries,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	}, nil
}

func (s *activityLogService) ExportWorkspaceActivity(workspaceID string, userID string, dateFrom string, dateTo string, format string) ([]byte, string, error) {
	if dateFrom == "" || dateTo == "" {
		return nil, "", apperror.NewAppError(http.StatusBadRequest, "DATE_RANGE_REQUIRED", "date_from and date_to are required")
	}
	from, err := time.Parse("2006-01-02", dateFrom)
	if err != nil {
		return nil, "", apperror.NewAppError(http.StatusBadRequest, "VALIDATION_ERROR", "Invalid date_from format")
	}
	to, err := time.Parse("2006-01-02", dateTo)
	if err != nil {
		return nil, "", apperror.NewAppError(http.StatusBadRequest, "VALIDATION_ERROR", "Invalid date_to format")
	}
	if to.Sub(from) > 365*24*time.Hour {
		return nil, "", apperror.NewAppError(http.StatusBadRequest, "DATE_RANGE_TOO_LARGE", "Date range cannot exceed 1 year")
	}

	filters := map[string][]string{
		"date_from": {dateFrom},
		"date_to":   {dateTo},
	}
	resp, err := s.activityLogRepo.ListByWorkspace(workspaceID, filters, 10000, "")
	if err != nil {
		return nil, "", apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to export activity")
	}

	var buf strings.Builder
	writer := csv.NewWriter(&buf)
	writer.Write([]string{"id", "action", "entity_type", "entity_id", "actor_name", "actor_email", "project_name", "description", "created_at"})

	for _, item := range resp.Data {
		actorEmail := ""
		writer.Write([]string{
			item.ID,
			item.Action,
			item.EntityType,
			item.EntityID,
			item.Actor.FullName,
			actorEmail,
			helper.SafeProjectName(item.Project),
			item.Description,
			item.CreatedAt,
		})
	}
	writer.Flush()

	ws, _ := s.workspaceRepo.GetByID(workspaceID)
	filename := fmt.Sprintf("activity_%s_%s.csv", helper.SafeStr2(ws, "workspace"), dateFrom)
	return []byte(buf.String()), filename, nil
}
