package projection

import "time"

type WorkspaceDetailRow struct {
	ID             string    `gorm:"column:id"`
	Name           string    `gorm:"column:name"`
	Domain         *string   `gorm:"column:domain"`
	Plan           string    `gorm:"column:plan"`
	OwnerID        string    `gorm:"column:owner_id"`
	OwnerFullName  string    `gorm:"column:owner_full_name"`
	OwnerAvatarURL *string   `gorm:"column:owner_avatar_url"`
	MemberCount    int       `gorm:"column:member_count"`
	ProjectCount   int       `gorm:"column:project_count"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}
