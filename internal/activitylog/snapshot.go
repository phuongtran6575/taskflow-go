package activitylog

import "fmt"

type TaskSnapshot struct {
	TaskRef     string `json:"task_ref"`
	TaskTitle   string `json:"task_title"`
	ProjectKey  string `json:"project_key"`
}

type ProjectSnapshot struct {
	ProjectName string `json:"project_name"`
	ProjectKey  string `json:"project_key"`
}

type ColumnSnapshot struct {
	ColumnTitle string `json:"column_title"`
	ProjectKey  string `json:"project_key"`
}

type WorkspaceSnapshot struct {
	WorkspaceName string `json:"workspace_name"`
}

type CommentSnapshot struct {
	TaskRef   string `json:"task_ref"`
	TaskTitle string `json:"task_title"`
}

func BuildTaskSnapshot(taskRef, taskTitle, projectKey string) map[string]interface{} {
	return map[string]interface{}{
		"task_ref":    taskRef,
		"task_title":  taskTitle,
		"project_key": projectKey,
	}
}

func BuildProjectSnapshot(projectName, projectKey string) map[string]interface{} {
	return map[string]interface{}{
		"project_name": projectName,
		"project_key":  projectKey,
	}
}

func BuildColumnSnapshot(columnTitle, projectKey string) map[string]interface{} {
	return map[string]interface{}{
		"column_title": columnTitle,
		"project_key":  projectKey,
	}
}

func BuildWorkspaceSnapshot(workspaceName string) map[string]interface{} {
	return map[string]interface{}{
		"workspace_name": workspaceName,
	}
}

func BuildCommentSnapshot(taskRef, taskTitle string) map[string]interface{} {
	return map[string]interface{}{
		"task_ref":   taskRef,
		"task_title": taskTitle,
	}
}

func Truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return fmt.Sprintf("%s...", string(runes[:maxLen]))
}
