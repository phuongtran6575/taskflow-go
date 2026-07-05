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

	projects := api.Group("/workspaces/:workspace_id/projects", auth, r.mw.RequireWorkspaceRole())
	{
		projects.GET("/", r.handler.ListProjects)
		projects.POST("/", r.handler.CreateProject)

		notArchived := r.mw.RequireProjectNotArchived()
		projectMember := r.mw.RequireProjectMember()

		projects.GET("/:project_id", projectMember, r.handler.GetProjectDetails)
		projects.PATCH("/:project_id", projectMember, notArchived, r.mw.RequireProjectPermission(middleware.PermProjectUpdate), r.handler.UpdateProject)
		projects.PATCH("/:project_id/archive", projectMember, r.mw.RequireProjectPermission(middleware.PermProjectArchive), r.handler.ArchiveProject)
		projects.PATCH("/:project_id/unarchive", projectMember, r.mw.RequireProjectPermission(middleware.PermProjectArchive), r.handler.UnarchiveProject)
		projects.PATCH("/:project_id/favorite", projectMember, r.handler.ToggleFavorite)
		projects.DELETE("/:project_id", projectMember, r.mw.RequireProjectPermission(middleware.PermProjectDelete), r.handler.DeleteProject)
	}

	columns := api.Group("/workspaces/:workspace_id/projects/:project_id/columns", auth, r.mw.RequireWorkspaceRole())
	{
		notArchived := r.mw.RequireProjectNotArchived()
		projectMember := r.mw.RequireProjectMember()

		columns.GET("/", projectMember, r.columnHandler.ListColumns)
		columns.POST("/", projectMember, notArchived, r.mw.RequireProjectPermission(middleware.PermColumnCreate), r.columnHandler.CreateColumn)
		columns.PATCH("/:column_id/title", projectMember, notArchived, r.mw.RequireProjectPermission(middleware.PermColumnUpdate), r.columnHandler.UpdateColumnTitle)
		columns.PATCH("/:column_id/position", projectMember, notArchived, r.mw.RequireProjectPermission(middleware.PermColumnUpdate), r.columnHandler.UpdateColumnPosition)
		columns.DELETE("/:column_id", projectMember, notArchived, r.mw.RequireProjectPermission(middleware.PermColumnDelete), r.columnHandler.DeleteColumn)
	}

	members := api.Group("/workspaces/:workspace_id/projects/:project_id/members", auth, r.mw.RequireWorkspaceRole())
	{
		notArchived := r.mw.RequireProjectNotArchived()
		projectMember := r.mw.RequireProjectMember()

		members.GET("/", projectMember, r.memberHandler.ListMembers)
		members.GET("/available", projectMember, r.mw.RequireProjectPermission(middleware.PermProjectManageMembers), r.memberHandler.GetAvailableWorkspaceMembers)
		members.POST("/", projectMember, notArchived, r.mw.RequireProjectPermission(middleware.PermProjectManageMembers), r.memberHandler.AddMembers)
		members.PATCH("/:user_id/role", projectMember, notArchived, r.mw.RequireProjectPermission(middleware.PermProjectManageMembers), r.memberHandler.UpdateMemberRole)
		members.DELETE("/:user_id", projectMember, notArchived, r.mw.RequireProjectPermission(middleware.PermProjectManageMembers), r.memberHandler.RemoveMember)
		members.DELETE("/me", projectMember, r.memberHandler.LeaveProject)
	}
}
