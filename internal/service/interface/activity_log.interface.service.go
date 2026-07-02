package _interface

import "TaskFlow-Go/internal/dto"

type ActivityLogService interface {
	ListWorkspaceActivity(workspaceID string, userID string, filters map[string][]string, limit int, cursor string) (*dto.ActivityLogListResponse, error)
	ListProjectActivity(workspaceID string, userID string, projectID string, filters map[string][]string, limit int, cursor string) (*dto.ActivityLogListResponse, error)
	ListTaskTimeline(workspaceID string, userID string, projectID string, taskID string, includeComments bool, limit int, cursor string, direction string) (*dto.TaskTimelineResponse, error)
	ExportWorkspaceActivity(workspaceID string, userID string, dateFrom string, dateTo string, format string) ([]byte, string, error)
}
