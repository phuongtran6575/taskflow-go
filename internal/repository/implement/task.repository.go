package implement

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
	"TaskFlow-Go/internal/projection"
	_interface "TaskFlow-Go/internal/repository/interface"
)

type taskRepository struct{ db *gorm.DB }

func NewTaskRepository(db *gorm.DB) _interface.TaskRepository {
	return &taskRepository{db: db}
}

func (r *taskRepository) WithTx(tx *gorm.DB) _interface.TaskRepository {
	return &taskRepository{db: tx}
}

func (r *taskRepository) Create(task *models.Task) error {
	return r.db.Create(task).Error
}

func (r *taskRepository) GetByID(id string) (*models.Task, error) {
	var task models.Task
	err := r.db.Where("id = ?", id).First(&task).Error
	return &task, err
}

func (r *taskRepository) ListByColumnID(columnID string) ([]models.Task, error) {
	var tasks []models.Task
	err := r.db.Where("column_id = ?", columnID).Order("position ASC").Find(&tasks).Error
	return tasks, err
}

func (r *taskRepository) Update(task *models.Task) error {
	return r.db.Save(task).Error
}

func (r *taskRepository) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.Task{}).Error
}

func (r *taskRepository) Reorder(columnID string, taskIDs []string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for i, id := range taskIDs {
			if err := tx.Model(&models.Task{}).
				Where("id = ? AND column_id = ?", id, columnID).
				Update("position", float64((i+1)*1000)).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *taskRepository) CountByParentID(parentID string, count *int64) error {
	return r.db.Model(&models.Task{}).Where("parent_id = ? AND deleted_at IS NULL", parentID).Count(count).Error
}

func (r *taskRepository) GetNextTaskNumber(projectID string) (int, error) {
	var maxNum struct {
		Max int `gorm:"column:max"`
	}
	err := r.db.Table("tasks").
		Select("COALESCE(MAX(task_number), 0) + 1 as max").
		Where("project_id = ? AND deleted_at IS NULL", projectID).
		Scan(&maxNum).Error
	return maxNum.Max, err
}

func (r *taskRepository) CascadeDelete(taskID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var subtaskIDs []string
		tx.Table("tasks").Where("parent_id = ?", taskID).Pluck("id", &subtaskIDs)

		allIDs := append([]string{taskID}, subtaskIDs...)

		if err := tx.Model(&models.Comment{}).
			Where("task_id IN ?", allIDs).
			Update("deleted_at", gorm.Expr("NOW()")).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.Attachment{}).
			Where("task_id IN ?", allIDs).
			Update("deleted_at", gorm.Expr("NOW()")).Error; err != nil {
			return err
		}

		if err := tx.Where("id IN ?", subtaskIDs).Delete(&models.Task{}).Error; err != nil {
			return err
		}

		return tx.Where("id = ?", taskID).Delete(&models.Task{}).Error
	})
}

func (r *taskRepository) ListIDsByParentID(parentID string) ([]string, error) {
	var ids []string
	err := r.db.Table("tasks").
		Where("parent_id = ? AND deleted_at IS NULL", parentID).
		Pluck("id", &ids).Error
	return ids, err
}

func (r *taskRepository) GetMaxPositionInColumn(projectID, columnID string) (float64, error) {
	var result struct {
		Max float64 `gorm:"column:max"`
	}
	err := r.db.Table("tasks").
		Select("COALESCE(MAX(position), 0) as max").
		Where("project_id = ? AND column_id = ? AND parent_id IS NULL AND deleted_at IS NULL", projectID, columnID).
		Scan(&result).Error
	return result.Max, err
}

func (r *taskRepository) GetMaxPositionInParent(parentID string) (float64, error) {
	var result struct {
		Max float64 `gorm:"column:max"`
	}
	err := r.db.Table("tasks").
		Select("COALESCE(MAX(position), 0) as max").
		Where("parent_id = ? AND deleted_at IS NULL", parentID).
		Scan(&result).Error
	return result.Max, err
}

func (r *taskRepository) ExistsInColumn(columnID string, position float64) (bool, error) {
	var count int64
	err := r.db.Model(&models.Task{}).
		Where("column_id = ? AND position = ? AND deleted_at IS NULL", columnID, position).
		Count(&count).Error
	return count > 0, err
}

func (r *taskRepository) CountBetweenPositions(columnID string, prevPos, nextPos float64) (int64, error) {
	var count int64
	err := r.db.Model(&models.Task{}).
		Where("column_id = ? AND position > ? AND position < ? AND deleted_at IS NULL", columnID, prevPos, nextPos).
		Count(&count).Error
	return count, err
}

func (r *taskRepository) UpdatePositionAtomic(taskID string, columnID string, position float64, lastKnownUpdatedAt time.Time) (int64, error) {
	result := r.db.Model(&models.Task{}).
		Where("id = ? AND updated_at = ?", taskID, lastKnownUpdatedAt).
		Updates(map[string]interface{}{
			"column_id":  columnID,
			"position":   position,
			"updated_at": time.Now(),
		})
	return result.RowsAffected, result.Error
}

