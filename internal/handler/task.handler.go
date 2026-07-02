package handler

import (
	"TaskFlow-Go/internal/dto"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
	"TaskFlow-Go/internal/shared/appresponse"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type TaskHandler struct {
	taskService _interface.TaskService
}

func NewTaskHandler(taskService _interface.TaskService) *TaskHandler {
	return &TaskHandler{taskService: taskService}
}

// ListTasks
// @Summary      List tasks
// @Description  List tasks in a project with filters
// @Tags         Tasks
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Param        column_id query string false "Filter by column ID"
// @Param        priority query string false "Filter by priority"
// @Param        assignee_id query string false "Filter by assignee ID"
// @Param        label_id query string false "Filter by label ID"
// @Param        due_date_from query string false "Due date from (RFC3339)"
// @Param        due_date_to query string false "Due date to (RFC3339)"
// @Param        has_assignee query bool false "Filter tasks with/without assignee"
// @Param        search query string false "Search keyword"
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/tasks [get]
func (h *TaskHandler) ListTasks(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	param := parsePagination(c)
	columnID := c.Query("column_id")
	priority := c.Query("priority")
	assigneeID := c.Query("assignee_id")
	labelID := c.Query("label_id")
	dueDateFrom := c.Query("due_date_from")
	dueDateTo := c.Query("due_date_to")
	search := c.Query("search")

	var hasAssignee *bool
	if v := c.Query("has_assignee"); v != "" {
		b, _ := strconv.ParseBool(v)
		hasAssignee = &b
	}

	result, pagination, err := h.taskService.ListTasks(workspaceID, userID, projectID, columnID, priority, assigneeID, labelID, dueDateFrom, dueDateTo, hasAssignee, search, param.Page, param.Limit)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			appresponse.Fail(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		appresponse.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	appresponse.OKWithMeta(c, result, pagination)
}

// CreateTask
// @Summary      Create task
// @Description  Create a new task in a project
// @Tags         Tasks
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        request body dto.CreateTaskRequest true "Task details"
// @Success      201  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/tasks [post]
func (h *TaskHandler) CreateTask(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	var req dto.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.taskService.CreateTask(workspaceID, userID, projectID, &req)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			appresponse.Fail(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		appresponse.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	appresponse.Created(c, result)
}

// GetTaskDetails
// @Summary      Get task details
// @Description  Get details of a specific task
// @Tags         Tasks
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        task_id path string true "Task ID"
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Failure      404  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/tasks/{task_id} [get]
func (h *TaskHandler) GetTaskDetails(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	taskID := c.Param("task_id")

	result, err := h.taskService.GetTaskById(workspaceID, userID, projectID, taskID)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			appresponse.Fail(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		appresponse.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	appresponse.OK(c, result)
}

// UpdateTask
// @Summary      Update task
// @Description  Update a task's details
// @Tags         Tasks
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        task_id path string true "Task ID"
// @Param        request body dto.UpdateTaskRequest true "Task update data"
// @Success      200  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Failure      404  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/tasks/{task_id} [patch]
func (h *TaskHandler) UpdateTask(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	taskID := c.Param("task_id")
	var req dto.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.taskService.UpdateTask(workspaceID, userID, projectID, taskID, &req)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			appresponse.Fail(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		appresponse.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	appresponse.OK(c, result)
}

// DeleteTask
// @Summary      Delete task
// @Description  Permanently delete a task
// @Tags         Tasks
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        task_id path string true "Task ID"
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Failure      404  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/tasks/{task_id} [delete]
func (h *TaskHandler) DeleteTask(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	taskID := c.Param("task_id")

	result, err := h.taskService.DeleteTask(workspaceID, userID, projectID, taskID)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			appresponse.Fail(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		appresponse.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	appresponse.OK(c, result)
}

// CreateSubtask
// @Summary      Create subtask
// @Description  Create a subtask under an existing task
// @Tags         Tasks
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        task_id path string true "Parent task ID"
// @Param        request body dto.CreateSubtaskRequest true "Subtask details"
// @Success      201  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/tasks/{task_id}/subtasks [post]
func (h *TaskHandler) CreateSubtask(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	taskID := c.Param("task_id")
	var req dto.CreateSubtaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.taskService.CreateSubtask(workspaceID, userID, projectID, taskID, &req)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			appresponse.Fail(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		appresponse.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	appresponse.Created(c, result)
}

// ListSubtasks
// @Summary      List subtasks
// @Description  List all subtasks of a task
// @Tags         Tasks
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        task_id path string true "Parent task ID"
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/tasks/{task_id}/subtasks [get]
func (h *TaskHandler) ListSubtasks(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	taskID := c.Param("task_id")

	result, err := h.taskService.ListSubtasks(workspaceID, userID, projectID, taskID)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			appresponse.Fail(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		appresponse.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	appresponse.OK(c, result)
}

// GetMyTasks
// @Summary      Get my tasks
// @Description  Get tasks assigned to the current user in a workspace
// @Tags         Tasks
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Param        priority query string false "Filter by priority"
// @Param        project_id query string false "Filter by project ID"
// @Param        column_id query string false "Filter by column ID"
// @Param        due_date_from query string false "Due date from (RFC3339)"
// @Param        due_date_to query string false "Due date to (RFC3339)"
// @Param        overdue query bool false "Filter overdue tasks"
// @Param        include_archived query bool false "Include archived projects" default(false)
// @Param        include_subtasks query bool false "Include subtasks" default(true)
// @Param        search query string false "Search keyword"
// @Param        sort_by query string false "Sort field" default(due_date)
// @Param        sort_dir query string false "Sort direction" default(asc)
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/my-tasks [get]
func (h *TaskHandler) GetMyTasks(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	param := parsePagination(c)
	priority := c.Query("priority")
	projectID := c.Query("project_id")
	columnID := c.Query("column_id")
	dueDateFrom := c.Query("due_date_from")
	dueDateTo := c.Query("due_date_to")
	search := c.Query("search")
	sortBy := c.DefaultQuery("sort_by", "due_date")
	sortDir := c.DefaultQuery("sort_dir", "asc")

	var overdue *bool
	if v := c.Query("overdue"); v != "" {
		b, _ := strconv.ParseBool(v)
		overdue = &b
	}
	includeArchived, _ := strconv.ParseBool(c.DefaultQuery("include_archived", "false"))
	includeSubtasks, _ := strconv.ParseBool(c.DefaultQuery("include_subtasks", "true"))

	result, err := h.taskService.GetMyTasks(workspaceID, userID, priority, projectID, columnID, dueDateFrom, dueDateTo, overdue, includeArchived, includeSubtasks, search, sortBy, sortDir, param.Page, param.Limit)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			appresponse.Fail(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		appresponse.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	appresponse.OK(c, result)
}

// SearchTasks
// @Summary      Search tasks
// @Description  Search tasks with advanced filters
// @Tags         Tasks
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Param        search query string false "Search keyword"
// @Param        priority query string false "Filter by priority"
// @Param        assignee_id query string false "Filter by assignee ID"
// @Param        label_id query string false "Filter by label ID"
// @Param        creator_id query string false "Filter by creator ID"
// @Param        column_id query string false "Filter by column ID"
// @Param        due_date_from query string false "Due date from (RFC3339)"
// @Param        due_date_to query string false "Due date to (RFC3339)"
// @Param        has_assignee query bool false "Filter tasks with/without assignee"
// @Param        overdue query bool false "Filter overdue tasks"
// @Param        include_subtasks query bool false "Include subtasks in results" default(false)
// @Param        sort_by query string false "Sort field" default(position)
// @Param        sort_dir query string false "Sort direction" default(asc)
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/tasks/search [get]
func (h *TaskHandler) SearchTasks(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	param := parsePagination(c)
	search := c.Query("search")
	priority := c.Query("priority")
	assigneeID := c.Query("assignee_id")
	labelID := c.Query("label_id")
	creatorID := c.Query("creator_id")
	columnID := c.Query("column_id")
	dueDateFrom := c.Query("due_date_from")
	dueDateTo := c.Query("due_date_to")
	sortBy := c.DefaultQuery("sort_by", "position")
	sortDir := c.DefaultQuery("sort_dir", "asc")

	var hasAssignee *bool
	if v := c.Query("has_assignee"); v != "" {
		b, _ := strconv.ParseBool(v)
		hasAssignee = &b
	}
	var overdue *bool
	if v := c.Query("overdue"); v != "" {
		b, _ := strconv.ParseBool(v)
		overdue = &b
	}
	includeSubtasks, _ := strconv.ParseBool(c.DefaultQuery("include_subtasks", "false"))

	result, err := h.taskService.SearchTasks(workspaceID, userID, projectID, search, priority, assigneeID, labelID, creatorID, columnID, dueDateFrom, dueDateTo, hasAssignee, overdue, includeSubtasks, sortBy, sortDir, param.Page, param.Limit)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			appresponse.Fail(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		appresponse.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	appresponse.OK(c, result)
}
