package implement

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
	_interface "TaskFlow-Go/internal/repository/interface"
)

type projectMemberRepository struct{ db *gorm.DB }

func NewProjectMemberRepository(db *gorm.DB) _interface.ProjectMemberRepository {
	return &projectMemberRepository{db: db}
}

func (r *projectMemberRepository) Update(member *models.ProjectMember) error {
	err := r.db.Model(&models.ProjectMember{}).
		Where("project_id = ? AND user_id = ?", member.ProjectID, member.UserID).
		Updates(member).Error
	return err
}

func (r *projectMemberRepository) ListAvailableWorkspaceMembers(workspaceID string, projectID string, search string, page int, limit int) ([]dto.AvailableWorkspaceMember, *dto.Pagination, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit

	var total int64
	countQuery := r.db.Table("workspace_members wm").
		Joins("JOIN users u ON u.id = wm.user_id").
		Where("wm.workspace_id = ?", workspaceID).
		Where("wm.user_id NOT IN (SELECT user_id FROM project_members WHERE project_id = ?)", projectID)
	if search != "" {
		countQuery = countQuery.Where("(u.full_name ILIKE ? OR u.email ILIKE ?)", "%"+search+"%", "%"+search+"%")
	}
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, nil, errors.New("failed to count available members")
	}

	var rows []struct {
		UserID        string  `gorm:"column:user_id"`
		FullName      string  `gorm:"column:full_name"`
		Username      string  `gorm:"column:username"`
		Email         string  `gorm:"column:email"`
		AvatarURL     *string `gorm:"column:avatar_url"`
		WorkspaceRole string  `gorm:"column:workspace_role"`
	}
	query := r.db.Table("workspace_members wm").
		Joins("JOIN users u ON u.id = wm.user_id").
		Select("u.id as user_id, u.full_name, u.username, u.email, u.avatar_url, wm.role as workspace_role").
		Where("wm.workspace_id = ?", workspaceID).
		Where("wm.user_id NOT IN (SELECT user_id FROM project_members WHERE project_id = ?)", projectID)
	if search != "" {
		query = query.Where("(u.full_name ILIKE ? OR u.email ILIKE ?)", "%"+search+"%", "%"+search+"%")
	}
	if err := query.Order("u.full_name ASC").Limit(limit).Offset(offset).Scan(&rows).Error; err != nil {
		return nil, nil, errors.New("failed to get available members")
	}

	members := make([]dto.AvailableWorkspaceMember, len(rows))
	for i, r := range rows {
		members[i] = dto.AvailableWorkspaceMember{
			UserID:        r.UserID,
			FullName:      r.FullName,
			Username:      r.Username,
			Email:         r.Email,
			AvatarURL:     r.AvatarURL,
			WorkspaceRole: r.WorkspaceRole,
		}
	}

	return members, &dto.Pagination{
		Total:      int(total),
		Page:       page,
		Limit:      limit,
		TotalPages: (int(total) + limit - 1) / limit,
	}, nil
}

func (r *projectMemberRepository) BulkAddMember(projectID string, users []dto.MemberRolePair) ([]models.ProjectMember, error) {
	members := make([]models.ProjectMember, len(users))
	for i, user := range users {
		members[i] = models.ProjectMember{
			ProjectID: projectID,
			UserID:    user.UserID,
			RoleID:    &user.RoleID,
		}
	}
	err := r.db.Create(&members).Error
	return members, err
}

func (r *projectMemberRepository) GetByIDWithRelationRole(projectID string, userID string) (*models.ProjectMember, error) {
	var member models.ProjectMember
	err := r.db.Preload("Role").Where("project_id = ? AND user_id = ?", projectID, userID).First(&member).Error
	return &member, err
}

func (r *projectMemberRepository) GetProjectMembersByRoleID(roleID string) ([]models.ProjectMember, error) {
	var members []models.ProjectMember
	err := r.db.Where("role_id = ?", roleID).Find(&members).Error
	return members, err
}

func (r *projectMemberRepository) GetByID(projectID, userID string) (*models.ProjectMember, error) {
	var member models.ProjectMember
	err := r.db.Where("project_id = ? AND user_id = ?", projectID, userID).First(&member).Error
	return &member, err
}

func (r *projectMemberRepository) ListByProjectID(projectID string) ([]models.ProjectMember, error) {
	var members []models.ProjectMember
	err := r.db.Where("project_id = ?", projectID).Find(&members).Error
	return members, err
}

func (r *projectMemberRepository) UpdateRole(projectID, userID string, roleID string) error {
	err := r.db.Model(&models.ProjectMember{}).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Update("role_id", roleID).Error
	return err
}

