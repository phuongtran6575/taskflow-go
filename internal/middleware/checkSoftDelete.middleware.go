package middleware

import (
	"errors"
	"net/http"

	"TaskFlow-Go/internal/shared/apperror"
	"TaskFlow-Go/internal/shared/appresponse"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RequireProjectNotArchived kiểm tra project không ở trạng thái archived.
// Áp dụng cho tất cả write operations (tạo/sửa task, column, comment...)
// BR-PROJ-05: Archived project chặn mọi hành động ghi
func (m *Middleware) RequireProjectNotArchived() gin.HandlerFunc {
	return func(c *gin.Context) {
		projectID := c.Param("project_id")
		if projectID == "" {
			appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "project_id is required")
			c.Abort()
			return
		}

		project, err := m.projectRepo.GetByID(projectID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				appresponse.Fail(c, http.StatusNotFound, apperror.ErrProjectNotFound.Code, apperror.ErrProjectNotFound.Message)
				c.Abort()
				return
			}
			appresponse.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check project status")
			c.Abort()
			return
		}

		if project.IsArchived {
			appresponse.Fail(c, http.StatusForbidden, apperror.ErrProjectArchived.Code, apperror.ErrProjectArchived.Message)
			c.Abort()
			return
		}

		c.Next()
	}
}
