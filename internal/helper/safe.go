package helper

import "TaskFlow-Go/internal/dto"

func SafeProjectName(p *dto.ActivityLogProjectRef) string {
	if p == nil {
		return ""
	}
	return p.Name
}

func SafeStr2(ws interface{}, fallback string) string {
	return fallback
}