func (r *taskRepository) RebalanceColumn(columnID string) ([]dto.TaskPositionInfo, error) {
	var tasks []models.Task
	if err := r.db.Where("column_id = ? AND deleted_at IS NULL", columnID).
		Order("position ASC, created_at DESC").
		Find(&tasks).Error; err != nil {
		return nil, err
	}
	positions := make([]dto.TaskPositionInfo, len(tasks))
	for i, t := range tasks {
		newPos := float64((i + 1) * 1000)
		positions[i] = dto.TaskPositionInfo{ID: t.ID, Position: newPos}
		if t.Position != newPos {
			if err := r.db.Model(&models.Task{}).
				Where("id = ?", t.ID).
				Update("position", newPos).Error; err != nil {
				return nil, err
			}
		}
	}
	return positions, nil
}

func (r *taskRepository) ListOverdueIDs() ([]string, error) {
	var ids []string
	err := r.db.Table("tasks t").
		Joins("JOIN columns c ON c.id = t.column_id").
		Where("t.due_date < NOW() AND t.due_date IS NOT NULL AND c.is_done = false AND t.deleted_at IS NULL").
		Pluck("t.id", &ids).Error
	return ids, err
}

func (r *taskRepository) GetCreateTaskResponse(taskID string) (*dto.TaskCreateResponse, error) {
	var row struct {
		ID          string     `gorm:"column:id"`
		TaskNumber  int        `gorm:"column:task_number"`
		TaskRef     string     `gorm:"column:task_ref"`
		Title       string     `gorm:"column:title"`
		Description *string    `gorm:"column:description"`
		Priority    string     `gorm:"column:priority"`
		StartDate   *time.Time `gorm:"column:start_date"`
		DueDate     *time.Time `gorm:"column:due_date"`
		Position    float64    `gorm:"column:position"`
		CreatedAt   time.Time  `gorm:"column:created_at"`
		ColumnID    string     `gorm:"column:column_id"`
		ColumnTitle string     `gorm:"column:column_title"`
		CreatorID   string     `gorm:"column:creator_id"`
		CreatorName string     `gorm:"column:creator_name"`
		ParentID    *string    `gorm:"column:parent_id"`
	}
	err := r.db.Table("tasks t").
		Select(`
			t.id, t.task_number, CONCAT(p.key, '-', t.task_number) as task_ref,
			t.title, t.description, t.priority,
			t.start_date, t.due_date, t.position, t.created_at,
			c.id as column_id, c.title as column_title,
			u.id as creator_id, u.full_name as creator_name,
			t.parent_id`).
		Joins("JOIN columns c ON c.id = t.column_id").
		Joins("JOIN projects p ON p.id = t.project_id").
		Joins("JOIN users u ON u.id = t.creator_id").
		Where("t.id = ? AND t.deleted_at IS NULL", taskID).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	return &dto.TaskCreateResponse{
		ID:          row.ID,
		TaskNumber:  row.TaskNumber,
		TaskRef:     row.TaskRef,
		Title:       row.Title,
		Description: row.Description,
		Priority:    row.Priority,
		StartDate:   row.StartDate,
		DueDate:     row.DueDate,
		Column:      dto.ColumnRef{ID: row.ColumnID, Title: row.ColumnTitle},
		Creator:     dto.CreatorInfo{UserID: row.CreatorID, FullName: row.CreatorName},
		ParentID:    row.ParentID,
		Position:    row.Position,
		CreatedAt:   row.CreatedAt,
	}, nil
}

