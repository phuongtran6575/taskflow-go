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

type TaskBoardHandler struct {
	boardService _interface.TaskBoardService
}

func NewTaskBoardHandler(boardService _interface.TaskBoardService) *TaskBoardHandler {
	return &TaskBoardHandler{boardService: boardService}
}

// GetBoardData
// @Summary      Get board data
// @Description  Get board view data with tasks grouped by column and filters
// @Tags         Board
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        priority query []string false "Filter by priorities"
// @Param        assignee_id query []string false "Filter by assignee IDs"
// @Param        label_id query []string false "Filter by label IDs"
// @Param        due_date_from query string false "Due date from (RFC3339)"
// @Param        due_date_to query string false "Due date to (RFC3339)"
// @Param        creator_id query string false "Filter by creator ID"
// @Param        has_assignee query bool false "Filter tasks with/without assignee"
// @Param        overdue query bool false "Filter overdue tasks"
// @Param        search query string false "Search keyword"
// @Param        tasks_per_column query int false "Tasks per column" default(20)
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/board [get]
func (h *TaskBoardHandler) GetBoardData(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")

	priority := c.QueryArray("priority")
	assigneeID := c.QueryArray("assignee_id")
	labelID := c.QueryArray("label_id")
	dueDateFrom := c.Query("due_date_from")
	dueDateTo := c.Query("due_date_to")
	creatorID := c.Query("creator_id")
	search := c.Query("search")

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

	tasksPerColumn, _ := strconv.Atoi(c.DefaultQuery("tasks_per_column", "20"))
	if tasksPerColumn < 1 {
		tasksPerColumn = 20
	}

	result, err := h.boardService.GetBoardData(workspaceID, userID, projectID, priority, assigneeID, labelID, dueDateFrom, dueDateTo, creatorID, hasAssignee, overdue, search, tasksPerColumn)
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

// LoadMoreTasksInColumn
// @Summary      Load more tasks in a column
// @Description  Cursor-based pagination for tasks in a specific column
// @Tags         Board
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        column_id path string true "Column ID"
// @Param        cursor query string false "Opaque cursor for pagination"
// @Param        limit query int false "Number of tasks" default(20)
// @Param        priority query []string false "Filter by priorities"
// @Param        assignee_id query []string false "Filter by assignee IDs"
// @Param        label_id query []string false "Filter by label IDs"
// @Param        due_date_from query string false "Due date from (RFC3339)"
// @Param        due_date_to query string false "Due date to (RFC3339)"
// @Param        creator_id query string false "Filter by creator ID"
// @Param        has_assignee query bool false "Filter tasks with/without assignee"
// @Param        overdue query bool false "Filter overdue tasks"
// @Param        search query string false "Search keyword"
// @Success      200  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/board/columns/{column_id}/tasks [get]
func (h *TaskBoardHandler) LoadMoreTasksInColumn(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	columnID := c.Param("column_id")

	cursor := c.Query("cursor")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit < 1 {
		limit = 20
	}

	priority := c.QueryArray("priority")
	assigneeID := c.QueryArray("assignee_id")
	labelID := c.QueryArray("label_id")
	dueDateFrom := c.Query("due_date_from")
	dueDateTo := c.Query("due_date_to")
	creatorID := c.Query("creator_id")
	search := c.Query("search")

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

	result, err := h.boardService.LoadMoreTasksInColumn(workspaceID, userID, projectID, columnID, cursor, limit, priority, assigneeID, labelID, dueDateFrom, dueDateTo, creatorID, hasAssignee, overdue, search)
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

// MoveTask
// @Summary      Move task
// @Description  Move a task to a different column or position
// @Tags         Board
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        task_id path string true "Task ID"
// @Param        request body dto.MoveTaskRequest true "Move details (target column and position)"
// @Success      200  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/tasks/{task_id}/move [patch]
func (h *TaskBoardHandler) MoveTask(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	taskID := c.Param("task_id")
	var req dto.MoveTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.boardService.MoveTask(workspaceID, userID, projectID, taskID, &req)
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
