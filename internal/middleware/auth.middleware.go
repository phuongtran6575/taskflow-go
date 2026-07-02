package middleware

import (
	"TaskFlow-Go/internal/helper"
	"TaskFlow-Go/internal/shared/appresponse"
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	AuthorizationHeader = "Authorization"
	BearerPrefix        = "Bearer "
	ContextKeyUserID    = "user_id"
	ContextKeyEmail     = "email"
	//ContextKeyRole      = "role"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(AuthorizationHeader)
		if authHeader == "" {
			appresponse.Fail(c, 401, "UNAUTHORIZED", "Missing authorization header")
			c.Abort()
			return
		}

		if !strings.HasPrefix(authHeader, BearerPrefix) {
			appresponse.Fail(c, 401, "UNAUTHORIZED", "Invalid authorization format, expected: Bearer <token>")
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, BearerPrefix)

		claims, err := helper.ValidateAccessToken(tokenStr)
		if err != nil {
			if errors.Is(err, helper.ErrTokenExpired) {
				appresponse.Fail(c, 401, "TOKEN_EXPIRED", "Token has expired")
				c.Abort()
				return
			}
			appresponse.Fail(c, 401, "UNAUTHORIZED", "Invalid token")
			c.Abort()
			return
		}

		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyEmail, claims.Email)
		//c.Set(ContextKeyRole, claims.Role)
		c.Next()
	}
}
