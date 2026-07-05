package helper

import (
	"path/filepath"
	"strings"
	"unicode"
)

func SanitizeFileName(name string) string {
	if name == "" {
		return "unnamed_file"
	}

	ext := filepath.Ext(name)
	baseName := name
	if ext != "" {
		baseName = name[:len(name)-len(ext)]
	}

	baseName = strings.TrimSpace(baseName)

	if len(baseName) > 255-len(ext) {
		baseName = baseName[:255-len(ext)]
	}

	baseName = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, baseName)

	baseName = strings.NewReplacer("/", "_", "\\", "_").Replace(baseName)

	unsafe := []string{"<", ">", ":", "\"", "|", "?", "*"}
	for _, c := range unsafe {
		baseName = strings.ReplaceAll(baseName, c, "_")
	}

	baseName = strings.TrimSpace(baseName)
	if baseName == "" {
		return "unnamed_file" + ext
	}

	result := baseName + ext
	if len(result) > 255 {
		result = result[:255]
	}

	if strings.IndexFunc(result, func(r rune) bool {
		return r < 0x20 || r == 0x7f
	}) != -1 {
		return "unnamed_file" + ext
	}

	_ = unicode.IsPrint

	return result
}
