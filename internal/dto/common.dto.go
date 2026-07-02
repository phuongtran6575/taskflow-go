package dto

type Pagination struct {
	Total      int `json:"total"`
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalPages int `json:"total_pages"`
}

type PaginationParam struct {
	Page  int `form:"page" json:"page"`
	Limit int `form:"limit" json:"limit"`
}

func (p PaginationParam) Offset() int {
	page := p.Page
	if page <= 0 {
		page = 1
	}
	limit := p.Limit
	if limit <= 0 {
		limit = 20
	}
	return (page - 1) * limit
}

func NewPagination(total int64, param PaginationParam) *Pagination {
	page := param.Page
	if page <= 0 {
		page = 1
	}
	limit := param.Limit
	if limit <= 0 {
		limit = 20
	}
	totalPages := (int(total) + limit - 1) / limit
	return &Pagination{
		Total:      int(total),
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}
}

