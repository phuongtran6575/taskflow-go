package implement

import (
	"encoding/base64"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/markdown"
	"TaskFlow-Go/internal/models"
	_interface "TaskFlow-Go/internal/repository/interface"
)

type commentRepository struct{ db *gorm.DB }

func NewCommentRepository(db *gorm.DB) _interface.CommentRepository {
	return &commentRepository{db: db}
}

func (r *commentRepository) Create(comment *models.Comment) error {
	return r.db.Create(comment).Error
}

func (r *commentRepository) GetByID(id string) (*models.Comment, error) {
	var c models.Comment
	err := r.db.Where("id = ?", id).First(&c).Error
	return &c, err
}

func (r *commentRepository) Update(comment *models.Comment) error {
	return r.db.Save(comment).Error
}

func (r *commentRepository) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.Comment{}).Error
}

func (r *commentRepository) ListByTaskIDWithCursor(taskID string, limit int, cursor string, direction string) (*dto.CommentListResponse, error) {
	type commentRow struct {
		ID          string          `gorm:"column:id"`
		Content     string          `gorm:"column:content"`
		CreatedAt   time.Time       `gorm:"column:created_at"`
		UpdatedAt   time.Time       `gorm:"column:updated_at"`
		DeletedAt   gorm.DeletedAt  `gorm:"column:deleted_at"`
		AuthorID    *string         `gorm:"column:author_id"`
		AuthorName  string          `gorm:"column:author_name"`
		AuthorUN    string          `gorm:"column:author_username"`
		AuthorAV    *string         `gorm:"column:author_avatar"`
	}

	var total int64
	if err := r.db.Model(&models.Comment{}).Where("task_id = ?", taskID).Count(&total).Error; err != nil {
		return nil, err
	}

	query := r.db.Table("comments c").
		Select(`
			c.id, c.content, c.created_at, c.updated_at, c.deleted_at,
			u.id as author_id, u.full_name as author_name,
			u.username as author_username, u.avatar_url as author_avatar
		`).
		Joins("LEFT JOIN users u ON u.id = c.user_id").
		Where("c.task_id = ?", taskID)

	if cursor != "" {
		b, err := base64.StdEncoding.DecodeString(cursor)
		if err == nil {
			t, err := time.Parse(time.RFC3339Nano, string(b))
			if err == nil {
				if direction == "desc" {
					query = query.Where("c.created_at < ?", t)
				} else {
					query = query.Where("c.created_at > ?", t)
				}
			}
		}
	}

	if direction == "desc" {
		query = query.Order("c.created_at DESC")
	} else {
		query = query.Order("c.created_at ASC")
	}

	fetchLimit := limit + 1
	var rows []commentRow
	if err := query.Limit(fetchLimit).Scan(&rows).Error; err != nil {
		return nil, err
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	type taskInfo struct {
		TaskID  string `gorm:"column:task_id"`
		TaskRef string `gorm:"column:task_ref"`
	}
	var ti taskInfo
	if err := r.db.Table("tasks t").
		Select("t.id as task_id, CONCAT(p.key, '-', t.task_number) as task_ref").
		Joins("JOIN projects p ON p.id = t.project_id").
		Where("t.id = ?", taskID).
		First(&ti).Error; err != nil {
		return nil, err
	}

	commentIDs := make([]string, len(rows))
	for i, row := range rows {
		commentIDs[i] = row.ID
	}

	type mentionRow struct {
		CommentID string `gorm:"column:comment_id"`
		UserID    string `gorm:"column:user_id"`
		Username  string `gorm:"column:username"`
		FullName  string `gorm:"column:full_name"`
	}
	var mrows []mentionRow
	if len(commentIDs) > 0 {
		r.db.Table("comment_mentions cm").
			Select("cm.comment_id, cm.user_id, u.username, u.full_name").
			Joins("JOIN users u ON u.id = cm.user_id").
			Where("cm.comment_id IN ?", commentIDs).
			Scan(&mrows)
	}

	mentionMap := make(map[string][]dto.MentionUser)
	for _, m := range mrows {
		mentionMap[m.CommentID] = append(mentionMap[m.CommentID], dto.MentionUser{
			UserID: m.UserID, Username: m.Username, FullName: m.FullName,
		})
	}

	data := make([]dto.CommentInfo, len(rows))
	var nextCursor *string
	for i, row := range rows {
		isDeleted := row.DeletedAt.Valid
		isEdited := !row.CreatedAt.Equal(row.UpdatedAt)

		var content *string
		var contentHTML *string
		var author *dto.CommentAuthor
		if !isDeleted {
			c := row.Content
			content = &c
			mm := make([]markdown.MentionUser, len(mentionMap[row.ID]))
			for j, m := range mentionMap[row.ID] {
				mm[j] = markdown.MentionUser{UserID: m.UserID, Username: m.Username}
			}
			html := markdown.RenderToHTML(row.Content, mm)
			contentHTML = &html
			if row.AuthorID != nil {
				author = &dto.CommentAuthor{
					UserID:   *row.AuthorID,
					FullName: row.AuthorName,
					Username: row.AuthorUN,
					AvatarURL: row.AuthorAV,
				}
			}
		}

		data[i] = dto.CommentInfo{
			ID:          row.ID,
			Content:     content,
			ContentHTML: contentHTML,
			IsDeleted:   isDeleted,
			Author:      author,
			Mentions:    mentionMap[row.ID],
			IsEdited:    isEdited,
			CreatedAt:   row.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   row.UpdatedAt.Format(time.RFC3339),
		}
	}

	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		encoded := base64.StdEncoding.EncodeToString([]byte(last.CreatedAt.Format(time.RFC3339Nano)))
		nextCursor = &encoded
	}

	return &dto.CommentListResponse{
		TaskID:     ti.TaskID,
		TaskRef:    ti.TaskRef,
		Total:      int(total),
		Data:       data,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	}, nil
}