func (r *projectMemberRepository) ValidateMembersExist(projectID string, userIDs []string) ([]string, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	var found []string
	if err := r.db.Table("project_members").
		Where("project_id = ? AND user_id IN ?", projectID, userIDs).
		Pluck("user_id", &found).Error; err != nil {
		return nil, err
	}
	foundMap := make(map[string]struct{}, len(found))
	for _, id := range found {
		foundMap[id] = struct{}{}
	}
	var invalidIDs []string
	for _, uid := range userIDs {
		if _, ok := foundMap[uid]; !ok {
			invalidIDs = append(invalidIDs, uid)
		}
	}
	return invalidIDs, nil
}

func (r *projectMemberRepository) Delete(projectID, userID string) error {
	return r.db.Where("project_id = ? AND user_id = ?", projectID, userID).
		Delete(&models.ProjectMember{}).Error
}

func (r *projectMemberRepository) DeleteByWorkspace(workspaceID, userID string) error {
	return r.db.Where("user_id = ? AND project_id IN (SELECT id FROM projects WHERE workspace_id = ? AND deleted_at IS NULL)", userID, workspaceID).
		Delete(&models.ProjectMember{}).Error
}

func (r *projectMemberRepository) HasPermission(projectID, userID, permissionSlug string) (bool, error) {
	var count int64
	err := r.db.Table("project_members pm").
		Joins("JOIN role_permissions rp ON rp.role_id = pm.role_id").
		Joins("JOIN permissions p ON p.id = rp.permission_id").
		Where("pm.project_id = ? AND pm.user_id = ? AND p.slug = ?", projectID, userID, permissionSlug).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

type projectMemberRow struct {
	UserID        string    `gorm:"column:user_id"`
	FullName      string    `gorm:"column:full_name"`
	Username      string    `gorm:"column:username"`
	Email         string    `gorm:"column:email"`
	AvatarURL     *string   `gorm:"column:avatar_url"`
	WorkspaceRole string    `gorm:"column:workspace_role"`
	RoleID        *string   `gorm:"column:role_id"`
	RoleName      *string   `gorm:"column:role_name"`
	IsFavorite    bool      `gorm:"column:is_favorite"`
	JoinedAt      time.Time `gorm:"column:joined_at"`
}

func (r *projectMemberRepository) ListWithPagination(projectID string, search string, roleID string, page int, limit int) ([]dto.ProjectMemberInfo, *dto.Pagination, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit

	var total int64
	countQuery := r.db.Table("project_members pm").
		Joins("JOIN projects p ON p.id = pm.project_id").
		Joins("JOIN users u ON u.id = pm.user_id")
	if search != "" {
		countQuery = countQuery.Where("(u.full_name ILIKE ? OR u.email ILIKE ?)", "%"+search+"%", "%"+search+"%")
	}
	if roleID != "" {
		countQuery = countQuery.Where("pm.role_id = ?", roleID)
	}
	if err := countQuery.Where("pm.project_id = ?", projectID).Count(&total).Error; err != nil {
		return nil, nil, errors.New("failed to count members")
	}

	query := r.db.Table("project_members pm").
		Joins("JOIN projects p ON p.id = pm.project_id").
		Joins("JOIN users u ON u.id = pm.user_id").
		Joins("JOIN workspace_members wm ON wm.workspace_id = p.workspace_id AND wm.user_id = pm.user_id").
		Joins("LEFT JOIN roles r ON r.id = pm.role_id").
		Select("u.id as user_id, u.full_name, u.username, u.email, u.avatar_url, wm.role as workspace_role, pm.role_id, r.name as role_name, pm.is_favorite, pm.joined_at").
		Where("pm.project_id = ?", projectID)
	if search != "" {
		query = query.Where("(u.full_name ILIKE ? OR u.email ILIKE ?)", "%"+search+"%", "%"+search+"%")
	}
	if roleID != "" {
		query = query.Where("pm.role_id = ?", roleID)
	}
	var rows []projectMemberRow
	if err := query.Order("pm.joined_at DESC").Limit(limit).Offset(offset).Scan(&rows).Error; err != nil {
		return nil, nil, errors.New("failed to get members")
	}

	members := make([]dto.ProjectMemberInfo, len(rows))
	for i, r := range rows {
		var roleRef *dto.RoleRef
		if r.RoleID != nil {
			roleName := ""
			if r.RoleName != nil {
				roleName = *r.RoleName
			}
			roleRef = &dto.RoleRef{
				ID:   *r.RoleID,
				Name: roleName,
			}
		}
		members[i] = dto.ProjectMemberInfo{
			UserID:        r.UserID,
			FullName:      r.FullName,
			Username:      r.Username,
			Email:         r.Email,
			AvatarURL:     r.AvatarURL,
			WorkspaceRole: r.WorkspaceRole,
			ProjectRole:   roleRef,
			IsFavorite:    r.IsFavorite,
			JoinedAt:      r.JoinedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return members, &dto.Pagination{
		Total:      int(total),
		Page:       page,
		Limit:      limit,
		TotalPages: (int(total) + limit - 1) / limit,
	}, nil
}
