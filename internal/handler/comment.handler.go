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

type CommentHandler struct {
	commentService _interface.CommentService
}

func NewCommentHandler(commentService _interface.CommentService) *CommentHandler {
	return &CommentHandler{commentService: commentService}
}

// ListComments
// @Summary      List comments
// @Description  List comments on a task with cursor-based pagination
// @Tags         Comments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        task_id path string true "Task ID"
// @Param        limit query int false "Items per page" default(30)
// @Param        cursor query string false "Cursor for pagination"
// @Param        direction query string false "Sort direction (asc/desc)" default(asc)
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/tasks/{task_id}/comments [get]
func (h *CommentHandler) ListComments(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	taskID := c.Param("task_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "30"))
	cursor := c.Query("cursor")
	direction := c.DefaultQuery("direction", "asc")

	result, err := h.commentService.ListComments(workspaceID, userID, projectID, taskID, limit, cursor, direction)
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

// CreateComment
// @Summary      Create comment
// @Description  Create a new comment on a task
// @Tags         Comments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        task_id path string true "Task ID"
// @Param        request body dto.CreateCommentRequest true "Comment content"
// @Success      201  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/tasks/{task_id}/comments [post]
func (h *CommentHandler) CreateComment(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	taskID := c.Param("task_id")
	var req dto.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.commentService.CreateComment(workspaceID, userID, projectID, taskID, &req)
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

// UpdateComment
// @Summary      Update comment
// @Description  Update a comment's content
// @Tags         Comments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        task_id path string true "Task ID"
// @Param        comment_id path string true "Comment ID"
// @Param        request body dto.UpdateCommentRequest true "Updated comment content"
// @Success      200  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Failure      404  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/tasks/{task_id}/comments/{comment_id} [patch]
func (h *CommentHandler) UpdateComment(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	taskID := c.Param("task_id")
	commentID := c.Param("comment_id")
	var req dto.UpdateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.commentService.UpdateComment(workspaceID, userID, projectID, taskID, commentID, &req)
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

// DeleteComment
// @Summary      Delete comment
// @Description  Permanently delete a comment
// @Tags         Comments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        task_id path string true "Task ID"
// @Param        comment_id path string true "Comment ID"
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Failure      404  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/tasks/{task_id}/comments/{comment_id} [delete]
func (h *CommentHandler) DeleteComment(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	taskID := c.Param("task_id")
	commentID := c.Param("comment_id")

	result, err := h.commentService.DeleteComment(workspaceID, userID, projectID, taskID, commentID)
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

// GetMentionableUsers
// @Summary      Get mentionable users
// @Description  Get users that can be mentioned in a comment
// @Tags         Comments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        task_id path string true "Task ID"
// @Param        search query string false "Search keyword"
// @Param        limit query int false "Maximum results" default(10)
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/tasks/{task_id}/comments/mentionable [get]
func (h *CommentHandler) GetMentionableUsers(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	taskID := c.Param("task_id")
	search := c.Query("search")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	result, err := h.commentService.GetMentionableUsers(workspaceID, userID, projectID, taskID, search, limit)
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
