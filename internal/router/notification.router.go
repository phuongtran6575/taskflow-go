package router

import (
	"TaskFlow-Go/internal/handler"
	"TaskFlow-Go/internal/middleware"

	"github.com/gin-gonic/gin"
)

type NotificationRouter struct {
	handler *handler.NotificationHandler
	mw      *middleware.Middleware
}

func NewNotificationRouter(handler *handler.NotificationHandler, mw *middleware.Middleware) *NotificationRouter {
	return &NotificationRouter{handler: handler, mw: mw}
}

func (r *NotificationRouter) RegisterRoutes(api *gin.RouterGroup) {
	notif := api.Group("/workspaces/:workspace_id/notifications", middleware.AuthMiddleware())
	{
		notif.GET("/", r.handler.ListNotifications)
		notif.GET("/unread-count", r.handler.GetUnreadCount)
		notif.PATCH("/:notification_id/read", r.handler.MarkAsRead)
		notif.PATCH("/read-all", r.handler.MarkAllAsRead)
		notif.POST("/announcements", r.mw.RequireWorkspaceRole("OWNER", "ADMIN"), r.handler.CreateAnnouncement)
	}
}
