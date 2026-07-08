package implement

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
	"TaskFlow-Go/internal/projection"
	_interface "TaskFlow-Go/internal/repository/interface"
)

type taskAssigneeRepository struct{ db *gorm.DB }

func NewTaskAssigneeRepository(db *gorm.DB) _interface.TaskAssigneeRepository {
	return &taskAssigneeRepository{db: db}
}

func (r *taskAssigneeRepository) WithTx(tx *gorm.DB) _interface.TaskAssigneeRepository {
	return &taskAssigneeRepository{db: tx}
}

func (r *taskAssigneeRepository) ListByTaskIDs(taskIDs []string) (map[string][]dto.AssigneeInfo, error) {
	if len(taskIDs) == 0 {
		return map[string][]dto.AssigneeInfo{}, nil
	}
	var rows []struct {
		TaskID string
		dto.AssigneeInfo
	}
	err := r.db.Table("task_assignees ta").
		Select("ta.task_id, ta.user_id, u.avatar_url, u.full_name, ta.assigned_at").
		Joins("JOIN users u ON ta.user_id = u.id").
		Where("ta.task_id IN ?", taskIDs).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[string][]dto.AssigneeInfo, len(taskIDs))
	for _, row := range rows {
		result[row.TaskID] = append(result[row.TaskID], row.AssigneeInfo)
	}
	return result, nil
}

func (r *taskAssigneeRepository) ListByTaskID(taskID string) ([]dto.AssigneeInfo, error) {
	var assignees []dto.AssigneeInfo
	err := r.db.Table("task_assignees ta").
		Select("ta.user_id, u.avatar_url, u.full_name, ta.assigned_at").
		Joins("JOIN users u ON ta.user_id = u.id").
		Where("ta.task_id = ?", taskID).Scan(&assignees).Error
	return assignees, err
}

func (r *taskAssigneeRepository) BulkTaskAssignee(taskID string, assigneeIDs []string, creatorID string) error {
	var taskAssignees []models.TaskAssignee
	for _, assigneeID := range assigneeIDs {
		if _, err := r.GetByID(taskID, assigneeID); err == nil {
			continue
		}
		taskAssignees = append(taskAssignees, models.TaskAssignee{
			TaskID:       taskID,
			UserID:       assigneeID,
			AssignedByID: &creatorID,
		})
	}
	if len(taskAssignees) == 0 {
		return nil
	}
	if err := r.db.Create(&taskAssignees).Error; err != nil {
		return errors.New("failed to create task assignees")
	}
	return nil
}

func (r *taskAssigneeRepository) Create(assignee *models.TaskAssignee) error {
	return r.db.Create(assignee).Error
}

func (r *taskAssigneeRepository) GetByID(taskID, userID string) (*models.TaskAssignee, error) {
	var a models.TaskAssignee
	err := r.db.Where("task_id = ? AND user_id = ?", taskID, userID).First(&a).Error
	return &a, err
}

func (r *taskAssigneeRepository) ListTaskAssigneesByTaskID(taskID string) ([]models.TaskAssignee, error) {
	var assignees []models.TaskAssignee
	err := r.db.Where("task_id = ?", taskID).Find(&assignees).Error
	return assignees, err
}

func (r *taskAssigneeRepository) Delete(taskID, userID string) error {
	return r.db.Where("task_id = ? AND user_id = ?", taskID, userID).
		Delete(&models.TaskAssignee{}).Error
}

func (r *taskAssigneeRepository) ListByTaskIDWithDetail(taskID string) ([]dto.AssigneeDetail, error) {
	type detailRow struct {
		UserID         string     `gorm:"column:user_id"`
		FullName       string     `gorm:"column:full_name"`
		Username       string     `gorm:"column:username"`
		AvatarURL      *string    `gorm:"column:avatar_url"`
		RoleID         *string    `gorm:"column:role_id"`
		RoleName       *string    `gorm:"column:role_name"`
		AssignedAt     time.Time  `gorm:"column:assigned_at"`
		AssignedByID   *string    `gorm:"column:assigned_by_id"`
		AssignedByName *string    `gorm:"column:assigned_by_name"`
	}
	var rows []detailRow
	err := r.db.Table("task_assignees ta").
		Select(`
			ta.user_id, u.full_name, u.username, u.avatar_url,
			r.id as role_id, r.name as role_name,
			ta.assigned_at,
			ta.assigned_by_id, ab.full_name as assigned_by_name
		`).
		Joins("JOIN users u ON u.id = ta.user_id").
		Joins("JOIN tasks t ON t.id = ta.task_id").
		Joins("LEFT JOIN project_members pm ON pm.project_id = t.project_id AND pm.user_id = ta.user_id").
		Joins("LEFT JOIN roles r ON r.id = pm.role_id").
		Joins("LEFT JOIN users ab ON ab.id = ta.assigned_by_id").
		Where("ta.task_id = ?", taskID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]dto.AssigneeDetail, len(rows))
	for i, row := range rows {
		var roleRef *dto.RoleRef
		if row.RoleID != nil {
			name := ""
			if row.RoleName != nil {
				name = *row.RoleName
			}
			roleRef = &dto.RoleRef{ID: *row.RoleID, Name: name}
		}
		result[i] = dto.AssigneeDetail{
			UserID:      row.UserID,
			FullName:    row.FullName,
			Username:    row.Username,
			AvatarURL:   row.AvatarURL,
			ProjectRole: roleRef,
			AssignedAt:  row.AssignedAt.Format(time.RFC3339),
		}
		if row.AssignedByID != nil {
			name := ""
			if row.AssignedByName != nil {
				name = *row.AssignedByName
			}
			result[i].AssignedBy = &struct {
				UserID   string `json:"user_id"`
				FullName string `json:"full_name"`
			}{
				UserID:   *row.AssignedByID,
				FullName: name,
			}
		}
	}
	return result, nil
}

