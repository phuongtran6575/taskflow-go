package validator

import (
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	ErrNameEmpty         = errors.New("workspace name cannot be empty or consist only of whitespace")
	ErrNameLength        = errors.New("workspace name must be between 2 and 100 characters")
	ErrDomainLength      = errors.New("workspace domain must be between 3 and 50 characters")
	ErrDomainFormat      = errors.New("workspace domain must start and end with a letter or digit, and only contain lowercase letters, digits, and hyphens")
	ErrDomainBlacklisted = errors.New("workspace domain is reserved or blacklisted")

	domainRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$`)

	domainBlacklist = map[string]bool{
		"admin":   true,
		"api":     true,
		"app":     true,
		"www":     true,
		"mail":    true,
		"support": true,
		"help":    true,
		"login":   true,
		"signup":  true,
		"billing": true,
		"static":  true,
		"cdn":     true,
		"dev":     true,
		"staging": true,
	}
)

// ValidateWorkspaceName validates workspace name according to business rules:
// - Length: 2 - 100 characters (runes)
// - Accept all Unicode characters (supports Vietnamese)
// - Must not consist of only whitespace
func ValidateWorkspaceName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ErrNameEmpty
	}

	runeCount := utf8.RuneCountInString(name)
	if runeCount < 2 || runeCount > 100 {
		return ErrNameLength
	}

	return nil
}

// ValidateWorkspaceDomain validates workspace domain according to business rules:
// - Length: 3 - 50 characters
// - Valid characters: a-z, 0-9, and hyphen (-)
// - Must start and end with a letter/digit
// - No uppercase characters
// - No special characters except (-)
// - Must not be in blacklist
func ValidateWorkspaceDomain(domain string) error {
	if len(domain) < 3 || len(domain) > 50 {
		return ErrDomainLength
	}

	if !domainRegex.MatchString(domain) {
		return ErrDomainFormat
	}

	if domainBlacklist[domain] {
		return ErrDomainBlacklisted
	}

	return nil
}