func (r *commentRepository) ListMentionableUsers(projectID string, search string, limit int) (*dto.MentionableUsersResponse, error) {
	type userRow struct {
		UserID    string  `gorm:"column:user_id"`
		Username  string  `gorm:"column:username"`
		FullName  string  `gorm:"column:full_name"`
		AvatarURL *string `gorm:"column:avatar_url"`
		RoleID    string  `gorm:"column:role_id"`
		RoleName  string  `gorm:"column:role_name"`
	}

	query := r.db.Table("project_members pm").
		Select("pm.user_id, u.username, u.full_name, u.avatar_url, r.id as role_id, r.name as role_name").
		Joins("JOIN users u ON u.id = pm.user_id").
		Joins("JOIN roles r ON r.id = pm.role_id").
		Where("pm.project_id = ?", projectID)

	if search != "" {
		p := "%" + search + "%"
		query = query.Where("(u.username ILIKE ? OR u.full_name ILIKE ?)", p, p)
	}

	var rows []userRow
	if err := query.Limit(limit).Scan(&rows).Error; err != nil {
		return nil, err
	}

	data := make([]dto.MentionableUserInfo, len(rows))
	for i, row := range rows {
		data[i] = dto.MentionableUserInfo{
			UserID:    row.UserID,
			Username:  row.Username,
			FullName:  row.FullName,
			AvatarURL: row.AvatarURL,
			ProjectRole: &dto.RoleRef{
				ID:   row.RoleID,
				Name: row.RoleName,
			},
		}
	}

	return &dto.MentionableUsersResponse{Data: data, Total: len(data)}, nil
}

func (r *commentRepository) CreateMention(mention *models.CommentMention) error {
	return r.db.Create(mention).Error
}

func (r *commentRepository) DeleteMentionsByCommentID(commentID string) error {
	return r.db.Where("comment_id = ?", commentID).Delete(&models.CommentMention{}).Error
}

func (r *commentRepository) GetMentionsByCommentID(commentID string) ([]models.CommentMention, error) {
	var ms []models.CommentMention
	err := r.db.Where("comment_id = ?", commentID).Find(&ms).Error
	return ms, err
}

func (r *commentRepository) ListPreviousCommenters(taskID string, excludeUserID string) ([]string, error) {
	var userIDs []string
	query := r.db.Table("comments").
		Select("DISTINCT user_id").
		Where("task_id = ? AND deleted_at IS NULL", taskID)
	if excludeUserID != "" {
		query = query.Where("user_id != ?", excludeUserID)
	}
	if err := query.Pluck("user_id", &userIDs).Error; err != nil {
		return nil, err
	}
	return userIDs, nil
}

func (r *commentRepository) ResolveUsernames(projectID string, workspaceID string, usernames []string) (map[string]dto.MentionUser, error) {
	if len(usernames) == 0 {
		return map[string]dto.MentionUser{}, nil
	}
	type row struct {
		UserID   string `gorm:"column:user_id"`
		Username string `gorm:"column:username"`
		FullName string `gorm:"column:full_name"`
	}
	var rows []row
	err := r.db.Table("users u").
		Select("u.id as user_id, u.username, u.full_name").
		Where("u.username IN ? AND u.deleted_at IS NULL AND u.is_active = true", usernames).
		Where(`
			EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id = ? AND pm.user_id = u.id)
			OR u.id = (SELECT w.owner_id FROM workspaces w WHERE w.id = ?)
		`, projectID, workspaceID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[string]dto.MentionUser, len(rows))
	for _, r := range rows {
		result[r.Username] = dto.MentionUser{
			UserID: r.UserID, Username: r.Username, FullName: r.FullName,
		}
	}
	return result, nil
}


