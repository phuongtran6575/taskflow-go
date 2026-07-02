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

type WorkspaceMemberHandler struct {
	memberService _interface.WorkspaceMemberService
}

func NewWorkspaceMemberHandler(memberService _interface.WorkspaceMemberService) *WorkspaceMemberHandler {
	return &WorkspaceMemberHandler{memberService: memberService}
}

// ListMembers
// @Summary      List workspace members
// @Description  List members of a workspace with pagination and filters
// @Tags         Workspace Members
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Param        search query string false "Search keyword"
// @Param        role query string false "Filter by role"
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Failure      403  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/members [get]
func (h *WorkspaceMemberHandler) ListMembers(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	param := parsePagination(c)
	search := c.Query("search")
	role := c.Query("role")

	result, pagination, err := h.memberService.ListMembers(workspaceID, userID, param.Page, param.Limit, search, role)
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

// GetMemberDetails
// @Summary      Get member details
// @Description  Get details of a specific workspace member
// @Tags         Workspace Members
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        user_id path string true "User ID"
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Failure      404  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/members/{user_id} [get]
func (h *WorkspaceMemberHandler) GetMemberDetails(c *gin.Context) {
	workspaceID := c.Param("workspace_id")
	targetUserID := c.Param("user_id")

	result, err := h.memberService.GetMemberDetails(workspaceID, targetUserID)
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

// UpdateMemberRole
// @Summary      Update member role
// @Description  Update the role of a workspace member
// @Tags         Workspace Members
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        user_id path string true "Target user ID"
// @Param        request body dto.UpdateMemberRoleRequest true "New role details"
// @Success      200  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Failure      403  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/members/{user_id}/role [patch]
func (h *WorkspaceMemberHandler) UpdateMemberRole(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	targetUserID := c.Param("user_id")
	var req dto.UpdateMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.memberService.UpdateMemberRole(workspaceID, userID, targetUserID, &req)
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

// TransferOwnership
// @Summary      Transfer workspace ownership
// @Description  Transfer workspace ownership to another member
// @Tags         Workspace Members
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        request body dto.TransferOwnershipRequest true "New owner details"
// @Success      200  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Failure      403  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/members/transfer-ownership [post]
func (h *WorkspaceMemberHandler) TransferOwnership(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	var req dto.TransferOwnershipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.memberService.TransferOwnership(workspaceID, userID, &req)
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

// KickMember
// @Summary      Remove workspace member
// @Description  Remove a member from the workspace
// @Tags         Workspace Members
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        user_id path string true "Target user ID"
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Failure      403  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/members/{user_id} [delete]
func (h *WorkspaceMemberHandler) KickMember(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	targetUserID := c.Param("user_id")

	result, err := h.memberService.KickMember(workspaceID, userID, targetUserID)
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

// LeaveWorkspace
// @Summary      Leave workspace
// @Description  Leave a workspace
// @Tags         Workspace Members
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        request body dto.LeaveWorkspaceRequest true "Leaving workspace confirmation"
// @Success      200  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/members/me [delete]
func (h *WorkspaceMemberHandler) LeaveWorkspace(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	var req dto.LeaveWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.memberService.LeaveWorkspace(workspaceID, userID, &req)
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
