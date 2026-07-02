package handler

import (
	"strconv"

	"TaskFlow-Go/internal/dto"

	"github.com/gin-gonic/gin"
)

func parsePagination(c *gin.Context) dto.PaginationParam {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return dto.PaginationParam{Page: page, Limit: limit}
}