func (r *taskRepository) ListWithFilters(projectID string, filters map[string]interface{}, page int, limit int) ([]dto.TaskSummary, *dto.Pagination, error) {
	buildQuery := func(tx *gorm.DB) *gorm.DB {
		q := tx.Where("t.project_id = ? AND t.parent_id IS NULL AND t.deleted_at IS NULL", projectID).
			Joins("JOIN columns c ON c.id = t.column_id")

		if v, ok := filters["column_id"]; ok && v != "" {
			q = q.Where("t.column_id = ?", v)
		}
		if v, ok := filters["priority"]; ok && v != "" {
			q = q.Where("t.priority = ?", v)
		}
		if v, ok := filters["assignee_id"]; ok && v != "" {
			q = q.Where("EXISTS (SELECT 1 FROM task_assignees ta WHERE ta.task_id = t.id AND ta.user_id = ?)", v)
		}
		if v, ok := filters["label_id"]; ok && v != "" {
			q = q.Where("EXISTS (SELECT 1 FROM task_labels tl WHERE tl.task_id = t.id AND tl.label_id = ?)", v)
		}
		if v, ok := filters["due_date_from"]; ok && v != "" {
			q = q.Where("t.due_date >= ?", v)
		}
		if v, ok := filters["due_date_to"]; ok && v != "" {
			q = q.Where("t.due_date <= ?", v)
		}
		if v, ok := filters["has_assignee"]; ok && v != nil {
			if b, ok2 := v.(bool); ok2 && b {
				q = q.Where("EXISTS (SELECT 1 FROM task_assignees ta2 WHERE ta2.task_id = t.id)")
			} else {
				q = q.Where("NOT EXISTS (SELECT 1 FROM task_assignees ta2 WHERE ta2.task_id = t.id)")
			}
		}
		if v, ok := filters["has_label"]; ok && v != nil {
			if b, ok2 := v.(bool); ok2 && b {
				q = q.Where("EXISTS (SELECT 1 FROM task_labels tl3 WHERE tl3.task_id = t.id)")
			} else {
				q = q.Where("NOT EXISTS (SELECT 1 FROM task_labels tl3 WHERE tl3.task_id = t.id)")
			}
		}
		if v, ok := filters["search"]; ok && v != "" {
			search := "%" + v.(string) + "%"
			q = q.Where("t.title ILIKE ? OR CAST(t.task_number AS TEXT) ILIKE ?", search, search)
		}
		return q
	}

	var total int64
	if err := buildQuery(r.db.Table("tasks t")).
		Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, nil, err
	}

	offset := (page - 1) * limit
	var rows []projection.TaskSummaryRow
	if err := buildQuery(r.db.Table("tasks t").
		Select(`
			t.id, t.task_number, CONCAT(p.key, '-', t.task_number) as task_ref,
			t.title, t.priority, t.due_date, t.position, t.created_at,
			c.id as column_id, c.title as column_title,
			(SELECT COUNT(*) FROM tasks sub WHERE sub.parent_id = t.id AND sub.deleted_at IS NULL) as subtask_count,
			(SELECT COUNT(*) FROM comments cm WHERE cm.task_id = t.id AND cm.deleted_at IS NULL) as comment_count,
			(SELECT COUNT(*) FROM attachments att WHERE att.task_id = t.id AND att.deleted_at IS NULL) as attachment_count`).
		Joins("JOIN projects p ON p.id = t.project_id")).
		Offset(offset).Limit(limit).Order("t.position ASC, t.created_at DESC").
		Scan(&rows).Error; err != nil {
		return nil, nil, err
	}

	tasks := make([]dto.TaskSummary, len(rows))
	for i, row := range rows {
		tasks[i] = dto.TaskSummary{
			ID:              row.ID,
			TaskNumber:      row.TaskNumber,
			TaskRef:         row.TaskRef,
			Title:           row.Title,
			Priority:        row.Priority,
			DueDate:         row.DueDate,
			Column:          dto.ColumnRef{ID: row.ColumnID, Title: row.ColumnTitle},
			SubtaskCount:    row.SubtaskCount,
			CommentCount:    row.CommentCount,
			AttachmentCount: row.AttachmentCount,
			Position:        row.Position,
			CreatedAt:       row.CreatedAt,
		}
	}

	return tasks, dto.NewPagination(total, dto.PaginationParam{Page: page, Limit: limit}), nil
}

