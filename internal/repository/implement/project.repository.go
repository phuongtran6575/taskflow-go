package implement

import (
	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
	"TaskFlow-Go/internal/projection"
	_interface "TaskFlow-Go/internal/repository/interface"
)

type projectRepository struct{ db *gorm.DB }

func (r *projectRepository) WithTx(tx *gorm.DB) _interface.ProjectRepository {
	return &projectRepository{db: tx}
}

func NewProjectRepository(db *gorm.DB) _interface.ProjectRepository {
	return &projectRepository{db: db}
}

func (r *projectRepository) GetCreateProjectResponse(id string) (*dto.ProjectCreateResponse, error) {
	var rows []projection.ProjectCreateRow
	err := r.db.Table("projects p").
		Select("p.id, p.name, p.key, p.icon, p.is_archived, p.background, p.created_at, u.id as owner_id, u.full_name, u.avatar_url, c.id as column_id, c.title, c.position").
		Joins("join users u on u.id = p.owner_id").
		Joins("join columns c on c.project_id = p.id").
		Where("p.id = ?", id).
		Order("c.position").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := dto.ProjectCreateResponse{
		ID:         rows[0].ID,
		Name:       rows[0].Name,
		Key:        rows[0].Key,
		Icon:       rows[0].Icon,
		IsArchived: rows[0].IsArchived,
		Background: rows[0].Background,
		CreatedAt:  rows[0].CreatedAt,
		Owner: dto.ProjectOwnerInfo{
			UserID:    rows[0].OwnerID,
			FullName:  rows[0].FullName,
			AvatarURL: rows[0].AvatarURL,
		},
	}

	result.DefaultColumns = make([]dto.DefaultColumn, len(rows))
	for i, r := range rows {
		result.DefaultColumns[i] = dto.DefaultColumn{
			ID:       r.ColumnID,
			Title:    r.Title,
			Position: r.Position,
		}
	}

	return &result, nil
}

func (r *projectRepository) GetListMemberProject(workspaceID string, userID string, isArchived *bool, isFavorite *bool, search string, param dto.PaginationParam) ([]dto.ProjectSummary, *dto.Pagination, error) {
	buildQuery := func(tx *gorm.DB) *gorm.DB {
		q := tx.Where("p.workspace_id = ? and pm.user_id = ?", workspaceID, userID)
		if isArchived != nil {
			q = q.Where("p.is_archived = ?", *isArchived)
		}
		if isFavorite != nil {
			q = q.Where("pm.is_favorite = ?", *isFavorite)
		}
		if search != "" {
			q = q.Where("p.name ILIKE ?", "%"+search+"%")
		}
		return q
	}

	var total int64
	var rows []projection.ProjectSummaryRow

	if err := buildQuery(r.db.Table("projects p").
		Joins("join project_members pm on pm.project_id = p.id")).
		Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, nil, err
	}

	if err := buildQuery(r.db.Table("projects p").
		Joins("join users u on u.id = p.owner_id").
		Joins("join project_members pm on pm.project_id = p.id").
		Joins("left join roles r on r.id = pm.role_id")).
		Select("p.id, p.name, p.key, p.icon, p.is_archived, p.background, p.created_at, p.updated_at, "+
			"p.owner_id, u.avatar_url, u.full_name, pm.is_favorite, "+
			"r.id as role_id, r.name as role_name, pm.joined_at, "+
			"(select count(*) from project_members pm2 where pm2.project_id = p.id) as member_count, "+
			"(select count(*) from tasks t where t.project_id = p.id and t.deleted_at is null) as open_task_count").
		Offset(param.Offset()).Limit(param.Limit).Order("p.created_at DESC").
		Scan(&rows).Error; err != nil {
		return nil, nil, err
	}

	projects := make([]dto.ProjectSummary, len(rows))
	for i, row := range rows {
		summary := dto.ProjectSummary{
			ID:            row.ID,
			Name:          row.Name,
			Key:           row.Key,
			Icon:          row.Icon,
			Background:    row.Background,
			IsArchived:    row.IsArchived,
			IsFavorite:    row.IsFavorite,
			MemberCount:   row.MemberCount,
			OpenTaskCount: row.OpenTaskCount,
			Owner: dto.ProjectOwnerInfo{
				UserID:    row.OwnerID,
				FullName:  row.FullName,
				AvatarURL: row.AvatarURL,
			},
			JoinedAt:  row.JoinedAt,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
			MyRole:    nil,
		}
		if row.RoleID != nil {
			summary.MyRole = &dto.RoleRef{
				ID:   *row.RoleID,
				Name: *row.RoleName,
			}
		}
		projects[i] = summary
	}

	return projects, dto.NewPagination(total, param), nil
}

