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

type PermissionHandler struct {
	permissionService _interface.PermissionService
}

func NewPermissionHandler(permissionService _interface.PermissionService) *PermissionHandler {
	return &PermissionHandler{permissionService: permissionService}
}

// setCacheHeaders gắn ETag và Cache-Control vào response.
// Nếu If-None-Match match → trả về 304, không gọi service.
func (h *PermissionHandler) setCacheHeaders(c *gin.Context) bool {
	etag := h.permissionService.GetPermissionsETag()
	if etag == "" {
		return false
	}

	c.Header("ETag", etag)
	c.Header("Cache-Control", "public, max-age=3600")

	if match := c.GetHeader("If-None-Match"); match != "" && match == etag {
		c.Status(http.StatusNotModified)
		return true
	}
	return false
}

// ListPermissions
// @Summary      List all permissions
// @Description  List all permissions, optionally grouped by module
// @Tags         Permissions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        grouped query bool false "Group by module" default(true)
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /permissions [get]
func (h *PermissionHandler) ListPermissions(c *gin.Context) {
	if h.setCacheHeaders(c) {
		return
	}

	userID := c.GetString("user_id")
	grouped, _ := strconv.ParseBool(c.DefaultQuery("grouped", "true"))
	var Resp any
	var err error
	if grouped {
		Resp, err = h.permissionService.ListGroupedPermissions(userID)
	} else {
		Resp, err = h.permissionService.ListFlatPermissions(userID)
	}
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			appresponse.Fail(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		appresponse.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	appresponse.OK(c, Resp)
}

// ListModules
// @Summary      List permission modules
// @Description  List all permission modules
// @Tags         Permissions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /permissions/modules [get]
func (h *PermissionHandler) ListModules(c *gin.Context) {
	if h.setCacheHeaders(c) {
		return
	}

	userID := c.GetString("user_id")
	result, err := h.permissionService.ListModules(userID)
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

// GetPermissionsByModule
// @Summary      Get permissions by module
// @Description  Get all permissions for a specific module
// @Tags         Permissions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        module path string true "Module name"
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Failure      404  {object}  appresponse.Response
// @Router       /permissions/modules/{module} [get]
func (h *PermissionHandler) GetPermissionsByModule(c *gin.Context) {
	if h.setCacheHeaders(c) {
		return
	}

	userID := c.GetString("user_id")
	module := c.Param("module")
	result, err := h.permissionService.GetPermissionsByModule(userID, module)
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

// GetPermissionByIdOrSlug
// @Summary      Get permission by ID or slug
// @Description  Get a single permission by its ID or slug
// @Tags         Permissions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id_or_slug path string true "Permission ID or slug"
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Failure      404  {object}  appresponse.Response
// @Router       /permissions/{id_or_slug} [get]
func (h *PermissionHandler) GetPermissionByIdOrSlug(c *gin.Context) {
	// Single permission — cache ít giá trị, bỏ qua ETag
	userID := c.GetString("user_id")
	idOrSlug := c.Param("id_or_slug")
	result, err := h.permissionService.GetPermissionByIdOrSlug(userID, idOrSlug)
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
