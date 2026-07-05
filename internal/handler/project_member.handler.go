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

type ProjectMemberHandler struct {
	memberService _interface.ProjectMemberService
}

func NewProjectMemberHandler(memberService _interface.ProjectMemberService) *ProjectMemberHandler {
	return &ProjectMemberHandler{memberService: memberService}
}

// ListMembers
// @Summary      List project members
// @Description  List members of a project with pagination
// @Tags         Project Members
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Param        search query string false "Search keyword"
// @Param        role_id query string false "Filter by role ID"
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/members [get]
func (h *ProjectMemberHandler) ListMembers(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	param := parsePagination(c)
	search := c.Query("search")
	roleID := c.Query("role_id")

	result, pagination, err := h.memberService.ListMembers(workspaceID, userID, projectID, param.Page, param.Limit, search, roleID)
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

// GetAvailableWorkspaceMembers
// @Summary      Get available workspace members
// @Description  Get workspace members that are not yet added to the project
// @Tags         Project Members
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Param        search query string false "Search keyword"
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/members/available [get]
func (h *ProjectMemberHandler) GetAvailableWorkspaceMembers(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	param := parsePagination(c)
	search := c.Query("search")

	result, pagination, err := h.memberService.GetAvailableWorkspaceMembers(workspaceID, userID, projectID, search, param.Page, param.Limit)
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

// AddMembers
// @Summary      Add members to project
// @Description  Add workspace members to a project
// @Tags         Project Members
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        request body dto.AddMembersRequest true "Members to add"
// @Success      201  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/members [post]
func (h *ProjectMemberHandler) AddMembers(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	var req dto.AddMembersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.memberService.AddMembersToProject(workspaceID, userID, projectID, &req)
	if err != nil {
		var invRoleErr *apperror.InvalidRoleIDsError
		if errors.As(err, &invRoleErr) {
			appresponse.FailWithData(c, http.StatusBadRequest, "INVALID_ROLE_ID",
				invRoleErr.Message,
				map[string]interface{}{
					"code":            "INVALID_ROLE_ID",
					"invalid_role_ids": invRoleErr.InvalidRoleIDs,
				},
			)
			return
		}
		var invUserErr *apperror.InvalidUserIDsError
		if errors.As(err, &invUserErr) {
			appresponse.FailWithData(c, http.StatusBadRequest, "USER_NOT_IN_WORKSPACE",
				invUserErr.Message,
				map[string]interface{}{
					"code":             "USER_NOT_IN_WORKSPACE",
					"invalid_user_ids": invUserErr.InvalidUserIDs,
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

// UpdateMemberRole
// @Summary      Update project member role
// @Description  Update a project member's role
// @Tags         Project Members
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        user_id path string true "Target user ID"
// @Param        request body dto.UpdateProjectMemberRoleRequest true "New role details"
// @Success      200  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Failure      403  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/members/{user_id}/role [patch]
func (h *ProjectMemberHandler) UpdateMemberRole(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	targetUserID := c.Param("user_id")
	var req dto.UpdateProjectMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.memberService.UpdateMemberRole(workspaceID, userID, projectID, targetUserID, &req)
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

// RemoveMember
// @Summary      Remove project member
// @Description  Remove a member from the project
// @Tags         Project Members
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        user_id path string true "Target user ID"
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Failure      403  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/members/{user_id} [delete]
func (h *ProjectMemberHandler) RemoveMember(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	targetUserID := c.Param("user_id")

	result, err := h.memberService.RemoveMemberFromProject(workspaceID, userID, projectID, targetUserID)
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

// LeaveProject
// @Summary      Leave project
// @Description  Leave a project
// @Tags         Project Members
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        request body dto.LeaveProjectRequest true "Leaving confirmation"
// @Success      200  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/members/me [delete]
func (h *ProjectMemberHandler) LeaveProject(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	var req dto.LeaveProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.memberService.LeaveProject(workspaceID, userID, projectID, &req)
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
