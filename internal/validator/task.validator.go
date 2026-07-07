package validator

import (
	"TaskFlow-Go/internal/models"
	"TaskFlow-Go/internal/shared/apperror"
)

func ValidateTaskPriority(priority string) error {
	if priority == "" {
		return nil
	}
	switch models.TaskPriority(priority) {
	case models.TaskPriorityLOW, models.TaskPriorityMED, models.TaskPriorityHIGH, models.TaskPriorityURGENT:
		return nil
	default:
		return apperror.ErrInvalidPriority
	}
}