func (r *taskRepository) GetByIDWithDetail(taskID string) (*dto.TaskDetailResponse, error) {
	var row projection.TaskDetailRow
	err := r.db.Table("tasks t").
		Select(`
			t.id, t.task_number, CONCAT(p.key, '-', t.task_number) as task_ref,
			t.title, t.description, t.priority,
			t.start_date, t.due_date, t.position, t.created_at, t.updated_at,
			c.id as column_id, c.title as column_title,
			p.id as project_id, p.name as project_name, p.key as project_key,
			u.id as creator_id, u.full_name as creator_name, u.avatar_url as creator_avatar,
			parent.id as parent_id, parent.task_number as parent_task_number,
			CONCAT(p.key, '-', parent.task_number) as parent_task_ref, parent.title as parent_title,
			(SELECT COUNT(*) FROM tasks sub WHERE sub.parent_id = t.id AND sub.deleted_at IS NULL) as subtask_count,
			(SELECT COUNT(*) FROM tasks sub
				JOIN columns done_col ON done_col.id = sub.column_id
				WHERE sub.parent_id = t.id AND sub.deleted_at IS NULL AND done_col.is_done = true) as subtask_done_count,
			(SELECT COUNT(*) FROM comments cm WHERE cm.task_id = t.id AND cm.deleted_at IS NULL) as comment_count,
			(SELECT COUNT(*) FROM attachments att WHERE att.task_id = t.id AND att.deleted_at IS NULL) as attachment_count`).
		Joins("JOIN columns c ON c.id = t.column_id").
		Joins("JOIN projects p ON p.id = t.project_id").
		Joins("LEFT JOIN users u ON u.id = t.creator_id").
		Joins("LEFT JOIN tasks parent ON parent.id = t.parent_id AND parent.deleted_at IS NULL").
		Where("t.id = ? AND t.deleted_at IS NULL", taskID).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == "" {
		return nil, gorm.ErrRecordNotFound
	}

	result := &dto.TaskDetailResponse{
		ID:          row.ID,
		TaskNumber:  row.TaskNumber,
		TaskRef:     row.TaskRef,
		Title:       row.Title,
		Description: row.Description,
		Priority:    row.Priority,
		StartDate:   row.StartDate,
		DueDate:     row.DueDate,
		Column: dto.ColumnRef{
			ID:    row.ColumnID,
			Title: row.ColumnTitle,
		},
		Project: dto.ProjectRef{
			ID:   row.ProjectID,
			Name: row.ProjectName,
			Key:  row.ProjectKey,
		},
		Creator: func() *dto.CreatorInfo {
			if row.CreatorID == nil {
				return nil
			}
			return &dto.CreatorInfo{
				UserID:    *row.CreatorID,
				FullName:  *row.CreatorName,
				AvatarURL: row.CreatorAvatar,
			}
		}(),
		Parent: func() *dto.TaskParentRef {
			if row.ParentID == nil {
				return nil
			}
			return &dto.TaskParentRef{
				ID:         *row.ParentID,
				TaskNumber: *row.ParentTaskNumber,
				TaskRef:    *row.ParentTaskRef,
				Title:      *row.ParentTitle,
			}
		}(),
		SubtaskCount:     row.SubtaskCount,
		SubtaskDoneCount: row.SubtaskDoneCount,
		CommentCount:     row.CommentCount,
		AttachmentCount:  row.AttachmentCount,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}

	return result, nil
}

