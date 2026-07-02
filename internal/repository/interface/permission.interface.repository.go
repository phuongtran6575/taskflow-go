package _interface

import (
	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
)

type PermissionRepository interface {
	GetByID(id string) (*models.Permission, error)
	List() ([]dto.PermissionInfo, error)

	GetByIDOrSlug(idOrSlug string) (*dto.PermissionDetailResponse, error)
	ListModules(description map[string]string) ([]dto.ModuleInfo, error)
	GetByModule(module string) (*dto.ModulePermissionsResponse, error)
	GetListPermissions(permissionIDs []string) ([]dto.PermissionAssignedInfo, error)
	GetListPermissionsByModule(permissionIDs []string) (map[string][]dto.PermissionInfo, *int, error)
}
