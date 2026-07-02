package _interface

import "TaskFlow-Go/internal/dto"

type StorageService interface {
	GetWorkspaceStorageUsage(workspaceID string, userID string) (*dto.StorageUsageResponse, error)
}
