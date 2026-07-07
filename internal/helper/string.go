package helper

import "strings"

func DedupStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; !ok {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

func MaskIP(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return ip
	}
	return parts[0] + "." + parts[1] + ".x.x"
}

func TruncateContent(content string, maxLen int) string {
	runes := []rune(content)
	if len(runes) > maxLen {
		return string(runes[:maxLen]) + "..."
	}
	return content
}

func NormalizeColor(color string) string {
	color = strings.ToUpper(color)
	if len(color) == 4 {
		color = "#" + string(color[1]) + string(color[1]) + string(color[2]) + string(color[2]) + string(color[3]) + string(color[3])
	}
	return color
}
