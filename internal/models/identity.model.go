package models

import (
	"time"

	"gorm.io/gorm"
)

type AuthProvider string

const (
	AuthProviderEmail  AuthProvider = "EMAIL"
	AuthProviderGoogle AuthProvider = "GOOGLE"
	AuthProviderGithub AuthProvider = "GITHUB"
)

type User struct {
	ID           string       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Username     string       `gorm:"type:varchar(50);not null"`
	PhoneNumber  string       `gorm:"type:varchar(50);not null"`
	Email        string       `gorm:"type:varchar(255);not null"`
	PasswordHash *string      `gorm:"type:text"`
	FullName     string       `gorm:"type:varchar(100);not null"`
	AvatarURL    *string      `gorm:"type:text"`
	AuthProvider AuthProvider `gorm:"type:varchar(20);not null;default:'EMAIL'"`
	IsActive     bool         `gorm:"not null;default:true"`
	LastLogin    *time.Time   `gorm:"type:timestamp"`
	CreatedAt    time.Time    `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt    time.Time    `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	DeletedAt    gorm.DeletedAt

	Sessions []Session `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

func (User) TableName() string { return "users" }
