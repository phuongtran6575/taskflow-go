package models

import (
	"time"

	"gorm.io/gorm"
)

type Session struct {
	ID           string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID       string         `gorm:"type:uuid;not null;index"`
	RefreshToken string         `gorm:"type:text;not null"`
	UserAgent    string         `gorm:"type:text"`
	IPAddress    string         `gorm:"type:varchar(45)"`
	IsRevoked    bool           `gorm:"not null;default:false"`
	LastUsedAt   *time.Time     `gorm:"type:timestamp"`
	ExpiresAt    time.Time      `gorm:"type:timestamp;not null"`
	CreatedAt    time.Time      `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	DeletedAt    gorm.DeletedAt

	User User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

func (Session) TableName() string { return "sessions" }
