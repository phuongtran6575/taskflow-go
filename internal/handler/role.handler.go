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

type RoleHandler struct {
	roleService _interface.RoleService
}

func NewRoleHandler(roleService _interface.RoleService) *RoleHandler {
	return &RoleHandler{roleService: roleService}
}

// ListRoles
// @Summary      List roles
// @Description  List all roles in a workspace with pagination
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Param        search query string false "Search keyword"
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/roles [get]
func (h *RoleHandler) ListRoles(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	param := parsePagination(c)
	search := c.Query("search")

	result, pagination, err := h.roleService.ListRoles(workspaceID, userID, search, param.Page, param.Limit)
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

// CreateRole
// @Summary      Create role
// @Description  Create a new role in a workspace
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        request body dto.CreateRoleRequest true "Role details"
// @Success      201  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/roles [post]
func (h *RoleHandler) CreateRole(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	var req dto.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.roleService.CreateRole(workspaceID, userID, &req)
	if err != nil {
		var invPermErr *apperror.InvalidPermissionIDsError
		if errors.As(err, &invPermErr) {
			appresponse.FailWithData(c, http.StatusBadRequest, "INVALID_PERMISSION_IDS",
				invPermErr.Message,
				map[string]interface{}{
					"code":         "INVALID_PERMISSION_IDS",
					"invalid_ids":  invPermErr.InvalidIDs,
				},
			)
			return
		}
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

// GetRoleDetails
// @Summary      Get role details
// @Description  Get details of a specific role including permissions
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        role_id path string true "Role ID"
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Failure      404  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/roles/{role_id} [get]
func (h *RoleHandler) GetRoleDetails(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	roleID := c.Param("role_id")

	result, err := h.roleService.GetRoleById(workspaceID, userID, roleID)
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

// UpdateRole
// @Summary      Update role
// @Description  Update role name or description
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        role_id path string true "Role ID"
// @Param        request body dto.UpdateRoleRequest true "Role update data"
// @Success      200  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Failure      404  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/roles/{role_id} [patch]
func (h *RoleHandler) UpdateRole(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	roleID := c.Param("role_id")
	var req dto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.roleService.UpdateRole(workspaceID, userID, roleID, &req)
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

// AssignPermissions
// @Summary      Assign permissions to role
// @Description  Assign permissions to a role
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        role_id path string true "Role ID"
// @Param        request body dto.AssignPermissionsRequest true "Permissions to assign"
// @Success      200  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Failure      404  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/roles/{role_id}/permissions [post]
func (h *RoleHandler) AssignPermissions(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	roleID := c.Param("role_id")
	var req dto.AssignPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.roleService.AssignPermissionsToRole(workspaceID, userID, roleID, &req)
	if err != nil {
		var invPermErr *apperror.InvalidPermissionIDsError
		if errors.As(err, &invPermErr) {
			appresponse.FailWithData(c, http.StatusBadRequest, "INVALID_PERMISSION_IDS",
				invPermErr.Message,
				map[string]interface{}{
					"code":         "INVALID_PERMISSION_IDS",
					"invalid_ids":  invPermErr.InvalidIDs,
				},
			)
			return
		}
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

// RemovePermissions
// @Summary      Remove permissions from role
// @Description  Remove permissions from a role
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        role_id path string true "Role ID"
// @Param        request body dto.RemovePermissionsRequest true "Permissions to remove"
// @Success      200  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Failure      404  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/roles/{role_id}/permissions [delete]
func (h *RoleHandler) RemovePermissions(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	roleID := c.Param("role_id")
	var req dto.RemovePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.roleService.RemovePermissionsFromRole(workspaceID, userID, roleID, &req)
	if err != nil {
		var invPermErr *apperror.InvalidPermissionIDsError
		if errors.As(err, &invPermErr) {
			appresponse.FailWithData(c, http.StatusBadRequest, "INVALID_PERMISSION_IDS",
				invPermErr.Message,
				map[string]interface{}{
					"code":         "INVALID_PERMISSION_IDS",
					"invalid_ids":  invPermErr.InvalidIDs,
				},
			)
			return
		}
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

// DeleteRole
// @Summary      Delete role
// @Description  Permanently delete a role from the workspace
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        role_id path string true "Role ID"
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Failure      403  {object}  appresponse.Response
// @Failure      404  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/roles/{role_id} [delete]
func (h *RoleHandler) DeleteRole(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	roleID := c.Param("role_id")

	err := h.roleService.DeleteRole(workspaceID, userID, roleID)
	if err != nil {
		var roleInUseErr *apperror.RoleInUseError
		if errors.As(err, &roleInUseErr) {
			projects, _ := roleInUseErr.AffectedProjects.([]dto.AffectedProject)
			appresponse.FailWithData(c, http.StatusConflict, "ROLE_IN_USE",
				roleInUseErr.Message,
				dto.RoleInUseErrorResponse{
					Code:                 "ROLE_IN_USE",
					Message:              roleInUseErr.Message,
					AffectedProjects:     projects,
					TotalAffectedMembers: roleInUseErr.TotalAffectedMembers,
				},
			)
			return
		}
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			appresponse.Fail(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		appresponse.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	appresponse.OK(c, dto.RoleDeleteResponse{Message: "Role deleted successfully.", DeletedRoleID: roleID})
}
