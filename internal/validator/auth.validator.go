package validator

import (
	"regexp"
	"strings"
)

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

func NewValidationError(msg string) *ValidationError {
	return &ValidationError{Message: msg}
}

var usernameRegex = regexp.MustCompile(`^[a-z][a-z0-9_]{2,29}$`)

func ValidateUsername(username string) error {
	if len(username) < 3 || len(username) > 30 {
		return NewValidationError("Username must be between 3 and 30 characters")
	}
	if username != strings.ToLower(username) {
		return NewValidationError("Username must not contain uppercase letters")
	}
	if strings.Contains(username, " ") {
		return NewValidationError("Username must not contain spaces")
	}
	if !usernameRegex.MatchString(username) {
		return NewValidationError("Username must start with a letter and contain only a-z, 0-9, underscores")
	}
	return nil
}
