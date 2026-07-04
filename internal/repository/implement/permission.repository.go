package implement

import (
	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
	_interface "TaskFlow-Go/internal/repository/interface"
)

type permissionRepository struct{ db *gorm.DB }

func NewPermissionRepository(db *gorm.DB) _interface.PermissionRepository {
	return &permissionRepository{db: db}
}

func (r *permissionRepository) GetListPermissions(permissionIDs []string) ([]dto.PermissionAssignedInfo, error) {
	var permissions []dto.PermissionAssignedInfo
	err := r.db.Table("permissions").
		Select("id, slug").
		Where("id IN ?", permissionIDs).
		Scan(&permissions).Error
	if err != nil {
		return nil, err
	}
	return permissions, nil
}

func (r *permissionRepository) GetListPermissionsByModule(permissionIDs []string) (map[string][]dto.PermissionInfo, *int, error) {
	var permissions []dto.PermissionInfo
	err := r.db.Table("permissions").
		Select("id, slug, module, description, is_system").
		Where("id IN ?", permissionIDs).
		Scan(&permissions).Error
	if err != nil {
		return nil, nil, err
	}
	count := len(permissions)
	listByModule := make(map[string][]dto.PermissionInfo)
	for _, p := range permissions {
		listByModule[p.Module] = append(listByModule[p.Module], p)
	}

	return listByModule, &count, nil
}

func (r *permissionRepository) ValidatePermissionIDs(ids []string) (foundIDs []string, invalidIDs []string, err error) {
	var found []string
	err = r.db.Table("permissions").Where("id IN ?", ids).Pluck("id", &found).Error
	if err != nil {
		return nil, nil, err
	}
	foundSet := make(map[string]struct{}, len(found))
	for _, id := range found {
		foundSet[id] = struct{}{}
	}
	for _, id := range ids {
		if _, ok := foundSet[id]; !ok {
			invalidIDs = append(invalidIDs, id)
		}
	}
	return found, invalidIDs, nil
}

func (r *permissionRepository) GetByID(id string) (*models.Permission, error) {
	var p models.Permission
	err := r.db.Where("id = ?", id).First(&p).Error
	return &p, err
}

func (r *permissionRepository) List() ([]dto.PermissionInfo, error) {
	var permissions []dto.PermissionInfo
	err := r.db.Table("permissions").Select("id, slug, module, description, is_system").Scan(&permissions).Error
	return permissions, err
}

func (r *permissionRepository) GetByIDOrSlug(idOrSlug string) (*dto.PermissionDetailResponse, error) {
	var p models.Permission
	err := r.db.Where("id = ? OR slug = ?", idOrSlug, idOrSlug).First(&p).Error
	if err != nil {
		return nil, err
	}
	desc := ""
	if p.Description != nil {
		desc = *p.Description
	}
	return &dto.PermissionDetailResponse{
		ID:          p.ID,
		Slug:        p.Slug,
		Module:      p.Module,
		Description: desc,
		IsSystem:    p.IsSystem,
	}, nil
}

func (r *permissionRepository) ListModules(descriptions map[string]string) ([]dto.ModuleInfo, error) {
	modules := []dto.ModuleInfo{}
	err := r.db.Table("permissions").
		Select("module as name, count(*) as permission_count").
		Group("module").
		Scan(&modules).Error
	if err != nil {
		return nil, err
	}
	for i := range modules {
		if desc, ok := descriptions[modules[i].Name]; ok {
			modules[i].Description = desc
		}
	}

	return modules, nil
}

func (r *permissionRepository) GetByModule(module string) (*dto.ModulePermissionsResponse, error) {
	var rows []struct {
		ID          string `gorm:"column:id"`
		Slug        string `gorm:"column:slug"`
		Module      string `gorm:"column:module"`
		Description string `gorm:"column:description"`
		IsSystem    bool   `gorm:"column:is_system"`
	}
	err := r.db.Table("permissions").
		Select("id, slug, module, COALESCE(description, '') AS description, is_system").
		Where("module = ?", module).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	permissions := make([]dto.PermissionInfo, len(rows))
	for i, r := range rows {
		permissions[i] = dto.PermissionInfo{
			ID:          r.ID,
			Slug:        r.Slug,
			Module:      r.Module,
			Description: r.Description,
			IsSystem:    r.IsSystem,
		}
	}
	return &dto.ModulePermissionsResponse{
		Module: module,
		Data:   permissions,
		Total:  len(permissions),
	}, nil
}
