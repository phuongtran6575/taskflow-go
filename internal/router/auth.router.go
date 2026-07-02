package router

import (
	"TaskFlow-Go/internal/handler"
	"TaskFlow-Go/internal/middleware"

	"github.com/gin-gonic/gin"
)

type AuthRouter struct{ handler *handler.AuthHandler }

func NewAuthRouter(handler *handler.AuthHandler) *AuthRouter {
	return &AuthRouter{handler: handler}
}

func (r *AuthRouter) RegisterRoutes(api *gin.RouterGroup) {
	auth := api.Group("/auth")
	{
		auth.POST("/register", r.handler.Register)
		auth.POST("/login", r.handler.Login)
		auth.POST("/refresh", r.handler.RefreshToken)
		auth.POST("/forgot-password", r.handler.ForgotPassword)
		auth.POST("/reset-password", r.handler.ResetPassword)
		auth.GET("/oauth/google", r.handler.InitiateGoogleOAuth)
		auth.GET("/oauth/google/callback", r.handler.HandleGoogleCallback)
		auth.GET("/oauth/github", r.handler.InitiateGithubOAuth)
		auth.GET("/oauth/github/callback", r.handler.HandleGithubCallback)
	}

	protected := api.Group("/auth", middleware.AuthMiddleware())
	{
		protected.POST("/logout", r.handler.Logout)
	}
}
