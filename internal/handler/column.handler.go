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

type ColumnHandler struct {
	columnService _interface.ColumnService
}

func NewColumnHandler(columnService _interface.ColumnService) *ColumnHandler {
	return &ColumnHandler{columnService: columnService}
}

// ListColumns
// @Summary      List columns
// @Description  List all columns in a project
// @Tags         Columns
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/columns [get]
func (h *ColumnHandler) ListColumns(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")

	result, err := h.columnService.ListColumns(workspaceID, userID, projectID)
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

// CreateColumn
// @Summary      Create column
// @Description  Create a new column in a project
// @Tags         Columns
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        request body dto.CreateColumnRequest true "Column details"
// @Success      201  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/columns [post]
func (h *ColumnHandler) CreateColumn(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	var req dto.CreateColumnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.columnService.CreateColumn(workspaceID, userID, projectID, &req)
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

// UpdateColumnTitle
// @Summary      Update column title
// @Description  Update the title of a column
// @Tags         Columns
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        column_id path string true "Column ID"
// @Param        request body dto.UpdateColumnTitleRequest true "New title"
// @Success      200  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Failure      404  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/columns/{column_id}/title [patch]
func (h *ColumnHandler) UpdateColumnTitle(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	columnID := c.Param("column_id")
	var req dto.UpdateColumnTitleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.columnService.UpdateColumnTitle(workspaceID, userID, projectID, columnID, &req)
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

// UpdateColumnPosition
// @Summary      Update column position
// @Description  Update the position of a column
// @Tags         Columns
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        column_id path string true "Column ID"
// @Param        request body dto.UpdateColumnPositionRequest true "New position"
// @Success      200  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Failure      404  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/columns/{column_id}/position [patch]
func (h *ColumnHandler) UpdateColumnPosition(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	columnID := c.Param("column_id")
	var req dto.UpdateColumnPositionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.columnService.UpdateColumnPosition(workspaceID, userID, projectID, columnID, &req)
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

// DeleteColumn
// @Summary      Delete column
// @Description  Permanently delete a column
// @Tags         Columns
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        column_id path string true "Column ID"
// @Param        request body dto.DeleteColumnRequest true "Deletion confirmation"
// @Success      200  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Failure      404  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/columns/{column_id} [delete]
func (h *ColumnHandler) DeleteColumn(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	columnID := c.Param("column_id")
	var req dto.DeleteColumnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.columnService.DeleteColumn(workspaceID, userID, projectID, columnID, &req)
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
