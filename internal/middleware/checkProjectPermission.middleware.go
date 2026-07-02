package middleware

import (
	"errors"
	"net/http"

	"TaskFlow-Go/internal/shared/appresponse"

	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

// RequireProjectMember kiểm tra user có là member của project không.
// Chỉ verify membership, không check permission cụ thể.
// Dùng cho các action mà mọi project member đều được phép (xem board, xem task...).
//
// Sau middleware này, handler có thể đọc "project_role_id" từ context nếu cần.
func (m *Middleware) RequireProjectMember() gin.HandlerFunc {
	return func(c *gin.Context) {
		projectID := c.Param("project_id")
		if projectID == "" {
			appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "project_id is required")
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

		member, err := m.projectMemberRepo.GetByID(projectID, userID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				appresponse.Fail(c, http.StatusForbidden, "FORBIDDEN", "You are not a member of this project")
				c.Abort()
				return
			}
			appresponse.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			c.Abort()
			return
		}

		// Set role_id vào context để handler hoặc middleware sau có thể dùng
		if member.RoleID != nil {
			c.Set("project_role_id", *member.RoleID)
		}
		c.Next()
	}
}

// RequireProjectPermission kiểm tra user có ít nhất 1 trong các permission slugs được yêu cầu.
// Dùng cho các action cần RBAC chi tiết (tạo task, xóa column, quản lý member...).
//
// Cách dùng trong router:
//
//	tasks.POST("/", mw.RequireProjectPermission(middleware.PermTaskCreate), handler.CreateTask)
//	tasks.DELETE("/:id", mw.RequireProjectPermission(middleware.PermTaskDelete), handler.DeleteTask)
//
// Workspace OWNER/ADMIN có thể bypass check này bằng cách xem workspace_role từ context.
func (m *Middleware) RequireProjectPermission(requiredSlugs ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		projectID := c.Param("project_id")
		if projectID == "" {
			appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "project_id is required")
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

		// Workspace OWNER/ADMIN bypass project-level RBAC
		// (họ đã được check ở RequireWorkspaceRole trước đó nếu cần)
		if wsRole, ok := c.Get("workspace_role"); ok {
			role := wsRole.(string)
			if role == "OWNER" || role == "ADMIN" {
				c.Next()
				return
			}
		}

		// Kiểm tra từng permission slug, chỉ cần có 1 là đủ (OR logic)
		for _, slug := range requiredSlugs {
			has, err := m.projectMemberRepo.HasPermission(projectID, userID, slug)
			if err != nil {
				appresponse.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
				c.Abort()
				return
			}
			if has {
				c.Next()
				return
			}
		}

		appresponse.Fail(c, http.StatusForbidden, "FORBIDDEN", "Insufficient permissions in this project")
		c.Abort()
	}
}
