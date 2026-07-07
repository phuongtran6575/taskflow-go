package validator

import (
	"regexp"
	"strings"
)

var hexColorRegex = regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)
var validImageExt = regexp.MustCompile(`(?i)\.(jpg|jpeg|png|webp|gif)$`)

func ValidateHexColor(color string) bool {
	return hexColorRegex.MatchString(color)
}

func IsValidProjectBackground(bg string) bool {
	if hexColorRegex.MatchString(bg) {
		return true
	}
	if !strings.HasPrefix(bg, "https://") {
		return false
	}
	return validImageExt.MatchString(bg)
}