func (r *taskRepository) ListSubtasks(taskID string) (*dto.SubtaskListResponse, error) {
	var parentRow struct {
		ID         string `gorm:"column:id"`
		TaskNumber int    `gorm:"column:task_number"`
		TaskRef    string `gorm:"column:task_ref"`
		Title      string `gorm:"column:title"`
	}
	err := r.db.Table("tasks t").
		Select("t.id, t.task_number, CONCAT(p.key, '-', t.task_number) as task_ref, t.title").
		Joins("JOIN projects p ON p.id = t.project_id").
		Where("t.id = ? AND t.deleted_at IS NULL", taskID).
		Scan(&parentRow).Error
	if err != nil {
		return nil, err
	}
	if parentRow.ID == "" {
		return nil, gorm.ErrRecordNotFound
	}

	var progressRow projection.SubtaskCountRow
	err = r.db.Table("tasks sub").
		Select(`
			COUNT(*) as subtask_count,
			COUNT(*) FILTER (WHERE EXISTS (SELECT 1 FROM columns dc WHERE dc.id = sub.column_id AND dc.is_done = true)) as done_count`).
		Where("sub.parent_id = ? AND sub.deleted_at IS NULL", taskID).
		Scan(&progressRow).Error
	if err != nil {
		return nil, err
	}

	var rows []projection.SubtaskDetailRow
	err = r.db.Table("tasks sub").
		Select(`
			sub.id, sub.task_number, CONCAT(p.key, '-', sub.task_number) as task_ref,
			sub.title, sub.priority, sub.due_date, sub.position,
			c.id as column_id, c.title as column_title,
			(SELECT COUNT(*) FROM comments cm WHERE cm.task_id = sub.id AND cm.deleted_at IS NULL) as comment_count`).
		Joins("JOIN columns c ON c.id = sub.column_id").
		Joins("JOIN projects p ON p.id = sub.project_id").
		Where("sub.parent_id = ? AND sub.deleted_at IS NULL", taskID).
		Order("sub.position ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	total := progressRow.SubtaskCount
	done := progressRow.DoneCount
	percentage := 0
	if total > 0 {
		percentage = done * 100 / total
	}

	subtasks := make([]dto.SubtaskInfo, len(rows))
	for i, row := range rows {
		subtasks[i] = dto.SubtaskInfo{
			ID:           row.ID,
			TaskNumber:   row.TaskNumber,
			TaskRef:      row.TaskRef,
			Title:        row.Title,
			Priority:     row.Priority,
			DueDate:      row.DueDate,
			Column:       dto.ColumnRef{ID: row.ColumnID, Title: row.ColumnTitle},
			Position:     row.Position,
			CommentCount: row.CommentCount,
		}
	}

	return &dto.SubtaskListResponse{
		Parent: dto.TaskParentRef{
			ID:         parentRow.ID,
			TaskNumber: parentRow.TaskNumber,
			TaskRef:    parentRow.TaskRef,
			Title:      parentRow.Title,
		},
		Progress: dto.SubtaskProgress{
			Total:      total,
			Done:       done,
			Percentage: percentage,
		},
		Data: subtasks,
	}, nil
}

func (r *taskRepository) applyBoardFilters(q *gorm.DB, filters map[string]interface{}) *gorm.DB {
	if v, ok := filters["priority"]; ok {
		if arr, ok2 := v.([]string); ok2 && len(arr) > 0 {
			q = q.Where("t.priority IN ?", arr)
		}
	}
	if v, ok := filters["assignee_id"]; ok {
		if arr, ok2 := v.([]string); ok2 && len(arr) > 0 {
			q = q.Where("EXISTS (SELECT 1 FROM task_assignees ta WHERE ta.task_id = t.id AND ta.user_id IN ?)", arr)
		}
	}
	if v, ok := filters["label_id"]; ok {
		if arr, ok2 := v.([]string); ok2 && len(arr) > 0 {
			q = q.Where("EXISTS (SELECT 1 FROM task_labels tl WHERE tl.task_id = t.id AND tl.label_id IN ?)", arr)
		}
	}
	if v, ok := filters["due_date_from"]; ok && v != "" {
		q = q.Where("t.due_date >= ?", v)
	}
	if v, ok := filters["due_date_to"]; ok && v != "" {
		q = q.Where("t.due_date <= ?", v)
	}
	if v, ok := filters["creator_id"]; ok && v != "" {
		q = q.Where("t.creator_id = ?", v)
	}
	if v, ok := filters["has_assignee"]; ok && v != nil {
		if b, ok2 := v.(bool); ok2 && b {
			q = q.Where("EXISTS (SELECT 1 FROM task_assignees ta2 WHERE ta2.task_id = t.id)")
		} else {
			q = q.Where("NOT EXISTS (SELECT 1 FROM task_assignees ta2 WHERE ta2.task_id = t.id)")
		}
	}
	if v, ok := filters["has_label"]; ok && v != nil {
		if b, ok2 := v.(bool); ok2 && b {
			q = q.Where("EXISTS (SELECT 1 FROM task_labels tl2 WHERE tl2.task_id = t.id)")
		} else {
			q = q.Where("NOT EXISTS (SELECT 1 FROM task_labels tl2 WHERE tl2.task_id = t.id)")
		}
	}
	if v, ok := filters["overdue"]; ok && v != nil {
		if b, ok2 := v.(bool); ok2 && b {
			q = q.Where("t.due_date IS NOT NULL AND t.due_date < NOW()")
		}
	}
	if v, ok := filters["search"]; ok && v != "" {
		s := "%" + v.(string) + "%"
		q = q.Where("t.title ILIKE ? OR CAST(t.task_number AS TEXT) ILIKE ?", s, s)
	}
	return q
}

func (r *taskRepository) SearchWithFilters(projectID string, filters map[string]interface{}, page int, limit int, sortBy string, sortDir string) ([]dto.TaskSummary, *dto.Pagination, error) {
	page = max(page, 1)
	if limit <= 0 {
		limit = 20
	}

	includeSubtasks := false
	if v, ok := filters["include_subtasks"]; ok {
		if b, ok2 := v.(bool); ok2 {
			includeSubtasks = b
		}
	}

	buildQuery := func(tx *gorm.DB) *gorm.DB {
		q := tx.Where("t.project_id = ? AND t.deleted_at IS NULL", projectID).
			Joins("JOIN columns c ON c.id = t.column_id")

		if !includeSubtasks {
			q = q.Where("t.parent_id IS NULL")
		}

		q = r.applyBoardFilters(q, filters)
		return q
	}

	validSortFields := map[string]bool{
		"position": true, "due_date": true, "priority": true, "created_at": true,
	}
	if !validSortFields[sortBy] {
		sortBy = "position"
	}
	sortDir = strings.ToLower(sortDir)
	if sortDir != "desc" {
		sortDir = "asc"
	}

	orderClause := fmt.Sprintf("t.%s %s", sortBy, sortDir)
	if sortBy == "position" {
		orderClause += ", t.created_at DESC"
	}

	var total int64
	if err := buildQuery(r.db.Table("tasks t")).
		Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, nil, err
	}

	offset := (page - 1) * limit
	var rows []projection.TaskSummaryRow
	selectStr := `
		t.id, t.task_number, CONCAT(p.key, '-', t.task_number) as task_ref,
		t.title, t.priority, t.due_date, t.position, t.created_at,
		c.id as column_id, c.title as column_title,
		(SELECT COUNT(*) FROM tasks sub WHERE sub.parent_id = t.id AND sub.deleted_at IS NULL) as subtask_count,
		(SELECT COUNT(*) FROM comments cm WHERE cm.task_id = t.id AND cm.deleted_at IS NULL) as comment_count,
		(SELECT COUNT(*) FROM attachments att WHERE att.task_id = t.id AND att.deleted_at IS NULL) as attachment_count`

	if err := buildQuery(r.db.Table("tasks t").
		Select(selectStr).
		Joins("JOIN projects p ON p.id = t.project_id")).
		Offset(offset).Limit(limit).Order(orderClause).
		Scan(&rows).Error; err != nil {
		return nil, nil, err
	}

	now := time.Now()
	tasks := make([]dto.TaskSummary, len(rows))
	taskIDs := make([]string, len(rows))
	for i, row := range rows {
		taskIDs[i] = row.ID
		tasks[i] = dto.TaskSummary{
			ID:              row.ID,
			TaskNumber:      row.TaskNumber,
			TaskRef:         row.TaskRef,
			Title:           row.Title,
			Priority:        row.Priority,
			DueDate:         row.DueDate,
			Column:          dto.ColumnRef{ID: row.ColumnID, Title: row.ColumnTitle},
			SubtaskCount:    row.SubtaskCount,
			CommentCount:    row.CommentCount,
			AttachmentCount: row.AttachmentCount,
			Position:        row.Position,
			CreatedAt:       row.CreatedAt,
		}
		if row.DueDate != nil && row.DueDate.Before(now) {
			tasks[i].IsOverdue = true
		}
	}

	assigneeMap, _ := r.getAssigneeMap(taskIDs)
	labelMap, _ := r.getLabelMap(taskIDs)
	for i := range tasks {
		tasks[i].Assignees = assigneeMap[tasks[i].ID]
		tasks[i].Labels = labelMap[tasks[i].ID]
	}

	return tasks, dto.NewPagination(total, dto.PaginationParam{Page: page, Limit: limit}), nil
}

