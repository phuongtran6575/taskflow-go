package validator

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"TaskFlow-Go/internal/shared/apperror"
)

func ValidateColumnTitle(title string) error {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" || utf8.RuneCountInString(trimmed) < 1 {
		return apperror.NewAppError(http.StatusBadRequest, "VALIDATION_ERROR", "Column title must not be empty or whitespace-only")
	}
	if utf8.RuneCountInString(trimmed) > 50 {
		return apperror.NewAppError(http.StatusBadRequest, "VALIDATION_ERROR", "Column title must be at most 50 characters")
	}
	return nil
}
