package router

import (
	"TaskFlow-Go/internal/handler"
	"TaskFlow-Go/internal/middleware"

	"github.com/gin-gonic/gin"
)

type ProjectRouter struct {
	handler       *handler.ProjectHandler
	columnHandler *handler.ColumnHandler
	memberHandler *handler.ProjectMemberHandler
	mw            *middleware.Middleware
}

func NewProjectRouter(
	handler *handler.ProjectHandler,
	columnHandler *handler.ColumnHandler,
	memberHandler *handler.ProjectMemberHandler,
	mw *middleware.Middleware,
) *ProjectRouter {
	return &ProjectRouter{
		handler:       handler,
		columnHandler: columnHandler,
		memberHandler: memberHandler,
		mw:            mw,
	}
}

func (r *ProjectRouter) RegisterRoutes(api *gin.RouterGroup) {
	auth := middleware.AuthMiddleware()

	projects := api.Group("/workspaces/:workspace_id/projects", auth)
	{
		// Chỉ cần là workspace member mới thấy danh sách project
		projects.GET("/", r.mw.RequireWorkspaceRole(), r.handler.ListProjects)
		projects.POST("/", r.mw.RequireWorkspaceRole(), r.handler.CreateProject)
		// Các thao tác trên project cụ thể → cần là project member
		projects.GET("/:project_id", r.mw.RequireProjectMember(), r.handler.GetProjectDetails)
		projects.PATCH("/:project_id", r.mw.RequireProjectPermission(middleware.PermProjectUpdate), r.handler.UpdateProject)
		projects.PATCH("/:project_id/archive", r.mw.RequireProjectPermission(middleware.PermProjectArchive), r.handler.ArchiveProject)
		projects.PATCH("/:project_id/unarchive", r.mw.RequireProjectPermission(middleware.PermProjectArchive), r.handler.UnarchiveProject)
		projects.PATCH("/:project_id/favorite", r.mw.RequireProjectMember(), r.handler.ToggleFavorite)
		projects.DELETE("/:project_id", r.mw.RequireProjectPermission(middleware.PermProjectDelete), r.handler.DeleteProject)
	}

	columns := api.Group("/workspaces/:workspace_id/projects/:project_id/columns", auth)
	{
		columns.GET("/", r.mw.RequireProjectMember(), r.columnHandler.ListColumns)
		columns.POST("/", r.mw.RequireProjectPermission(middleware.PermColumnCreate), r.columnHandler.CreateColumn)
		columns.PATCH("/:column_id/title", r.mw.RequireProjectPermission(middleware.PermColumnUpdate), r.columnHandler.UpdateColumnTitle)
		columns.PATCH("/:column_id/position", r.mw.RequireProjectPermission(middleware.PermColumnUpdate), r.columnHandler.UpdateColumnPosition)
		columns.DELETE("/:column_id", r.mw.RequireProjectPermission(middleware.PermColumnDelete), r.columnHandler.DeleteColumn)
	}

	members := api.Group("/workspaces/:workspace_id/projects/:project_id/members", auth)
	{
		members.GET("/", r.mw.RequireProjectMember(), r.memberHandler.ListMembers)
		members.GET("/available", r.mw.RequireProjectPermission(middleware.PermProjectManageMembers), r.memberHandler.GetAvailableWorkspaceMembers)
		members.POST("/", r.mw.RequireProjectPermission(middleware.PermProjectManageMembers), r.memberHandler.AddMembers)
		members.PATCH("/:user_id/role", r.mw.RequireProjectPermission(middleware.PermProjectManageMembers), r.memberHandler.UpdateMemberRole)
		members.DELETE("/:user_id", r.mw.RequireProjectPermission(middleware.PermProjectManageMembers), r.memberHandler.RemoveMember)
		members.DELETE("/me", r.mw.RequireProjectMember(), r.memberHandler.LeaveProject)
	}
}