func (r *taskAssigneeRepository) ListAvailableForTask(taskID string, projectID string, search string, page int, limit int) ([]dto.AvailableAssigneeInfo, *dto.Pagination, error) {
	baseQuery := r.db.Table("project_members pm").
		Joins("JOIN users u ON u.id = pm.user_id").
		Joins("JOIN roles r ON r.id = pm.role_id").
		Where("pm.project_id = ?", projectID).
		Where("pm.user_id NOT IN (SELECT ta.user_id FROM task_assignees ta WHERE ta.task_id = ?)", taskID)

	if search != "" {
		pattern := "%" + search + "%"
		baseQuery = baseQuery.Where("(u.full_name ILIKE ? OR u.username ILIKE ?)", pattern, pattern)
	}

	var total int64
	if err := baseQuery.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, nil, err
	}

	offset := (page - 1) * limit
	type availableRow struct {
		UserID    string  `gorm:"column:user_id"`
		FullName  string  `gorm:"column:full_name"`
		Username  string  `gorm:"column:username"`
		AvatarURL *string `gorm:"column:avatar_url"`
		RoleID    string  `gorm:"column:role_id"`
		RoleName  string  `gorm:"column:role_name"`
	}
	var rows []availableRow
	if err := baseQuery.Select(`
		pm.user_id, u.full_name, u.username, u.avatar_url,
		r.id as role_id, r.name as role_name
	`).Offset(offset).Limit(limit).Scan(&rows).Error; err != nil {
		return nil, nil, err
	}

	result := make([]dto.AvailableAssigneeInfo, len(rows))
	for i, row := range rows {
		result[i] = dto.AvailableAssigneeInfo{
			UserID:    row.UserID,
			FullName:  row.FullName,
			Username:  row.Username,
			AvatarURL: row.AvatarURL,
			ProjectRole: &dto.RoleRef{
				ID:   row.RoleID,
				Name: row.RoleName,
			},
		}
	}

	return result, dto.NewPagination(total, dto.PaginationParam{Page: page, Limit: limit}), nil
}

