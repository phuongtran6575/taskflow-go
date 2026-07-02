package router

import (
	"TaskFlow-Go/internal/handler"
	"TaskFlow-Go/internal/middleware"

	"github.com/gin-gonic/gin"
)

type RoleRouter struct {
	handler *handler.RoleHandler
	mw      *middleware.Middleware
}

func NewRoleRouter(handler *handler.RoleHandler, mw *middleware.Middleware) *RoleRouter {
	return &RoleRouter{handler: handler, mw: mw}
}

func (r *RoleRouter) RegisterRoutes(api *gin.RouterGroup) {
	roles := api.Group("/workspaces/:workspace_id/roles", middleware.AuthMiddleware())
	{
		roles.GET("/", r.mw.RequireWorkspaceRole("OWNER", "ADMIN"), r.handler.ListRoles)
		roles.POST("/", r.mw.RequireWorkspaceRole("OWNER", "ADMIN"), r.handler.CreateRole)
		roles.GET("/:role_id", r.mw.RequireWorkspaceRole("OWNER", "ADMIN"), r.handler.GetRoleDetails)
		roles.PATCH("/:role_id", r.mw.RequireWorkspaceRole("OWNER", "ADMIN"), r.handler.UpdateRole)
		roles.POST("/:role_id/permissions", r.mw.RequireWorkspaceRole("OWNER", "ADMIN"), r.handler.AssignPermissions)
		roles.DELETE("/:role_id/permissions", r.mw.RequireWorkspaceRole("OWNER", "ADMIN"), r.handler.RemovePermissions)
		roles.DELETE("/:role_id", r.mw.RequireWorkspaceRole("OWNER"), r.handler.DeleteRole)
	}
}
