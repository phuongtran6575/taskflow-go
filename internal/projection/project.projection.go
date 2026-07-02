package projection

import "time"

type ProjectCreateRow struct {
	ID         string    `gorm:"column:id"`
	Name       string    `gorm:"column:name"`
	Key        string    `gorm:"column:key"`
	Icon       *string   `gorm:"column:icon"`
	IsArchived bool      `gorm:"column:is_archived"`
	Background string    `gorm:"column:background"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	OwnerID    string    `gorm:"column:owner_id"`
	FullName   string    `gorm:"column:full_name"`
	AvatarURL  *string   `gorm:"column:avatar_url"`
	ColumnID   string    `gorm:"column:column_id"`
	Title      string    `gorm:"column:title"`
	Position   float64   `gorm:"column:position"`
}

type ProjectSummaryRow struct {
	ID          string    `gorm:"column:id"`
	Name        string    `gorm:"column:name"`
	Key         string    `gorm:"column:key"`
	Icon        *string   `gorm:"column:icon"`
	IsArchived  bool      `gorm:"column:is_archived"`
	Background  string    `gorm:"column:background"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
	OwnerID     string    `gorm:"column:owner_id"`
	AvatarURL   *string   `gorm:"column:avatar_url"`
	FullName    string    `gorm:"column:full_name"`
	IsFavorite  bool      `gorm:"column:is_favorite"`
	RoleID      *string   `gorm:"column:role_id"`
	RoleName    *string   `gorm:"column:role_name"`
	JoinedAt    time.Time `gorm:"column:joined_at"`
	MemberCount  int `gorm:"column:member_count"`
	OpenTaskCount int `gorm:"column:open_task_count"`
}
