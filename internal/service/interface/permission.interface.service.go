package _interface

import "TaskFlow-Go/internal/dto"

type PermissionService interface {
	ListFlatPermissions(userID string) (*dto.PermissionFlatResponse, error)
	ListGroupedPermissions(userID string) (*dto.PermissionGroupedResponse, error)
	ListModules(userID string) (*dto.ModuleListResponse, error)
	GetPermissionsByModule(userID string, module string) (*dto.ModulePermissionsResponse, error)
	GetPermissionByIdOrSlug(userID string, idOrSlug string) (*dto.PermissionDetailResponse, error)

	// GetPermissionsETag trả về ETag hash của danh sách permissions.
	// Dùng cho client-side caching (If-None-Match).
	GetPermissionsETag() string
}
