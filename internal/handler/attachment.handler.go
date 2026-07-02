package handler

import (
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
	"TaskFlow-Go/internal/shared/appresponse"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AttachmentHandler struct {
	attachmentService _interface.AttachmentService
	storageService    _interface.StorageService
}

func NewAttachmentHandler(attachmentService _interface.AttachmentService, storageService _interface.StorageService) *AttachmentHandler {
	return &AttachmentHandler{attachmentService: attachmentService, storageService: storageService}
}

// ListAttachments
// @Summary      List task attachments
// @Description  List all attachments on a task
// @Tags         Attachments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        task_id path string true "Task ID"
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Param        file_type query string false "Filter by file type"
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/tasks/{task_id}/attachments [get]
func (h *AttachmentHandler) ListAttachments(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	taskID := c.Param("task_id")
	param := parsePagination(c)
	fileType := c.Query("file_type")

	result, err := h.attachmentService.ListAttachments(workspaceID, userID, projectID, taskID, fileType, param.Page, param.Limit)
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

// UploadAttachments
// @Summary      Upload attachments
// @Description  Upload multiple files as task attachments
// @Tags         Attachments
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        task_id path string true "Task ID"
// @Param        files formData file true "Files to upload (multiple allowed)"
// @Success      201  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/tasks/{task_id}/attachments [post]
func (h *AttachmentHandler) UploadAttachments(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	taskID := c.Param("task_id")

	form, err := c.MultipartForm()
	if err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "Failed to parse multipart form")
		return
	}
	files := form.File["files"]
	if len(files) == 0 {
		appresponse.Fail(c, http.StatusBadRequest, "NO_FILES_PROVIDED", "No files provided")
		return
	}

	result, err := h.attachmentService.UploadAttachments(workspaceID, userID, projectID, taskID, files)
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

// GetDownloadUrl
// @Summary      Get download URL
// @Description  Get a presigned download URL for an attachment
// @Tags         Attachments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        task_id path string true "Task ID"
// @Param        attachment_id path string true "Attachment ID"
// @Param        disposition query string false "Content disposition (attachment/inline)" default(attachment)
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Failure      404  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/tasks/{task_id}/attachments/{attachment_id}/download [get]
func (h *AttachmentHandler) GetDownloadUrl(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	taskID := c.Param("task_id")
	attachmentID := c.Param("attachment_id")
	disposition := c.DefaultQuery("disposition", "attachment")

	result, err := h.attachmentService.GetDownloadUrl(workspaceID, userID, projectID, taskID, attachmentID, disposition)
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

// GetPreviewUrl
// @Summary      Get preview URL
// @Description  Get a preview URL for an attachment (images, documents)
// @Tags         Attachments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        task_id path string true "Task ID"
// @Param        attachment_id path string true "Attachment ID"
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Failure      404  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/tasks/{task_id}/attachments/{attachment_id}/preview [get]
func (h *AttachmentHandler) GetPreviewUrl(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	taskID := c.Param("task_id")
	attachmentID := c.Param("attachment_id")

	result, err := h.attachmentService.GetPreviewUrl(workspaceID, userID, projectID, taskID, attachmentID)
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

// DeleteAttachment
// @Summary      Delete attachment
// @Description  Permanently delete an attachment
// @Tags         Attachments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Param        project_id path string true "Project ID"
// @Param        task_id path string true "Task ID"
// @Param        attachment_id path string true "Attachment ID"
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Failure      404  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/projects/{project_id}/tasks/{task_id}/attachments/{attachment_id} [delete]
func (h *AttachmentHandler) DeleteAttachment(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")
	projectID := c.Param("project_id")
	taskID := c.Param("task_id")
	attachmentID := c.Param("attachment_id")

	result, err := h.attachmentService.DeleteAttachment(workspaceID, userID, projectID, taskID, attachmentID)
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

// GetStorageUsage
// @Summary      Get storage usage
// @Description  Get storage usage for a workspace
// @Tags         Attachments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        workspace_id path string true "Workspace ID"
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /workspaces/{workspace_id}/storage [get]
func (h *AttachmentHandler) GetStorageUsage(c *gin.Context) {
	userID := c.GetString("user_id")
	workspaceID := c.Param("workspace_id")

	result, err := h.storageService.GetWorkspaceStorageUsage(workspaceID, userID)
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
