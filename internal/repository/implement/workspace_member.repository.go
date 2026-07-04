package implement

import (
	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
	"TaskFlow-Go/internal/projection"
	_interface "TaskFlow-Go/internal/repository/interface"
)

type workspaceMemberRepository struct{ db *gorm.DB }

func NewWorkspaceMemberRepository(db *gorm.DB) _interface.WorkspaceMemberRepository {
	return &workspaceMemberRepository{db: db}
}

func (r *workspaceMemberRepository) WithTx(tx *gorm.DB) _interface.WorkspaceMemberRepository {
	return &workspaceMemberRepository{db: tx}
}

func (r *workspaceMemberRepository) GetMemberWithInfor(workspaceID string, userID string) (*projection.MemberWithInfoRow, error) {
	var result projection.MemberWithInfoRow
	err := r.db.Table("workspace_members wm").
		Joins("JOIN users u ON wm.user_id = u.id").
		Where("workspace_id = ? AND user_id = ?", workspaceID, userID).
		Select("wm.user_id, u.full_name, wm.role").
		Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *workspaceMemberRepository) Create(member *models.WorkspaceMember) error {
	return r.db.Create(member).Error
}

func (r *workspaceMemberRepository) GetByID(workspaceID, userID string) (*models.WorkspaceMember, error) {
	var member models.WorkspaceMember
	err := r.db.Where("workspace_id = ? AND user_id = ?", workspaceID, userID).First(&member).Error
	return &member, err
}

func (r *workspaceMemberRepository) ListByUserID(userID string) ([]models.WorkspaceMember, error) {
	var members []models.WorkspaceMember
	err := r.db.Where("user_id = ?", userID).Find(&members).Error
	return members, err
}

func (r *workspaceMemberRepository) UpdateRole(workspaceID, userID string, role models.WorkspaceRole) error {
	return r.db.Model(&models.WorkspaceMember{}).
		Where("workspace_id = ? AND user_id = ?", workspaceID, userID).
		Update("role", role).Error
}

func (r *workspaceMemberRepository) Delete(workspaceID, userID string) error {
	return r.db.Where("workspace_id = ? AND user_id = ?", workspaceID, userID).
		Delete(&models.WorkspaceMember{}).Error
}

func (r *workspaceMemberRepository) ListWithPagination(workspaceID string, search string, role string, page int, limit int) ([]dto.MemberInfo, *dto.Pagination, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	query := r.db.Table("workspace_members wm").
		Joins("JOIN users u ON wm.user_id = u.id").
		Where("wm.workspace_id = ?", workspaceID)

	if search != "" {
		like := "%" + search + "%"
		query = query.Where("(u.full_name ILIKE ? OR u.email ILIKE ?)", like, like)
	}
	if role != "" {
		query = query.Where("wm.role = ?", role)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, nil, err
	}

	offset := (page - 1) * limit
	var members []dto.MemberInfo

	q := query.Select("wm.user_id, u.full_name, u.username, u.email, u.avatar_url, wm.role, wm.joined_at").
		Offset(offset).Limit(limit)

	switch {
	case search != "" && role == "":
		like := "%" + search + "%"
		q = q.Order(gorm.Expr("CASE WHEN u.full_name ILIKE ? THEN 0 ELSE 1 END ASC, wm.joined_at ASC", like))
	case role != "":
		q = q.Order("wm.joined_at ASC")
	default:
		q = q.Order("CASE wm.role WHEN 'OWNER' THEN 0 WHEN 'ADMIN' THEN 1 WHEN 'MEMBER' THEN 2 END ASC, wm.joined_at ASC")
	}

	if err := q.Scan(&members).Error; err != nil {
		return nil, nil, err
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}
	return members, &dto.Pagination{
		Total:      int(total),
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

func (r *workspaceMemberRepository) CountMembers(workspaceID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.WorkspaceMember{}).Where("workspace_id = ?", workspaceID).Count(&count).Error
	return count, err
}

func (r *workspaceMemberRepository) GetUserProjectIDsInWorkspace(workspaceID, userID string) ([]string, error) {
	var projectIDs []string
	err := r.db.Table("project_members pm").
		Joins("JOIN projects p ON pm.project_id = p.id AND p.deleted_at IS NULL").
		Where("pm.user_id = ? AND p.workspace_id = ?", userID, workspaceID).
		Pluck("p.id", &projectIDs).Error
	return projectIDs, err
}

func (r *workspaceMemberRepository) GetByIDWithDetails(workspaceID string, userID string) (*dto.MemberDetailResponse, error) {
	// Bước 1: Lấy thông tin member + user
	var result dto.MemberDetailResponse
	err := r.db.Table("workspace_members wm").
		Joins("JOIN users u ON wm.user_id = u.id AND u.deleted_at IS NULL").
		Where("wm.workspace_id = ? AND wm.user_id = ?", workspaceID, userID).
		Select("wm.user_id, u.full_name, u.username, u.email, u.avatar_url, wm.role, wm.joined_at").
		First(&result).Error
	if err != nil {
		return nil, err // Tự động trả ErrRecordNotFound nếu không có
	}

	// Bước 2: Lấy danh sách project mà member này tham gia trong workspace
	var projects []dto.ProjectRoleSummary
	err = r.db.Table("project_members pm").
		Joins("JOIN projects p ON pm.project_id = p.id AND p.deleted_at IS NULL").
		Joins("LEFT JOIN roles r ON pm.role_id = r.id").
		Where("pm.user_id = ? AND p.workspace_id = ?", userID, workspaceID).
		Select("p.id AS project_id, p.name AS name, p.key AS key, COALESCE(r.name, 'MEMBER') AS role_name").
		Scan(&projects).Error
	if err != nil {
		return nil, err
	}

	result.Projects = projects
	return &result, nil
}
