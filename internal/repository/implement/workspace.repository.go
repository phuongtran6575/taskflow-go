package implement

import (
	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
	"TaskFlow-Go/internal/projection"
	_interface "TaskFlow-Go/internal/repository/interface"
)

type workspaceRepository struct{ db *gorm.DB }

func (r *workspaceRepository) WithTx(tx *gorm.DB) _interface.WorkspaceRepository {
	return &workspaceRepository{db: tx}
}

func NewWorkspaceRepository(db *gorm.DB) _interface.WorkspaceRepository {
	return &workspaceRepository{db: db}
}
func (r *workspaceRepository) CountWorkspaceByPlan(
	plans []models.WorkspacePlan,
	role models.WorkspaceRole,
	userID string,
) ([]dto.WorkspacePlanCount, error) {
	var counts []dto.WorkspacePlanCount

	err := r.db.
		Table("workspaces w").
		Joins("JOIN workspace_members wm ON w.id = wm.workspace_id").
		Select("w.plan, COUNT(*) AS count").
		Where("w.plan IN ?", plans).
		Where("w.deleted_at IS NULL").
		Where("wm.user_id = ?", userID).
		Where("wm.role = ?", role).
		Group("w.plan").
		Scan(&counts).Error

	if err != nil {
		return nil, err
	}

	return counts, nil
}

func (r *workspaceRepository) GetWorkspaceByDomain(domain string) (*models.Workspace, error) {
	var workspace models.Workspace
	err := r.db.Where("LOWER(domain) = LOWER(?) AND deleted_at IS NULL", domain).First(&workspace).Error
	return &workspace, err
}

func (r *workspaceRepository) Create(workspace *models.Workspace) error {
	return r.db.Create(workspace).Error
}

func (r *workspaceRepository) GetByID(id string) (*models.Workspace, error) {
	var ws models.Workspace
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&ws).Error
	return &ws, err
}

func (r *workspaceRepository) Update(workspace *models.Workspace) error {
	return r.db.Save(workspace).Error
}

func (r *workspaceRepository) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.Workspace{}).Error
}

func (r *workspaceRepository) AddMember(member *models.WorkspaceMember) error {
	return r.db.Create(member).Error
}

func (r *workspaceRepository) GetMember(workspaceID, userID string) (*models.WorkspaceMember, error) {
	var member models.WorkspaceMember
	err := r.db.Where("workspace_id = ? AND user_id = ?", workspaceID, userID).First(&member).Error
	return &member, err
}

func (r *workspaceRepository) ListByUserIDWithSummary(userID string) ([]dto.WorkspaceSummary, int, error) {
	var summaries []dto.WorkspaceSummary

	err := r.db.Table("workspaces w").
		Joins("LEFT JOIN workspace_members wm ON w.id = wm.workspace_id AND wm.user_id = ?", userID).
		Where("w.owner_id = ? OR wm.user_id = ?", userID, userID).
		Where("w.deleted_at IS NULL").
		Select(`
            w.id                                                          AS id,
            w.name                                                        AS name,
            w.domain                                                      AS domain,
            w.plan                                                        AS plan,
            CASE WHEN w.owner_id = ? THEN 'OWNER' ELSE wm.role END       AS my_role,
            (SELECT COUNT(*) FROM workspace_members WHERE workspace_id = w.id) AS member_count,
            (SELECT COUNT(*) FROM projects WHERE workspace_id = w.id AND deleted_at IS NULL) AS project_count,
            COALESCE(wm.joined_at, w.created_at)                         AS joined_at
        `, userID).
		Scan(&summaries).Error

	if err != nil {
		return nil, 0, err
	}
	return summaries, len(summaries), nil
}

func (r *workspaceRepository) GetByIDWithDetail(workspaceID string) (*dto.WorkspaceDetailResponse, error) {
	var raw projection.WorkspaceDetailRow
	err := r.db.Table("workspaces w").
		Joins("JOIN users u ON u.id = w.owner_id AND u.deleted_at IS NULL").
		Where("w.id = ? AND w.deleted_at IS NULL", workspaceID).
		Select(`
            w.id                                                                        AS id,
            w.name                                                                      AS name,
            w.domain                                                                    AS domain,
            w.plan                                                                      AS plan,
            w.owner_id                                                                  AS owner_id,
            u.full_name                                                                 AS owner_full_name,
            u.avatar_url                                                                AS owner_avatar_url,
            (SELECT COUNT(*) FROM workspace_members WHERE workspace_id = w.id)          AS member_count,
            (SELECT COUNT(*) FROM projects WHERE workspace_id = w.id AND deleted_at IS NULL) AS project_count,
            w.created_at                                                                AS created_at,
            w.updated_at                                                                AS updated_at
        `).
		First(&raw).Error

	if err != nil {
		return nil, err
	}

	return &dto.WorkspaceDetailResponse{
		ID:     raw.ID,
		Name:   raw.Name,
		Domain: raw.Domain,
		Plan:   raw.Plan,
		Owner: dto.WorkspaceOwnerInfo{
			ID:        raw.OwnerID,
			FullName:  raw.OwnerFullName,
			AvatarURL: raw.OwnerAvatarURL,
		},
		MemberCount:  raw.MemberCount,
		ProjectCount: raw.ProjectCount,
		CreatedAt:    raw.CreatedAt,
		UpdatedAt:    raw.UpdatedAt,
	}, nil
}
