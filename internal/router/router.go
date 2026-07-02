package router

import (
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
}