func (r *taskAssigneeRepository) ListMyTasks(userID string, workspaceID string, filters map[string]interface{}, page int, limit int, sortBy string, sortDir string) ([]dto.MyTaskInfo, *dto.MyTaskSummary, *dto.Pagination, error) {
	baseJoins := "JOIN tasks t ON t.id = ta.task_id AND t.deleted_at IS NULL JOIN projects p ON p.id = t.project_id"

	buildQuery := func(tx *gorm.DB) *gorm.DB {
		q := tx.Where("ta.user_id = ? AND t.deleted_at IS NULL AND p.workspace_id = ?", userID, workspaceID).
			Where("EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id = p.id AND pm.user_id = ?)", userID)

		if v, ok := filters["project_id"]; ok && v != "" {
			q = q.Where("t.project_id = ?", v)
		}
		if v, ok := filters["priority"]; ok && v != "" {
			q = q.Where("t.priority = ?", v)
		}
		if v, ok := filters["due_date_from"]; ok && v != "" {
			q = q.Where("t.due_date >= ?", v)
		}
		if v, ok := filters["due_date_to"]; ok && v != "" {
			q = q.Where("t.due_date <= ?", v)
		}
		if v, ok := filters["overdue"]; ok && v == true {
			q = q.Where("t.due_date < NOW() AND t.due_date IS NOT NULL")
		}
		if v, ok := filters["search"]; ok && v != "" {
			search := "%" + v.(string) + "%"
			q = q.Where("t.title ILIKE ?", search)
		}
		if _, ok := filters["exclude_archived"]; ok {
			q = q.Where("p.is_archived = false")
		}
		return q
	}

	baseQuery := r.db.Table("task_assignees ta").Joins(baseJoins)

	var total int64
	if err := buildQuery(baseQuery.Session(&gorm.Session{})).
		Count(&total).Error; err != nil {
		return nil, nil, nil, err
	}

	// BR-ASSIGN-07: Summary block — tính trên toàn bộ my tasks (không bị filter ảnh hưởng)
	summaryQuery := r.db.Table("task_assignees ta").
		Select(`
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE t.due_date < NOW() AND c.is_done = false) AS overdue,
			COUNT(*) FILTER (WHERE t.due_date::date = CURRENT_DATE) AS due_today,
			COUNT(*) FILTER (WHERE t.due_date BETWEEN NOW() AND NOW() + INTERVAL '7 days') AS due_this_week,
			COUNT(*) FILTER (WHERE t.due_date IS NULL) AS no_due_date
		`).
		Joins("JOIN tasks t ON t.id = ta.task_id AND t.deleted_at IS NULL").
		Joins("JOIN projects p ON p.id = t.project_id").
		Joins("JOIN columns c ON c.id = t.column_id").
		Where("ta.user_id = ? AND p.workspace_id = ?", userID, workspaceID).
		Where("EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id = p.id AND pm.user_id = ?)", userID)

	var summary dto.MyTaskSummary
	if err := summaryQuery.Scan(&summary).Error; err != nil {
		return nil, nil, nil, err
	}

	// BR-ASSIGN-07: Sort mặc định theo 5 nhóm
	offset := (page - 1) * limit
	var orderClause string
	if sortBy == "br_default" || sortBy == "" {
		orderClause = `
			CASE
				WHEN t.due_date < NOW()                                           THEN 0
				WHEN t.due_date::date = CURRENT_DATE                              THEN 1
				WHEN t.due_date < NOW() + INTERVAL '7 days'                       THEN 2
				WHEN t.due_date IS NOT NULL                                       THEN 3
				ELSE 4
			END ASC,
			CASE
				WHEN t.due_date::date = CURRENT_DATE THEN
					CASE t.priority
						WHEN 'URGENT' THEN 0
						WHEN 'HIGH'   THEN 1
						WHEN 'MED'    THEN 2
						WHEN 'LOW'    THEN 3
					END
				ELSE 0
			END ASC,
			CASE WHEN t.due_date IS NOT NULL THEN t.due_date END ASC NULLS LAST,
			CASE WHEN t.due_date IS NULL THEN t.created_at END DESC`
	} else if sortBy != "" && sortDir != "" {
		orderClause = "t." + sortBy + " " + sortDir
	} else {
		orderClause = "t.created_at DESC"
	}

	var rows []projection.MyTaskRow
	if err := buildQuery(r.db.Table("task_assignees ta").
		Select(`
			t.id, t.task_number, CONCAT(p.key, '-', t.task_number) as task_ref,
			t.title, t.priority, t.due_date,
			c.id as column_id, c.title as column_title,
			p.id as project_id, p.name as project_name, p.key as project_key, p.icon as project_icon,
			parent.id as parent_id,
			CONCAT(p.key, '-', parent.task_number) as parent_ref,
			parent.title as parent_title`).
		Joins("JOIN tasks t ON t.id = ta.task_id AND t.deleted_at IS NULL").
		Joins("JOIN projects p ON p.id = t.project_id").
		Joins("JOIN columns c ON c.id = t.column_id").
		Joins("LEFT JOIN tasks parent ON parent.id = t.parent_id AND parent.deleted_at IS NULL")).
		Offset(offset).Limit(limit).Order(orderClause).
		Scan(&rows).Error; err != nil {
		return nil, nil, nil, err
	}

	tasks := make([]dto.MyTaskInfo, len(rows))
	for i, row := range rows {
		t := dto.MyTaskInfo{
			ID:       row.ID,
			TaskRef:  row.TaskRef,
			Title:    row.Title,
			Priority: row.Priority,
			DueDate:  row.DueDate,
			Column:   dto.ColumnRef{ID: row.ColumnID, Title: row.ColumnTitle},
			Project: dto.MyTaskProjectRef{
				ID:   row.ProjectID,
				Name: row.ProjectName,
				Key:  row.ProjectKey,
				Icon: row.ProjectIcon,
			},
		}
		if row.ParentID != nil {
			t.Parent = &struct {
				TaskRef string `json:"task_ref"`
				Title   string `json:"title"`
			}{
				TaskRef: *row.ParentRef,
				Title:   *row.ParentTitle,
			}
		}
		tasks[i] = t
	}

	return tasks, &summary, dto.NewPagination(total, dto.PaginationParam{Page: page, Limit: limit}), nil
}
