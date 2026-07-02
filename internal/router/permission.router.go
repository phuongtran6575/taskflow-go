package router

import (
	"TaskFlow-Go/internal/handler"
	"TaskFlow-Go/internal/middleware"

	"github.com/gin-gonic/gin"
)

type PermissionRouter struct {
	handler *handler.PermissionHandler
	mw      *middleware.Middleware
}

func NewPermissionRouter(handler *handler.PermissionHandler, mw *middleware.Middleware) *PermissionRouter {
	return &PermissionRouter{handler: handler, mw: mw}
}

func (r *PermissionRouter) RegisterRoutes(api *gin.RouterGroup) {
	perm := api.Group("/permissions", middleware.AuthMiddleware(), r.mw.RequireAnyWorkspaceAdminRole())
	{
		perm.GET("/", r.handler.ListPermissions)
		perm.GET("/modules", r.handler.ListModules)
		perm.GET("/modules/:module", r.handler.GetPermissionsByModule)
		perm.GET("/:id_or_slug", r.handler.GetPermissionByIdOrSlug)
	}
}
