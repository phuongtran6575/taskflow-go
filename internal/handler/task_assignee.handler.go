package handler

import (
	"TaskFlow-Go/internal/dto"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
	"TaskFlow-Go/internal/shared/appresponse"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type TaskAssigneeHandler struct {
	assigneeService _interface.TaskAssigneeService
}

func NewTaskAssigneeHandler(assigneeService _interface.TaskAssigneeService) *TaskAssigneeHandler {
	return &TaskAssigneeHandler{assigneeService: assigneeService}
}

// ListAssignees
// @Summary      List task assignees
// @Description  List all assignees of a task
// @Tags         Task Assignees
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        task_id path string true "Task ID"
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/tasks/{task_id}/assignees [get]
func (h *TaskAssigneeHandler) ListAssignees(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	taskID := c.Param("task_id")

	result, err := h.assigneeService.ListAssignees(workspaceID, userID, projectID, taskID)
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

// GetAvailableAssignees
// @Summary      Get available assignees
// @Description  Get project members available to be assigned to a task
// @Tags         Task Assignees
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        task_id path string true "Task ID"
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Param        search query string false "Search keyword"
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/tasks/{task_id}/assignees/available [get]
func (h *TaskAssigneeHandler) GetAvailableAssignees(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	taskID := c.Param("task_id")
	param := parsePagination(c)
	search := c.Query("search")

	result, err := h.assigneeService.GetAvailableAssignees(workspaceID, userID, projectID, taskID, search, param.Page, param.Limit)
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

// AssignMembers
// @Summary      Assign members to task
// @Description  Assign project members to a task
// @Tags         Task Assignees
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        task_id path string true "Task ID"
// @Param        request body dto.AssignMembersRequest true "Members to assign"
// @Success      200  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/tasks/{task_id}/assignees [post]
func (h *TaskAssigneeHandler) AssignMembers(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	taskID := c.Param("task_id")
	var req dto.AssignMembersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.assigneeService.AssignMembersToTask(workspaceID, userID, projectID, taskID, &req)
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

// UnassignMembers
// @Summary      Unassign members from task
// @Description  Unassign members from a task
// @Tags         Task Assignees
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        task_id path string true "Task ID"
// @Param        request body dto.UnassignMembersRequest true "Members to unassign"
// @Success      200  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/tasks/{task_id}/assignees [delete]
func (h *TaskAssigneeHandler) UnassignMembers(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	taskID := c.Param("task_id")
	var req dto.UnassignMembersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.assigneeService.UnassignMembersFromTask(workspaceID, userID, projectID, taskID, &req)
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

// SelfAssign
// @Summary      Self-assign to task
// @Description  Assign the current user to a task
// @Tags         Task Assignees
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        task_id path string true "Task ID"
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/tasks/{task_id}/assignees/me [post]
func (h *TaskAssigneeHandler) SelfAssign(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	taskID := c.Param("task_id")

	result, err := h.assigneeService.SelfAssignToTask(workspaceID, userID, projectID, taskID)
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

// SelfUnassign
// @Summary      Self-unassign from task
// @Description  Remove the current user from a task's assignees
// @Tags         Task Assignees
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        task_id path string true "Task ID"
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/tasks/{task_id}/assignees/me [delete]
func (h *TaskAssigneeHandler) SelfUnassign(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	taskID := c.Param("task_id")

	result, err := h.assigneeService.SelfUnassignFromTask(workspaceID, userID, projectID, taskID)
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
