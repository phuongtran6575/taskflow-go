package router

import (
	"TaskFlow-Go/internal/handler"
	"TaskFlow-Go/internal/middleware"

	"github.com/gin-gonic/gin"
)

type ActivityLogRouter struct {
	handler *handler.ActivityLogHandler
	mw      *middleware.Middleware
}

func NewActivityLogRouter(handler *handler.ActivityLogHandler, mw *middleware.Middleware) *ActivityLogRouter {
	return &ActivityLogRouter{handler: handler, mw: mw}
}

func (r *ActivityLogRouter) RegisterRoutes(api *gin.RouterGroup) {
	wsActivity := api.Group("/workspaces/:workspace_id/activity", middleware.AuthMiddleware())
	{
		wsActivity.GET("", r.mw.RequireWorkspaceRole("OWNER", "ADMIN"), r.handler.ListWorkspaceActivity)
		wsActivity.GET("/export", r.mw.RequireWorkspaceRole("OWNER"), r.handler.ExportWorkspaceActivity)
	}

	projectActivity := api.Group("/workspaces/:workspace_id/projects/:project_id/activity", middleware.AuthMiddleware())
	{
		projectActivity.GET("", r.mw.RequireProjectMember(), r.handler.ListProjectActivity)
	}

	taskActivity := api.Group("/workspaces/:workspace_id/projects/:project_id/tasks/:task_id/activity", middleware.AuthMiddleware())
	{
		taskActivity.GET("", r.mw.RequireProjectMember(), r.handler.ListTaskActivity)
	}
}