func (r *projectRepository) GetListWorkspaceProject(workspaceID string, userID string, isArchived *bool, isFavorite *bool, search string, param dto.PaginationParam) ([]dto.ProjectSummary, *dto.Pagination, error) {
	buildQuery := func(tx *gorm.DB) *gorm.DB {
		q := tx.Where("p.workspace_id = ?", workspaceID)
		if isArchived != nil {
			q = q.Where("p.is_archived = ?", *isArchived)
		}
		if search != "" {
			q = q.Where("p.name ILIKE ?", "%"+search+"%")
		}
		return q
	}

	var total int64
	var rows []projection.ProjectSummaryRow

	if err := buildQuery(r.db.Table("projects p")).
		Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, nil, err
	}

	if err := buildQuery(r.db.Table("projects p").
		Joins("join users u on u.id = p.owner_id").
		Joins("left join project_members pm on pm.project_id = p.id and pm.user_id = ?", userID).
		Joins("left join roles r on r.id = pm.role_id")).
		Select("p.id, p.name, p.key, p.icon, p.is_archived, p.background, p.created_at, p.updated_at, "+
			"p.owner_id, u.avatar_url, u.full_name, "+
			"coalesce(pm.is_favorite, false) as is_favorite, "+
			"r.id as role_id, r.name as role_name, "+
			"coalesce(pm.joined_at, p.created_at) as joined_at, "+
			"(select count(*) from project_members pm2 where pm2.project_id = p.id) as member_count, "+
			"(select count(*) from tasks t where t.project_id = p.id and t.deleted_at is null) as open_task_count").
		Offset(param.Offset()).Limit(param.Limit).Order("p.created_at DESC").
		Scan(&rows).Error; err != nil {
		return nil, nil, err
	}

	projects := make([]dto.ProjectSummary, len(rows))
	for i, row := range rows {
		summary := dto.ProjectSummary{
			ID:            row.ID,
			Name:          row.Name,
			Key:           row.Key,
			Icon:          row.Icon,
			Background:    row.Background,
			IsArchived:    row.IsArchived,
			IsFavorite:    row.IsFavorite,
			MemberCount:   row.MemberCount,
			OpenTaskCount: row.OpenTaskCount,
			Owner: dto.ProjectOwnerInfo{
				UserID:    row.OwnerID,
				FullName:  row.FullName,
				AvatarURL: row.AvatarURL,
			},
			JoinedAt:  row.JoinedAt,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
			MyRole:    nil,
		}
		if row.RoleID != nil {
			summary.MyRole = &dto.RoleRef{
				ID:   *row.RoleID,
				Name: *row.RoleName,
			}
		}
		projects[i] = summary
	}

	return projects, dto.NewPagination(total, param), nil
}

func (r *projectRepository) Create(project *models.Project) error {
	return r.db.Create(project).Error
}

func (r *projectRepository) GetByID(id string) (*models.Project, error) {
	var project models.Project
	err := r.db.Where("id = ?", id).First(&project).Error
	return &project, err
}

func (r *projectRepository) ListByWorkspaceID(workspaceID string) ([]models.Project, error) {
	var projects []models.Project
	err := r.db.Where("workspace_id = ?", workspaceID).Find(&projects).Error
	return projects, err
}

func (r *projectRepository) Update(project *models.Project) error {
	return r.db.Save(project).Error
}

