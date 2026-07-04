package implement

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
	_interface "TaskFlow-Go/internal/repository/interface"
)

type activityLogRepository struct{ db *gorm.DB }

func NewActivityLogRepository(db *gorm.DB) _interface.ActivityLogRepository {
	return &activityLogRepository{db: db}
}

func (r *activityLogRepository) Create(log *models.ActivityLog) error {
	return r.db.Create(log).Error
}

type activityLogRow struct {
	ID            string     `gorm:"column:id"`
	Action        string     `gorm:"column:action"`
	EntityType    string     `gorm:"column:entity_type"`
	EntityID      string     `gorm:"column:entity_id"`
	Description   *string    `gorm:"column:description"`
	UserID        *string    `gorm:"column:user_id"`
	UserFullName  string     `gorm:"column:user_full_name"`
	Username      string     `gorm:"column:username"`
	UserAvatar    *string    `gorm:"column:user_avatar"`
	WorkspaceID   *string    `gorm:"column:workspace_id"`
	WorkspaceName *string    `gorm:"column:workspace_name"`
	ProjectID     *string    `gorm:"column:project_id"`
	ProjectName   *string    `gorm:"column:project_name"`
	ProjectKey    *string    `gorm:"column:project_key"`
	MetadataStr   *string    `gorm:"column:metadata"`
	TaskRef       *string    `gorm:"column:task_ref"`
	TaskTitle     *string    `gorm:"column:task_title"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
}

func (r *activityLogRepository) buildBaseQuery(filters map[string][]string) *gorm.DB {
	q := r.db.Table("activity_logs al").
		Select(`
			al.id, al.action, al.entity_type, al.entity_id,
			al.metadata as description,
			u.id as user_id, u.full_name as user_full_name, u.username, u.avatar_url as user_avatar,
			w.id as workspace_id, w.name as workspace_name,
			p.id as project_id, p.name as project_name, p.key as project_key,
			al.metadata,
			CONCAT(p.key, '-', t.task_number) as task_ref, t.title as task_title,
			al.created_at
		`).
		Joins("LEFT JOIN users u ON u.id = al.user_id").
		Joins("LEFT JOIN workspaces w ON w.id = al.workspace_id").
		Joins("LEFT JOIN projects p ON p.id = al.project_id").
		Joins("LEFT JOIN tasks t ON t.id = al.entity_id AND al.entity_type = 'TASK'")

	if v, ok := filters["user_id"]; ok && len(v) > 0 {
		q = q.Where("al.user_id IN ?", v)
	}
	if v, ok := filters["entity_type"]; ok && len(v) > 0 {
		q = q.Where("al.entity_type IN ?", v)
	}
	if v, ok := filters["action"]; ok && len(v) > 0 {
		q = q.Where("al.action IN ?", v)
	}
	if v, ok := filters["date_from"]; ok && len(v) > 0 && v[0] != "" {
		q = q.Where("al.created_at >= ?", v[0])
	}
	if v, ok := filters["date_to"]; ok && len(v) > 0 && v[0] != "" {
		q = q.Where("al.created_at <= ?", v[0]+"T23:59:59Z")
	}
	return q
}

func (r *activityLogRepository) ListByWorkspace(workspaceID string, filters map[string][]string, limit int, cursor string) (*dto.ActivityLogListResponse, error) {
	q := r.buildBaseQuery(filters).Where("al.workspace_id = ?", workspaceID)

	if v, ok := filters["project_id"]; ok && len(v) > 0 {
		q = q.Where("al.project_id IN ?", v)
	}

	return r.fetchPaginated(q, limit, cursor)
}

func (r *activityLogRepository) ListByProject(projectID string, filters map[string][]string, limit int, cursor string) (*dto.ActivityLogListResponse, error) {
	q := r.buildBaseQuery(filters).Where("al.project_id = ?", projectID)
	return r.fetchPaginated(q, limit, cursor)
}

func (r *activityLogRepository) ListByTask(taskID string, limit int, cursor string, direction string) (*dto.ActivityLogListResponse, error) {
	q := r.db.Table("activity_logs al").
		Select(`
			al.id, al.action, al.entity_type, al.entity_id,
			al.metadata as description,
			u.id as user_id, u.full_name as user_full_name, u.username, u.avatar_url as user_avatar,
			p.id as project_id, p.name as project_name, p.key as project_key,
			al.metadata,
			CONCAT(p.key, '-', t.task_number) as task_ref, t.title as task_title,
			al.created_at
		`).
		Joins("LEFT JOIN users u ON u.id = al.user_id").
		Joins("LEFT JOIN projects p ON p.id = al.project_id").
		Joins("LEFT JOIN tasks t ON t.id = al.entity_id AND al.entity_type = 'TASK'").
		Where("al.entity_id = ?", taskID)

	return r.fetchPaginated(q, limit, cursor, direction)
}

func (r *activityLogRepository) fetchPaginated(q *gorm.DB, limit int, cursor string, opts ...string) (*dto.ActivityLogListResponse, error) {
	orderDir := "DESC"
	if len(opts) > 0 && opts[0] == "asc" {
		orderDir = "ASC"
	}

	if cursor != "" {
		b, err := base64.StdEncoding.DecodeString(cursor)
		if err == nil {
			parts := strings.SplitN(string(b), ":", 2)
			if len(parts) == 2 {
				t, err := time.Parse(time.RFC3339Nano, parts[0])
				if err == nil {
					if orderDir == "DESC" {
						q = q.Where("(al.created_at < ? OR (al.created_at = ? AND al.id < ?))", t, t, parts[1])
					} else {
						q = q.Where("(al.created_at > ? OR (al.created_at = ? AND al.id > ?))", t, t, parts[1])
					}
				}
			}
		}
	}

	q = q.Order(fmt.Sprintf("al.created_at %s, al.id %s", orderDir, orderDir))

	fetchLimit := limit + 1
	var rows []activityLogRow
	if err := q.Limit(fetchLimit).Scan(&rows).Error; err != nil {
		return nil, err
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	data := make([]dto.ActivityLogInfo, len(rows))
	for i, row := range rows {
		actor := dto.ActivityLogActor{
			UserID:    safeStr(row.UserID),
			FullName:  row.UserFullName,
			Username:  row.Username,
			AvatarURL: row.UserAvatar,
		}

		description := safeStr(row.Description)
		var metadata map[string]interface{}
		if row.MetadataStr != nil {
			json.Unmarshal([]byte(*row.MetadataStr), &metadata)
			if metadata == nil {
				metadata = make(map[string]interface{})
			}
			if desc, ok := metadata["description"]; ok {
				if s, ok2 := desc.(string); ok2 {
					description = s
				}
			}
		} else {
			metadata = make(map[string]interface{})
		}

		var wsRef *dto.ActivityLogWorkspaceRef
		if row.WorkspaceID != nil {
			wsRef = &dto.ActivityLogWorkspaceRef{
				ID:   *row.WorkspaceID,
				Name: safeStr(row.WorkspaceName),
			}
		}

		var projRef *dto.ActivityLogProjectRef
		if row.ProjectID != nil {
			projRef = &dto.ActivityLogProjectRef{
				ID:   *row.ProjectID,
				Name: safeStr(row.ProjectName),
				Key:  safeStr(row.ProjectKey),
			}
		}

		var snapshot *dto.EntitySnapshot
		if row.TaskRef != nil || row.TaskTitle != nil {
			snapshot = &dto.EntitySnapshot{
				TaskRef:   row.TaskRef,
				TaskTitle: row.TaskTitle,
			}
		}

		data[i] = dto.ActivityLogInfo{
			ID:            row.ID,
			Action:        row.Action,
			EntityType:    row.EntityType,
			EntityID:      row.EntityID,
			Description:   description,
			Actor:         actor,
			Workspace:     wsRef,
			Project:       projRef,
			EntitySnapshot: snapshot,
			Metadata:      metadata,
			CreatedAt:     row.CreatedAt.Format(time.RFC3339),
		}
	}

	var nextCursor *string
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		raw := fmt.Sprintf("%s:%s", last.CreatedAt.Format(time.RFC3339Nano), last.ID)
		enc := base64.StdEncoding.EncodeToString([]byte(raw))
		nextCursor = &enc
	}

	return &dto.ActivityLogListResponse{
		Data:       data,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	}, nil
}

func safeStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
