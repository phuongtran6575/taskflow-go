package _interface

import "TaskFlow-Go/internal/dto"

type TaskBoardService interface {
	GetBoardData(workspaceID string, userID string, projectID string, priority []string, assigneeID []string, labelID []string, dueDateFrom string, dueDateTo string, creatorID string, hasAssignee *bool, hasLabel *bool, overdue *bool, search string, tasksPerColumn int) (*dto.BoardResponse, error)
	LoadMoreTasksInColumn(workspaceID string, userID string, projectID string, columnID string, cursor string, limit int, priority []string, assigneeID []string, labelID []string, dueDateFrom string, dueDateTo string, creatorID string, hasAssignee *bool, hasLabel *bool, overdue *bool, search string) (*dto.LoadMoreTasksResponse, error)
	MoveTask(workspaceID string, userID string, projectID string, taskID string, req *dto.MoveTaskRequest) (*dto.MoveTaskResponse, error)
}
