package router

import (
	"TaskFlow-Go/internal/handler"
	"TaskFlow-Go/internal/middleware"

	"github.com/gin-gonic/gin"
)

type WorkspaceRouter struct {
	handler       *handler.WorkspaceHandler
	memberHandler *handler.WorkspaceMemberHandler
	inviteHandler *handler.WorkspaceInviteHandler
	mw            *middleware.Middleware
}

func NewWorkspaceRouter(
	handler *handler.WorkspaceHandler,
	memberHandler *handler.WorkspaceMemberHandler,
	inviteHandler *handler.WorkspaceInviteHandler,
	mw *middleware.Middleware,
) *WorkspaceRouter {
	return &WorkspaceRouter{
		handler:       handler,
		memberHandler: memberHandler,
		inviteHandler: inviteHandler,
		mw:            mw,
	}
}

func (r *WorkspaceRouter) RegisterRoutes(api *gin.RouterGroup) {
	auth := middleware.AuthMiddleware()

	ws := api.Group("/workspaces", auth)
	{
		ws.GET("/", r.handler.GetMyWorkspaces)
		ws.POST("/", r.handler.CreateWorkspace)
		// Chỉ cần là member của workspace mới xem được detail
		ws.GET("/:workspace_id", r.mw.RequireWorkspaceRole(), r.handler.GetWorkspaceDetails)
		// Chỉ OWNER/ADMIN mới được sửa workspace
		ws.PATCH("/:workspace_id", r.mw.RequireWorkspaceRole("OWNER", "ADMIN"), r.handler.UpdateWorkspace)
		ws.PUT("/:workspace_id/plan", r.mw.RequireWorkspaceRole("OWNER"), r.handler.UpgradePlan)
		ws.DELETE("/:workspace_id", r.mw.RequireWorkspaceRole("OWNER"), r.handler.DeleteWorkspace)
	}

	members := api.Group("/workspaces/:workspace_id/members", auth)
	{
		members.GET("/", r.mw.RequireWorkspaceRole(), r.memberHandler.ListMembers)
		members.GET("/:user_id", r.mw.RequireWorkspaceRole(), r.memberHandler.GetMemberDetails)
		members.PATCH("/:user_id/role", r.mw.RequireWorkspaceRole("OWNER", "ADMIN"), r.memberHandler.UpdateMemberRole)
		members.POST("/transfer-ownership", r.mw.RequireWorkspaceRole("OWNER"), r.memberHandler.TransferOwnership)
		members.DELETE("/:user_id", r.mw.RequireWorkspaceRole("OWNER", "ADMIN"), r.memberHandler.KickMember)
		members.DELETE("/me", r.mw.RequireWorkspaceRole(), r.memberHandler.LeaveWorkspace)
	}

	invitesPublic := api.Group("/workspaces/:workspace_id/invites")
	{
		invitesPublic.GET("/preview/:code", r.inviteHandler.PreviewInvite)
	}

	invites := api.Group("/workspaces/:workspace_id/invites", auth)
	{
		invites.GET("/", r.mw.RequireWorkspaceRole(), r.inviteHandler.ListInvites)
		invites.POST("/", r.mw.RequireWorkspaceRole("OWNER", "ADMIN"), r.inviteHandler.CreateInvite)
		invites.POST("/join/:code", r.inviteHandler.JoinWorkspace) // join bằng invite link, không check role
		invites.DELETE("/:invite_id", r.mw.RequireWorkspaceRole("OWNER", "ADMIN"), r.inviteHandler.RevokeInvite)
	}
}
