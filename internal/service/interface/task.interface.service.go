package _interface

import "TaskFlow-Go/internal/dto"

type TaskService interface {
	ListTasks(workspaceID string, userID string, projectID string, columnID string, priority string, assigneeID string, labelID string, dueDateFrom string, dueDateTo string, hasAssignee *bool, hasLabel *bool, search string, page int, limit int) ([]dto.TaskSummary, *dto.Pagination, error)
	CreateTask(workspaceID string, userID string, projectID string, req *dto.CreateTaskRequest) (*dto.TaskCreateResponse, error)
	GetTaskById(workspaceID string, userID string, projectID string, taskID string) (*dto.TaskDetailResponse, error)
	UpdateTask(workspaceID string, userID string, projectID string, taskID string, req *dto.UpdateTaskRequest) (*dto.UpdateTaskResponse, error)
	DeleteTask(workspaceID string, userID string, projectID string, taskID string) (*dto.TaskDeleteResponse, error)
	CreateSubtask(workspaceID string, userID string, projectID string, taskID string, req *dto.CreateSubtaskRequest) (*dto.SubtaskCreateResponse, error)
	ListSubtasks(workspaceID string, userID string, projectID string, taskID string) (*dto.SubtaskListResponse, error)
	GetMyTasks(workspaceID string, userID string, priority string, projectID string, columnID string, dueDateFrom string, dueDateTo string, overdue *bool, includeArchived bool, includeSubtasks bool, search string, sortBy string, sortDir string, page int, limit int) (*dto.MyTaskListResponse, error)
	SearchTasks(workspaceID string, userID string, projectID string, search string, priority string, assigneeID string, labelID string, creatorID string, columnID string, dueDateFrom string, dueDateTo string, hasAssignee *bool, hasLabel *bool, overdue *bool, includeSubtasks bool, sortBy string, sortDir string, page int, limit int) (*dto.TaskSearchResponse, error)
}