func (r *taskRepository) getAssigneeMap(taskIDs []string) (map[string][]dto.AssigneeInfo, error) {
	type assigneeRow struct {
		TaskID   string  `gorm:"column:task_id"`
		UserID   string  `gorm:"column:user_id"`
		FullName string  `gorm:"column:full_name"`
		Avatar   *string `gorm:"column:avatar_url"`
	}
	var rows []assigneeRow
	if err := r.db.Table("task_assignees ta").
		Select("ta.task_id, u.id as user_id, u.full_name, u.avatar_url").
		Joins("JOIN users u ON u.id = ta.user_id").
		Where("ta.task_id IN ?", taskIDs).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string][]dto.AssigneeInfo, len(taskIDs))
	for _, row := range rows {
		result[row.TaskID] = append(result[row.TaskID], dto.AssigneeInfo{
			UserID:    row.UserID,
			FullName:  row.FullName,
			AvatarURL: row.Avatar,
		})
	}
	return result, nil
}

func (r *taskRepository) getLabelMap(taskIDs []string) (map[string][]dto.LabelRef, error) {
	type labelRow struct {
		TaskID string `gorm:"column:task_id"`
		ID     string `gorm:"column:id"`
		Name   string `gorm:"column:name"`
		Color  string `gorm:"column:color"`
	}
	var rows []labelRow
	if err := r.db.Table("task_labels tl").
		Select("tl.task_id, l.id, l.name, l.color").
		Joins("JOIN labels l ON l.id = tl.label_id").
		Where("tl.task_id IN ?", taskIDs).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string][]dto.LabelRef, len(taskIDs))
	for _, row := range rows {
		result[row.TaskID] = append(result[row.TaskID], dto.LabelRef{
			ID:    row.ID,
			Name:  row.Name,
			Color: row.Color,
		})
	}
	return result, nil
}

