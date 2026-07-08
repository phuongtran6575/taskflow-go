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
//
// BR-PRA-03: Workspace OWNER bypass — tự động có quyền trong mọi project
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

		// BR-PRA-03: Workspace OWNER bypass — không cần trong project_members
		if wsRole, ok := c.Get("workspace_role"); ok {
			if wsRole.(string) == "OWNER" {
				c.Next()
				return
			}
		}

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
// Workspace OWNER bypass check này (BR-PRA-03). ADMIN không được bypass.
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

		// Workspace OWNER/ADMIN bypass project-level RBAC (BR-PERM-06).
		// Nếu workspace_role chưa có trong context (route không gọi RequireWorkspaceRole),
		// tự động query từ DB.
		wsRoleVal, ok := c.Get("workspace_role")
		if !ok {
			wsID := c.Param("workspace_id")
			if wsID != "" {
				member, err := m.workspaceMemberRepo.GetByID(wsID, userID)
				if err == nil {
					wsRoleVal = string(member.Role)
					c.Set("workspace_role", wsRoleVal)
				}
			}
		}
		if wsRoleStr, ok := wsRoleVal.(string); ok {
			if wsRoleStr == "OWNER" || wsRoleStr == "ADMIN" {
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
