package router

import (
	"TaskFlow-Go/internal/ws"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func SetupRoutes(
	api *gin.RouterGroup,
	auth *AuthRouter,
	user *UserRouter,
	workspace *WorkspaceRouter,
	permission *PermissionRouter,
	role *RoleRouter,
	project *ProjectRouter,
	task *TaskRouter,
	notification *NotificationRouter,
	activityLog *ActivityLogRouter,
	hub *ws.Hub,
) {
	api.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	auth.RegisterRoutes(api)
	user.RegisterRoutes(api)
	workspace.RegisterRoutes(api)
	permission.RegisterRoutes(api)
	role.RegisterRoutes(api)
	project.RegisterRoutes(api)
	task.RegisterRoutes(api)
	notification.RegisterRoutes(api)
	activityLog.RegisterRoutes(api)

	// WebSocket routes
	api.GET("/ws", ws.WSHandler(hub))
	api.GET("/ws/projects/:project_id", ws.WSProjectHandler(hub))
}
