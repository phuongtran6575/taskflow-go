package appresponse

import "TaskFlow-Go/internal/dto"

type Response struct {
	Success bool            `json:"success"`
	Data    interface{}     `json:"data,omitempty"`
	Error   *ErrorInfo      `json:"error,omitempty"`
	Meta    *dto.Pagination `json:"meta,omitempty"`
}
