package helper

import "fmt"

var blockedExtensions = map[string]bool{
	"exe": true, "msi": true, "dll": true, "scr": true, "pif": true, "com": true,
	"bat": true, "cmd": true, "vbs": true, "vbe": true,
	"js": true, "jse": true, "wsf": true, "wsh": true,
	"ps1": true, "ps2": true,
	"sh": true, "bash": true, "zsh": true, "fish": true, "csh": true,
}

var fileGroups = map[string]string{
	"jpg": "image", "jpeg": "image", "png": "image", "gif": "image",
	"webp": "image", "svg": "image", "bmp": "image", "tiff": "image",
	"tif": "image", "ico": "image", "heic": "image", "heif": "image",

	"pdf": "document", "doc": "document", "docx": "document",
	"xls": "document", "xlsx": "document", "ppt": "document", "pptx": "document",
	"txt": "document", "md": "document", "csv": "document",
	"odt": "document", "ods": "document", "odp": "document",
	"rtf": "document", "pages": "document", "numbers": "document", "key": "document",

	"mp4": "video", "mov": "video", "avi": "video", "mkv": "video",
	"webm": "video", "flv": "video", "wmv": "video", "m4v": "video", "3gp": "video",

	"mp3": "audio", "wav": "audio", "ogg": "audio", "flac": "audio",
	"aac": "audio", "m4a": "audio", "wma": "audio", "opus": "audio",

	"zip": "archive", "rar": "archive", "7z": "archive", "tar": "archive",
	"gz": "archive", "bz2": "archive", "xz": "archive", "dmg": "archive", "iso": "archive",

	"ts": "code", "py": "code", "java": "code", "cs": "code",
	"cpp": "code", "c": "code", "go": "code", "rs": "code", "rb": "code",
	"php": "code", "html": "code", "css": "code", "scss": "code",
	"json": "code", "xml": "code", "yaml": "code", "yml": "code",
	"sql": "code",

	"fig": "design", "sketch": "design", "xd": "design",
	"ai": "design", "psd": "design", "ae": "design", "indd": "design",
}

var executableExtensions = map[string]bool{
	"exe": true, "msi": true, "dll": true, "scr": true, "pif": true, "com": true,
	"bat": true, "cmd": true, "vbs": true, "vbe": true,
	"jse": true, "wsf": true, "wsh": true,
	"ps1": true, "ps2": true,
	"sh": true, "bash": true, "zsh": true, "fish": true, "csh": true,
}

func IsBlockedExtension(ext string) bool {
	return blockedExtensions[ext]
}

func ClassifyFileType(ext string) string {
	if group, ok := fileGroups[ext]; ok {
		return group
	}
	return "other"
}

func IsExecutableExtension(ext string) bool {
	return executableExtensions[ext]
}

func FormatSizeDisplay(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.0f KB", float64(bytes)/1024)
	}
	if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.0f MB", float64(bytes)/(1024*1024))
	}
	return fmt.Sprintf("%.0f GB", float64(bytes)/(1024*1024*1024))
}