func (r *taskRepository) GetBoardByProjectID(projectID string, filters map[string]interface{}, tasksPerColumn int) (*dto.BoardResponse, error) {
	var proj struct {
		ID         string `gorm:"column:id"`
		Name       string `gorm:"column:name"`
		Key        string `gorm:"column:key"`
		IsArchived bool   `gorm:"column:is_archived"`
	}
	if err := r.db.Table("projects").
		Select("id, name, key, is_archived").
		Where("id = ? AND deleted_at IS NULL", projectID).
		Scan(&proj).Error; err != nil {
		return nil, err
	}
	if proj.ID == "" {
		return nil, gorm.ErrRecordNotFound
	}

	var cols []models.Column
	if err := r.db.Where("project_id = ? AND deleted_at IS NULL", projectID).
		Order("position ASC").Find(&cols).Error; err != nil {
		return nil, err
	}

	now := time.Now()
	columns := make([]dto.BoardColumn, 0, len(cols))
	var allTaskIDs []string

	for _, col := range cols {
		var taskTotal int64
		r.db.Model(&models.Task{}).
			Where("column_id = ? AND parent_id IS NULL AND deleted_at IS NULL", col.ID).
			Count(&taskTotal)

		filteredTotalQ := r.db.Table("tasks t").
			Where("t.column_id = ? AND t.parent_id IS NULL AND t.deleted_at IS NULL", col.ID)
		filteredTotalQ = r.applyBoardFilters(filteredTotalQ, filters)
		var taskFiltered int64
		filteredTotalQ.Count(&taskFiltered)

		taskQ := r.db.Table("tasks t").
			Select(`
				t.id, t.task_number, CONCAT(p.key, '-', t.task_number) as task_ref,
				t.title, t.priority, t.due_date, t.position,
				c.id as column_id, c.title as column_title,
				(SELECT COUNT(*) FROM tasks sub WHERE sub.parent_id = t.id AND sub.deleted_at IS NULL) as subtask_count,
				(SELECT COUNT(*) FROM tasks sub
					JOIN columns done_col ON done_col.id = sub.column_id
					WHERE sub.parent_id = t.id AND sub.deleted_at IS NULL AND done_col.is_done = true) as subtask_done_count,
				(SELECT COUNT(*) FROM comments cm WHERE cm.task_id = t.id AND cm.deleted_at IS NULL) as comment_count,
				(SELECT COUNT(*) FROM attachments att WHERE att.task_id = t.id AND att.deleted_at IS NULL) as attachment_count`).
			Joins("JOIN columns c ON c.id = t.column_id").
			Joins("JOIN projects p ON p.id = t.project_id").
			Where("t.column_id = ? AND t.parent_id IS NULL AND t.deleted_at IS NULL", col.ID)

		taskQ = r.applyBoardFilters(taskQ, filters)
		taskQ = taskQ.Order("t.position ASC, t.created_at DESC")

		var boardRows []projection.BoardTaskRow
		if err := taskQ.Limit(tasksPerColumn + 1).Scan(&boardRows).Error; err != nil {
			return nil, err
		}

		hasMore := len(boardRows) > tasksPerColumn
		if hasMore {
			boardRows = boardRows[:tasksPerColumn]
		}

		colTaskIDs := make([]string, len(boardRows))
		tasks := make([]dto.BoardTaskInfo, len(boardRows))
		for i, row := range boardRows {
			colTaskIDs[i] = row.ID
			allTaskIDs = append(allTaskIDs, row.ID)
			tasks[i] = dto.BoardTaskInfo{
				ID:               row.ID,
				TaskNumber:       row.TaskNumber,
				TaskRef:          row.TaskRef,
				Title:            row.Title,
				Priority:         row.Priority,
				DueDate:          nil,
				IsOverdue:        false,
				Position:         row.Position,
				SubtaskCount:     row.SubtaskCount,
				SubtaskDoneCount: row.SubtaskDoneCount,
				CommentCount:     row.CommentCount,
				AttachmentCount:  row.AttachmentCount,
			}
			if row.DueDate != nil {
				s := row.DueDate.Format(time.RFC3339)
				tasks[i].DueDate = &s
				if row.DueDate.Before(now) {
					tasks[i].IsOverdue = true
				}
			}
		}

		var nextCursor *string
		if hasMore && len(boardRows) > 0 {
			lastTask := boardRows[len(boardRows)-1]
			raw := fmt.Sprintf("%.6f:%s", lastTask.Position, lastTask.ID)
			enc := base64.StdEncoding.EncodeToString([]byte(raw))
			nextCursor = &enc
		}

		bc := dto.BoardColumn{
			ID:           col.ID,
			Title:        col.Title,
			Position:     col.Position,
			TaskTotal:    int(taskTotal),
			TaskFiltered: int(taskFiltered),
			Tasks:        tasks,
			HasMore:      hasMore,
			NextCursor:   nextCursor,
		}
		columns = append(columns, bc)
	}

	assigneeMap, _ := r.getAssigneeMap(allTaskIDs)
	labelMap, _ := r.getLabelMap(allTaskIDs)
	for i := range columns {
		for j := range columns[i].Tasks {
			columns[i].Tasks[j].Assignees = assigneeMap[columns[i].Tasks[j].ID]
			columns[i].Tasks[j].Labels = labelMap[columns[i].Tasks[j].ID]
		}
	}

	filtersApplied := dto.BoardFilters{
		Priority:    nil,
		AssigneeID:  nil,
		LabelID:     nil,
		DueDateFrom: nil,
		DueDateTo:   nil,
		CreatorID:   nil,
		HasAssignee: nil,
		Overdue:     nil,
		Search:      nil,
	}
	if v, ok := filters["priority"]; ok {
		if arr, ok2 := v.([]string); ok2 {
			filtersApplied.Priority = arr
		}
	}
	if v, ok := filters["assignee_id"]; ok {
		if arr, ok2 := v.([]string); ok2 {
			filtersApplied.AssigneeID = arr
		}
	}
	if v, ok := filters["label_id"]; ok {
		if arr, ok2 := v.([]string); ok2 {
			filtersApplied.LabelID = arr
		}
	}
	if v, ok := filters["due_date_from"]; ok && v != "" {
		s := v.(string)
		filtersApplied.DueDateFrom = &s
	}
	if v, ok := filters["due_date_to"]; ok && v != "" {
		s := v.(string)
		filtersApplied.DueDateTo = &s
	}
	if v, ok := filters["creator_id"]; ok && v != "" {
		s := v.(string)
		filtersApplied.CreatorID = &s
	}
	if v, ok := filters["has_assignee"]; ok && v != nil {
		b := v.(bool)
		filtersApplied.HasAssignee = &b
	}
	if v, ok := filters["overdue"]; ok && v != nil {
		b := v.(bool)
		filtersApplied.Overdue = &b
	}
	if v, ok := filters["search"]; ok && v != "" {
		s := v.(string)
		filtersApplied.Search = &s
	}

	return &dto.BoardResponse{
		Project: dto.BoardProjectInfo{
			ID:         proj.ID,
			Name:       proj.Name,
			Key:        proj.Key,
			IsArchived: proj.IsArchived,
		},
		FiltersApplied: filtersApplied,
		Columns:        columns,
	}, nil
}

