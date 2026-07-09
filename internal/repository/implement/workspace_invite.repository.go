package implement

import (
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
	_interface "TaskFlow-Go/internal/repository/interface"
)

type workspaceInviteRepository struct{ db *gorm.DB }

func NewWorkspaceInviteRepository(db *gorm.DB) _interface.WorkspaceInviteRepository {
	return &workspaceInviteRepository{db: db}
}

func (r *workspaceInviteRepository) WithTx(tx *gorm.DB) _interface.WorkspaceInviteRepository {
	return &workspaceInviteRepository{db: tx}
}

func (r *workspaceInviteRepository) Create(invite *models.WorkspaceInvite) error {
	return r.db.Create(invite).Error
}

func (r *workspaceInviteRepository) GetByID(id string) (*models.WorkspaceInvite, error) {
	var invite models.WorkspaceInvite
	err := r.db.Where("id = ?", id).First(&invite).Error
	return &invite, err
}

func (r *workspaceInviteRepository) GetByCode(code string) (*models.WorkspaceInvite, error) {
	var invite models.WorkspaceInvite
	err := r.db.Where("code = ?", code).First(&invite).Error
	return &invite, err
}

func (r *workspaceInviteRepository) CountByCode(code string) (int64, error) {
	var count int64
	err := r.db.Model(&models.WorkspaceInvite{}).Where("code = ?", code).Count(&count).Error
	return count, err
}

func (r *workspaceInviteRepository) IncrementUses(id string) error {
	return r.db.Model(&models.WorkspaceInvite{}).Where("id = ?", id).
		UpdateColumn("uses_count", gorm.Expr("uses_count + 1")).Error
}

func (r *workspaceInviteRepository) SoftDelete(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.WorkspaceInvite{}).Error
}

// BR-INV-02: Đếm link thực sự ACTIVE (chưa revoked, chưa expired, chưa exhausted)
func (r *workspaceInviteRepository) CountActiveByWorkspaceID(workspaceID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.WorkspaceInvite{}).
		Where("workspace_id = ? AND deleted_at IS NULL", workspaceID).
		Where("(expires_at IS NULL OR expires_at > NOW())").
		Where("(max_uses IS NULL OR uses_count < max_uses)").
		Count(&count).Error
	return count, err
}

