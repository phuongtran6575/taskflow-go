package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
	"TaskFlow-Go/internal/shared/appresponse"
)

type ActivityLogHandler struct {
	activityLogService _interface.ActivityLogService
}

func NewActivityLogHandler(activityLogService _interface.ActivityLogService) *ActivityLogHandler {
	return &ActivityLogHandler{activityLogService: activityLogService}
}

func (h *ActivityLogHandler) ListWorkspaceActivity(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "30"))
	cursor := c.Query("cursor")

	filters := map[string][]string{}
	if v := c.QueryArray("project_id"); len(v) > 0 {
		filters["project_id"] = v
	}
	if v := c.QueryArray("user_id"); len(v) > 0 {
		filters["user_id"] = v
	}
	if v := c.QueryArray("entity_type"); len(v) > 0 {
		filters["entity_type"] = v
	}
	if v := c.QueryArray("action"); len(v) > 0 {
		filters["action"] = v
	}
	if v := c.Query("date_from"); v != "" {
		filters["date_from"] = []string{v}
	}
	if v := c.Query("date_to"); v != "" {
		filters["date_to"] = []string{v}
	}

	result, err := h.activityLogService.ListWorkspaceActivity(workspaceID, userID, filters, limit, cursor)
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

func (h *ActivityLogHandler) ListProjectActivity(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "30"))
	cursor := c.Query("cursor")

	filters := map[string][]string{}
	if v := c.QueryArray("user_id"); len(v) > 0 {
		filters["user_id"] = v
	}
	if v := c.QueryArray("entity_type"); len(v) > 0 {
		filters["entity_type"] = v
	}
	if v := c.QueryArray("action"); len(v) > 0 {
		filters["action"] = v
	}
	if v := c.Query("date_from"); v != "" {
		filters["date_from"] = []string{v}
	}
	if v := c.Query("date_to"); v != "" {
		filters["date_to"] = []string{v}
	}

	result, err := h.activityLogService.ListProjectActivity(workspaceID, userID, projectID, filters, limit, cursor)
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

func (h *ActivityLogHandler) ListTaskActivity(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	taskID := c.Param("task_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	cursor := c.Query("cursor")
	direction := c.DefaultQuery("direction", "asc")
	includeComments := true
	if v := c.Query("include_comments"); v != "" {
		includeComments, _ = strconv.ParseBool(v)
	}

	result, err := h.activityLogService.ListTaskTimeline(workspaceID, userID, projectID, taskID, includeComments, limit, cursor, direction)
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

func (h *ActivityLogHandler) ExportWorkspaceActivity(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")

	data, filename, err := h.activityLogService.ExportWorkspaceActivity(workspaceID, userID, dateFrom, dateTo, "csv")
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			appresponse.Fail(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		appresponse.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Data(http.StatusOK, "text/csv", data)
}
