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

type WorkspaceInviteHandler struct {
	inviteService _interface.WorkspaceInviteService
}

func NewWorkspaceInviteHandler(inviteService _interface.WorkspaceInviteService) *WorkspaceInviteHandler {
	return &WorkspaceInviteHandler{inviteService: inviteService}
}

// ListInvites
// @Summary      List workspace invites
// @Description  List all invites for a workspace
// @Tags         Workspace Invites
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Param        status query string false "Filter by status"
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Failure      403  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/invites [get]
func (h *WorkspaceInviteHandler) ListInvites(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	param := parsePagination(c)
	status := c.Query("status")

	result, pagination, err := h.inviteService.ListInvites(workspaceID, userID, status, param.Page, param.Limit)
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

// CreateInvite
// @Summary      Create invite
// @Description  Create an invite link for a workspace
// @Tags         Workspace Invites
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        request body dto.CreateInviteRequest true "Invite details"
// @Success      201  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Failure      403  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/invites [post]
func (h *WorkspaceInviteHandler) CreateInvite(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	var req dto.CreateInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.inviteService.CreateInvite(workspaceID, userID, &req)
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

// PreviewInvite
// @Summary      Preview invite
// @Description  Preview workspace invite details by code (public)
// @Tags         Workspace Invites
// @Produce      json
// @Param        workspace_id path string true "Workspace ID"
// @Param        code path string true "Invite code"
// @Success      200  {object}  appresponse.Response
// @Failure      404  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/invites/preview/{code} [get]
func (h *WorkspaceInviteHandler) PreviewInvite(c *gin.Context) {
	code := c.Param("code")
	result, err := h.inviteService.GetInvitePreview(code)
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

// JoinWorkspace
// @Summary      Join workspace via invite
// @Description  Join a workspace using an invite code
// @Tags         Workspace Invites
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        code path string true "Invite code"
// @Success      200  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/invites/join/{code} [post]
func (h *WorkspaceInviteHandler) JoinWorkspace(c *gin.Context) {
	userID := c.GetString("user_id")
	code := c.Param("code")
	result, err := h.inviteService.JoinWorkspaceByCode(code, userID)
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

// RevokeInvite
// @Summary      Revoke invite
// @Description  Revoke a pending workspace invite
// @Tags         Workspace Invites
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        invite_id path string true "Invite ID"
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Failure      403  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/invites/{invite_id} [delete]
func (h *WorkspaceInviteHandler) RevokeInvite(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	inviteID := c.Param("invite_id")
	result, err := h.inviteService.RevokeInvite(workspaceID, userID, inviteID)
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
