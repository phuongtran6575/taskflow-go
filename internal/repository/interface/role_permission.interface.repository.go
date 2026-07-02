package _interface

import "TaskFlow-Go/internal/models"

type RolePermissionRepository interface {
	GetPermissionsByRoleID(roleID string) ([]models.RolePermission, error)

	BulkCreate(roleID string, permissionIDs []string) error
	BulkDelete(roleID string, permissionIDs []string) error
}