func (r *projectRepository) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.Project{}).Error
}

func (r *projectRepository) GetByIDWithDetail(workspaceID string, userID string, projectID string) (*dto.ProjectDetailResponse, error) {
	var project models.Project
	if err := r.db.Where("id = ? AND workspace_id = ?", projectID, workspaceID).First(&project).Error; err != nil {
		return nil, err
	}

	var owner struct {
		UserID    string  `gorm:"column:user_id"`
		FullName  string  `gorm:"column:full_name"`
		AvatarURL *string `gorm:"column:avatar_url"`
	}
	if err := r.db.Table("users").Select("id as user_id, full_name, avatar_url").
		Where("id = ?", project.OwnerID).Scan(&owner).Error; err != nil {
		return nil, err
	}

	var member struct {
		RoleID     *string `gorm:"column:role_id"`
		RoleName   *string `gorm:"column:role_name"`
		IsFavorite bool    `gorm:"column:is_favorite"`
	}
	if err := r.db.Table("project_members pm").
		Joins("LEFT JOIN roles r ON r.id = pm.role_id").
		Select("pm.role_id, r.name as role_name, pm.is_favorite").
		Where("pm.project_id = ? AND pm.user_id = ?", projectID, userID).
		Scan(&member).Error; err != nil {
		return nil, err
	}

	var permSlugs []string
	if member.RoleID != nil {
		if err := r.db.Table("role_permissions rp").
			Joins("JOIN permissions p ON p.id = rp.permission_id").
			Where("rp.role_id = ?", *member.RoleID).
			Pluck("p.slug", &permSlugs).Error; err != nil {
			return nil, err
		}
	}

	var memberCount int64
	if err := r.db.Table("project_members").Where("project_id = ?", projectID).Count(&memberCount).Error; err != nil {
		return nil, err
	}

	var taskTotal int64
	if err := r.db.Table("tasks").Where("project_id = ?", projectID).Count(&taskTotal).Error; err != nil {
		return nil, err
	}

	type colTaskCount struct {
		ColumnID    string `gorm:"column:column_id"`
		ColumnTitle string `gorm:"column:column_title"`
		TaskCount   int    `gorm:"column:task_count"`
	}
	var byColumn []colTaskCount
	if err := r.db.Table("columns c").
		Select("c.id as column_id, c.title as column_title, count(t.id) as task_count").
		Joins("LEFT JOIN tasks t ON t.column_id = c.id AND t.deleted_at IS NULL").
		Where("c.project_id = ?", projectID).
		Group("c.id, c.title").
		Order("c.position").
		Scan(&byColumn).Error; err != nil {
		return nil, err
	}

	myRole := dto.MyRolePermissions{}
	if member.RoleID != nil && member.RoleName != nil {
		myRole = dto.MyRolePermissions{
			ID:          *member.RoleID,
			Name:        *member.RoleName,
			Permissions: permSlugs,
		}
	}

	taskSummary := dto.ProjectTaskSummary{
		Total: int(taskTotal),
	}
	for _, c := range byColumn {
		taskSummary.ByColumn = append(taskSummary.ByColumn, dto.ColumnTaskSummary{
			ColumnID:  c.ColumnID,
			Title:     c.ColumnTitle,
			TaskCount: c.TaskCount,
		})
	}

	return &dto.ProjectDetailResponse{
		ID:          project.ID,
		Name:        project.Name,
		Key:         project.Key,
		Icon:        project.Icon,
		Background:  project.Background,
		IsArchived:  project.IsArchived,
		IsFavorite:  member.IsFavorite,
		Owner: dto.ProjectOwnerInfo{
			UserID:    owner.UserID,
			FullName:  owner.FullName,
			AvatarURL: owner.AvatarURL,
		},
		MyRole:      myRole,
		MemberCount: int(memberCount),
		TaskSummary: taskSummary,
		CreatedAt:   project.CreatedAt,
		UpdatedAt:   project.UpdatedAt,
	}, nil
}
