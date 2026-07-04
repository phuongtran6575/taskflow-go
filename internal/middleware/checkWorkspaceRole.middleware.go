package middleware

import (
	"errors"
	"net/http"

	"TaskFlow-Go/internal/shared/appresponse"

	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

// RequireWorkspaceRole kiểm tra user có thuộc workspace và có role được phép không.
// Phải đặt SAU AuthMiddleware trong chain vì cần user_id đã được set vào context.
//
// Cách dùng trong router:
//
//	ws.PATCH("/:workspace_id", mw.RequireWorkspaceRole("OWNER", "ADMIN"), handler.Update)
//
// Nếu không truyền role → chỉ cần là member của workspace là đủ:
//
//	ws.GET("/:workspace_id", mw.RequireWorkspaceRole(), handler.GetDetails)
// RequireAnyWorkspaceAdminRole kiểm tra user có role OWNER hoặc ADMIN trong workspace bất kỳ.
// Dùng cho các endpoint system-wide như /permissions yêu cầu quyền quản trị.
func (m *Middleware) RequireAnyWorkspaceAdminRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(ContextKeyUserID)
		if userID == "" {
			appresponse.Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
			c.Abort()
			return
		}

		members, err := 	m.workspaceMemberRepo.ListByUserID(userID)
		if err != nil {
			appresponse.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			c.Abort()
			return
		}

		for _, member := range members {
			if member.Role == "OWNER" || member.Role == "ADMIN" {
				c.Next()
				return
			}
		}

		appresponse.Fail(c, http.StatusForbidden, "FORBIDDEN", "You do not have permission")
		c.Abort()
	}
}

func (m *Middleware) RequireWorkspaceRole(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		workspaceID := c.Param("workspace_id")
		if workspaceID == "" {
			appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "workspace_id is required")
			c.Abort()
			return
		}

		userIDVal, exists := c.Get(ContextKeyUserID)
		if !exists {
			appresponse.Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
			c.Abort()
			return
		}
		userID := userIDVal.(string)

		if _, err := m.workspaceRepo.GetByID(workspaceID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				appresponse.Fail(c, http.StatusNotFound, "WORKSPACE_NOT_FOUND", "Workspace not found or has been deleted")
				c.Abort()
				return
			}
			appresponse.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			c.Abort()
			return
		}

		member, err := m.workspaceMemberRepo.GetByID(workspaceID, userID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				appresponse.Fail(c, http.StatusForbidden, "FORBIDDEN", "You are not a member of this workspace")
				c.Abort()
				return
			}
			appresponse.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			c.Abort()
			return
		}

		if len(requiredRoles) == 0 {
			c.Set("workspace_role", string(member.Role))
			c.Next()
			return
		}

		userRole := string(member.Role)
		for _, r := range requiredRoles {
			if userRole == r {
				c.Set("workspace_role", userRole)
				c.Next()
				return
			}
		}

		appresponse.Fail(c, http.StatusForbidden, "FORBIDDEN", "Insufficient role in this workspace")
		c.Abort()
	}
}