type inviteListRow struct {
	ID              string     `gorm:"column:id"`
	Code            string     `gorm:"column:code"`
	URL             string     `gorm:"column:url"`
	Role            string     `gorm:"column:role"`
	MaxUses         *int       `gorm:"column:max_uses"`
	UsesCount       int        `gorm:"column:uses_count"`
	ExpiresAt       *time.Time `gorm:"column:expires_at"`
	Status          string     `gorm:"column:status"`
	CreatedByUserID string     `gorm:"column:created_by_user_id"`
	CreatedByName   string     `gorm:"column:created_by_name"`
	CreatedByAvatar *string    `gorm:"column:created_by_avatar"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	DeletedAt       *time.Time `gorm:"column:deleted_at"`
}

func (r *workspaceInviteRepository) ListWithPagination(workspaceID string, status string, page int, limit int) ([]dto.InviteInfo, *dto.Pagination, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	baseQuery := r.db.Table("workspace_invites wi").
		Joins("LEFT JOIN users u ON wi.created_by = u.id").
		Where("wi.workspace_id = ?", workspaceID)

	switch status {
	case "ACTIVE":
		baseQuery = baseQuery.Where("wi.deleted_at IS NULL AND (wi.expires_at IS NULL OR wi.expires_at > NOW()) AND (wi.max_uses IS NULL OR wi.uses_count < wi.max_uses)")
	case "EXPIRED":
		baseQuery = baseQuery.Where("wi.deleted_at IS NULL AND wi.expires_at IS NOT NULL AND wi.expires_at <= NOW()")
	case "EXHAUSTED":
		baseQuery = baseQuery.Where("wi.deleted_at IS NULL AND wi.max_uses IS NOT NULL AND wi.uses_count >= wi.max_uses")
	case "REVOKED":
		baseQuery = baseQuery.Where("wi.deleted_at IS NOT NULL")
	}

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, nil, err
	}

	offset := (page - 1) * limit
	var rows []inviteListRow
	if err := baseQuery.
		Select(`
			wi.id, wi.code,
			CONCAT('https://app.example.com/invite/', wi.code) AS url,
			wi.role, wi.max_uses, wi.uses_count, wi.expires_at,
			CASE
				WHEN wi.deleted_at IS NOT NULL THEN 'REVOKED'
				WHEN wi.expires_at IS NOT NULL AND wi.expires_at <= NOW() THEN 'EXPIRED'
				WHEN wi.max_uses IS NOT NULL AND wi.uses_count >= wi.max_uses THEN 'EXHAUSTED'
				ELSE 'ACTIVE'
			END AS status,
			COALESCE(wi.created_by, '') AS created_by_user_id,
			COALESCE(u.full_name, '') AS created_by_name,
			u.avatar_url AS created_by_avatar,
			wi.created_at, wi.deleted_at
		`).
		Offset(offset).Limit(limit).
		Order("wi.created_at DESC").
		Scan(&rows).Error; err != nil {
		return nil, nil, err
	}

	invites := make([]dto.InviteInfo, len(rows))
	for i, row := range rows {
		invites[i] = dto.InviteInfo{
			ID:        row.ID,
			Code:      row.Code,
			URL:       row.URL,
			Role:      row.Role,
			MaxUses:   row.MaxUses,
			UsesCount: row.UsesCount,
			ExpiresAt: row.ExpiresAt,
			Status:    row.Status,
			CreatedBy: dto.InviteCreatorInfo{
				UserID:    row.CreatedByUserID,
				FullName:  row.CreatedByName,
				AvatarURL: row.CreatedByAvatar,
			},
			CreatedAt: row.CreatedAt,
			DeletedAt: row.DeletedAt,
		}
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}
	return invites, &dto.Pagination{
		Total:      int(total),
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

func (r *workspaceInviteRepository) GetByCodeWithPreview(code string) (*dto.InvitePreviewResponse, error) {
	var result struct {
		WorkspaceID   string     `gorm:"column:workspace_id"`
		WorkspaceName string     `gorm:"column:workspace_name"`
		MemberCount   int        `gorm:"column:member_count"`
		Code          string     `gorm:"column:code"`
		Role          string     `gorm:"column:role"`
		ExpiresAt     *time.Time `gorm:"column:expires_at"`
		MaxUses       *int       `gorm:"column:max_uses"`
		UsesCount     int        `gorm:"column:uses_count"`
		DeletedAt     *time.Time `gorm:"column:deleted_at"`
	}

	err := r.db.Table("workspace_invites wi").
		Joins("JOIN workspaces w ON w.id = wi.workspace_id AND w.deleted_at IS NULL").
		Where("wi.code = ?", code).
		Unscoped().
		Select(`
			wi.workspace_id,
			w.name AS workspace_name,
			(SELECT COUNT(*) FROM workspace_members WHERE workspace_id = w.id) AS member_count,
			wi.code, wi.role, wi.expires_at, wi.max_uses, wi.uses_count, wi.deleted_at
		`).
		First(&result).Error
	if err != nil {
		return nil, err
	}

	status := "ACTIVE"
	if result.DeletedAt != nil {
		status = "REVOKED"
	} else if result.ExpiresAt != nil && result.ExpiresAt.Before(time.Now()) {
		status = "EXPIRED"
	} else if result.MaxUses != nil && result.UsesCount >= *result.MaxUses {
		status = "EXHAUSTED"
	}

	var remainingUses *int
	if result.MaxUses != nil {
		rem := *result.MaxUses - result.UsesCount
		if rem < 0 {
			rem = 0
		}
		remainingUses = &rem
	}

	return &dto.InvitePreviewResponse{
		Workspace: dto.InviteWorkspacePreview{
			ID:          result.WorkspaceID,
			Name:        result.WorkspaceName,
			MemberCount: result.MemberCount,
		},
		Invite: dto.InvitePreviewInfo{
			Code:          result.Code,
			Role:          result.Role,
			ExpiresAt:     result.ExpiresAt,
			RemainingUses: remainingUses,
		},
		Status: status,
	}, nil
}
