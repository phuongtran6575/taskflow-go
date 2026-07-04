package implement

import (
	"errors"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
	_interface "TaskFlow-Go/internal/repository/interface"
)

type roleRepository struct{ db *gorm.DB }

func NewRoleRepository(db *gorm.DB) _interface.RoleRepository {
	return &roleRepository{db: db}
}

func (r *roleRepository) Create(role *models.Role) (*models.Role, error) {
	err := r.db.Create(&role).Error
	if err != nil {
		return nil, errors.New("failed to create role")
	}
	return role, nil
}

func (r *roleRepository) GetByID(id string) (*models.Role, error) {
	var role models.Role
	err := r.db.Where("id = ?", id).First(&role).Error
	return &role, err
}

func (r *roleRepository) CountByWorkspaceID(workspaceID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Role{}).Where("workspace_id = ?", workspaceID).Count(&count).Error
	return count, err
}

func (r *roleRepository) ListByWorkspaceID(workspaceID string) ([]models.Role, error) {
	var roles []models.Role
	err := r.db.Where("workspace_id = ?", workspaceID).Find(&roles).Error
	return roles, err
}

func (r *roleRepository) GetAffectedProjectsByRoleID(roleID string) ([]dto.AffectedProject, int, error) {
	var results []struct {
		ProjectID   string `gorm:"column:project_id"`
		ProjectName string `gorm:"column:project_name"`
		MemberCount int    `gorm:"column:member_count"`
	}
	err := r.db.Table("project_members pm").
		Joins("JOIN projects p ON p.id = pm.project_id").
		Where("pm.role_id = ?", roleID).
		Select("p.id as project_id, p.name as project_name, COUNT(DISTINCT pm.user_id) as member_count").
		Group("p.id, p.name").
		Scan(&results).Error
	if err != nil {
		return nil, 0, err
	}
	projects := make([]dto.AffectedProject, len(results))
	total := 0
	for i, r := range results {
		projects[i] = dto.AffectedProject{
			ProjectID:   r.ProjectID,
			ProjectName: r.ProjectName,
			MemberCount: r.MemberCount,
		}
		total += r.MemberCount
	}
	return projects, total, nil
}

func (r *roleRepository) Update(role *models.Role) error {
	return r.db.Save(role).Error
}

func (r *roleRepository) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.Role{}).Error
}

func (r *roleRepository) ListWithPagination(workspaceID string, search string, page int, limit int) ([]dto.RoleSummary, *dto.Pagination, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit

	var total int64
	countQuery := r.db.Table("roles r").Where("r.workspace_id = ?", workspaceID)
	if search != "" {
		countQuery = countQuery.Where("r.name ILIKE ?", "%"+search+"%")
	}
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, nil, errors.New("failed to count roles")
	}

	var roles []dto.RoleSummary
	query := r.db.Table("roles r").
		Joins("LEFT JOIN role_permissions rp ON rp.role_id = r.id").
		Joins("LEFT JOIN project_members pm ON pm.role_id = r.id").
		Select("r.id as id, r.name as name, COALESCE(r.description, '') as description, r.updated_at as updated_at, COUNT(DISTINCT rp.permission_id) as permission_count, COUNT(DISTINCT pm.user_id) as member_count").
		Where("r.workspace_id = ?", workspaceID).
		Group("r.id, r.name, r.description, r.updated_at").
		Order("r.updated_at DESC").
		Limit(limit).
		Offset(offset)
	if search != "" {
		query = query.Where("r.name ILIKE ?", "%"+search+"%")
	}
	if err := query.Scan(&roles).Error; err != nil {
		return nil, nil, errors.New("failed to get roles")
	}

	if roles == nil {
		roles = []dto.RoleSummary{}
	}

	return roles, &dto.Pagination{
		Total:      int(total),
		Page:       page,
		Limit:      limit,
		TotalPages: (int(total) + limit - 1) / limit,
	}, nil
}

func (r *roleRepository) GetByIDWithDetail(workspaceID string, roleID string) (*dto.RoleDetailResponse, error) {
	var role models.Role
	err := r.db.Where("id = ? AND workspace_id = ?", roleID, workspaceID).First(&role).Error
	if err != nil {
		return nil, err
	}
	var members []dto.MemberPreview
	err = r.db.Table("project_members pm").
		Joins("JOIN users u ON u.id = pm.user_id").
		Joins("JOIN roles r ON r.id = pm.role_id").
		Where("pm.role_id = ?", role.ID).
		Select("u.id as user_id, u.full_name as full_name, u.avatar_url as avatar_url").Scan(&members).Error
	if err != nil {
		return nil, errors.New("failed to get members")
	}
	var permissions []models.Permission
	err = r.db.Table("permissions p").
		Joins("JOIN role_permissions as rp ON rp.permission_id = p.id").
		Where("rp.role_id = ?", role.ID).
		Find(&permissions).Error
	if err != nil {
		return nil, errors.New("failed to get permissions")
	}
	listByModule := make(map[string][]struct {
		ID          string `json:"id"`
		Slug        string `json:"slug"`
		Description string `json:"description"`
	})
	for _, p := range permissions {
		listByModule[p.Module] = append(listByModule[p.Module], struct {
			ID          string `json:"id"`
			Slug        string `json:"slug"`
			Description string `json:"description"`
		}{
			ID:          p.ID,
			Slug:        p.Slug,
			Description: *p.Description,
		})
	}

	desc := ""
	if role.Description != nil {
		desc = *role.Description
	}
	return &dto.RoleDetailResponse{
		ID:              role.ID,
		Name:            role.Name,
		Description:     desc,
		UpdatedAt:       role.UpdatedAt,
		PermissionCount: len(permissions),
		MemberCount:     len(members),
		MembersPreview:  members,
		Permissions:     listByModule,
	}, nil
}