func (r *taskRepository) LoadMoreTasksInColumn(columnID string, cursor string, limit int, filters map[string]interface{}) (*dto.LoadMoreTasksResponse, error) {
	now := time.Now()

	taskQ := r.db.Table("tasks t").
		Select(`
			t.id, t.task_number, CONCAT(p.key, '-', t.task_number) as task_ref,
			t.title, t.priority, t.due_date, t.position,
			c.id as column_id, c.title as column_title,
			(SELECT COUNT(*) FROM tasks sub WHERE sub.parent_id = t.id AND sub.deleted_at IS NULL) as subtask_count,
			(SELECT COUNT(*) FROM tasks sub
				JOIN columns done_col ON done_col.id = sub.column_id
				WHERE sub.parent_id = t.id AND sub.deleted_at IS NULL AND done_col.is_done = true) as subtask_done_count,
			(SELECT COUNT(*) FROM comments cm WHERE cm.task_id = t.id AND cm.deleted_at IS NULL) as comment_count,
			(SELECT COUNT(*) FROM attachments att WHERE att.task_id = t.id AND att.deleted_at IS NULL) as attachment_count`).
		Joins("JOIN columns c ON c.id = t.column_id").
		Joins("JOIN projects p ON p.id = t.project_id").
		Where("t.column_id = ? AND t.parent_id IS NULL AND t.deleted_at IS NULL", columnID)

	if cursor != "" {
		decoded, err := base64.StdEncoding.DecodeString(cursor)
		if err == nil {
			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) == 2 {
				taskQ = taskQ.Where("(t.position > ?) OR (t.position = ? AND t.id > ?)", parts[0], parts[0], parts[1])
			}
		}
	}

	taskQ = r.applyBoardFilters(taskQ, filters)
	taskQ = taskQ.Order("t.position ASC, t.created_at DESC")

	var boardRows []projection.BoardTaskRow
	if err := taskQ.Limit(limit + 1).Scan(&boardRows).Error; err != nil {
		return nil, err
	}

	hasMore := len(boardRows) > limit
	if hasMore {
		boardRows = boardRows[:limit]
	}

	taskIDs := make([]string, len(boardRows))
	tasks := make([]dto.BoardTaskInfo, len(boardRows))
	for i, row := range boardRows {
		taskIDs[i] = row.ID
		tasks[i] = dto.BoardTaskInfo{
			ID:               row.ID,
			TaskNumber:       row.TaskNumber,
			TaskRef:          row.TaskRef,
			Title:            row.Title,
			Priority:         row.Priority,
			DueDate:          nil,
			IsOverdue:        false,
			Position:         row.Position,
			SubtaskCount:     row.SubtaskCount,
			SubtaskDoneCount: row.SubtaskDoneCount,
			CommentCount:     row.CommentCount,
			AttachmentCount:  row.AttachmentCount,
		}
		if row.DueDate != nil {
			s := row.DueDate.Format(time.RFC3339)
			tasks[i].DueDate = &s
			if row.DueDate.Before(now) {
				tasks[i].IsOverdue = true
			}
		}
	}

	assigneeMap, _ := r.getAssigneeMap(taskIDs)
	labelMap, _ := r.getLabelMap(taskIDs)
	for i := range tasks {
		tasks[i].Assignees = assigneeMap[tasks[i].ID]
		tasks[i].Labels = labelMap[tasks[i].ID]
	}

	var nextCursor *string
	if hasMore && len(boardRows) > 0 {
		lastTask := boardRows[len(boardRows)-1]
		raw := fmt.Sprintf("%.6f:%s", lastTask.Position, lastTask.ID)
		enc := base64.StdEncoding.EncodeToString([]byte(raw))
		nextCursor = &enc
	}

	return &dto.LoadMoreTasksResponse{
		ColumnID:   columnID,
		Tasks:      tasks,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	}, nil
}
