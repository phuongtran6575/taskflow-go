package router

import (
	"TaskFlow-Go/internal/handler"
	"TaskFlow-Go/internal/middleware"

	"github.com/gin-gonic/gin"
)

type UserRouter struct{ handler *handler.UserHandler }

func NewUserRouter(handler *handler.UserHandler) *UserRouter {
	return &UserRouter{handler: handler}
}

func (r *UserRouter) RegisterRoutes(api *gin.RouterGroup) {
	users := api.Group("/users", middleware.AuthMiddleware())
	{
		users.GET("/me", r.handler.GetProfile)
		users.PATCH("/me", r.handler.UpdateProfile)
		users.POST("/me/avatar", r.handler.UploadAvatar)
		users.DELETE("/me/avatar", r.handler.RemoveAvatar)
		users.PUT("/me/password", r.handler.ChangePassword)
		users.GET("/me/sessions", r.handler.GetSessions)
		users.DELETE("/me/sessions/:session_id", r.handler.RevokeSession)
		users.DELETE("/me/sessions", r.handler.RevokeAllOtherSessions)
		users.DELETE("/me", r.handler.DeleteAccount)
	}
}
