package implement

import (
	"gorm.io/gorm"

	"TaskFlow-Go/internal/models"
	_interface "TaskFlow-Go/internal/repository/interface"
)

type rolePermissionRepository struct{ db *gorm.DB }

// BulkDelete implements [_interface.RolePermissionRepository].

func NewRolePermissionRepository(db *gorm.DB) _interface.RolePermissionRepository {
	return &rolePermissionRepository{db: db}
}
func (r *rolePermissionRepository) BulkDelete(roleID string, permissionIDs []string) error {
	rolePermissons := make([]models.RolePermission, 0, len(permissionIDs))
	for _, permissionID := range permissionIDs {
		rolePermissons = append(rolePermissons, models.RolePermission{
			RoleID:       roleID,
			PermissionID: permissionID,
		})

	}
	return r.db.Delete(&rolePermissons).Error
}

func (r *rolePermissionRepository) BulkCreate(roleID string, permissionIDs []string) error {
	rolePermissions := make([]models.RolePermission, 0, len(permissionIDs))

	for _, permissionID := range permissionIDs {
		rolePermissions = append(rolePermissions, models.RolePermission{
			RoleID:       roleID,
			PermissionID: permissionID,
		})
	}

	return r.db.Create(&rolePermissions).Error

	/*rolePermissions := make([]models.RolePermission, len(permissionIDs))

	for i, permissionID := range permissionIDs {
		rolePermissions[i] = models.RolePermission{
			RoleID:       roleID,
			PermissionID: permissionID,
		}
	}

	return r.db.
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&rolePermissions).Error*/
}

func (r *rolePermissionRepository) GetPermissionsByRoleID(roleID string) ([]models.RolePermission, error) {
	var rps []models.RolePermission
	err := r.db.Where("role_id = ?", roleID).Find(&rps).Error
	return rps, err
}
