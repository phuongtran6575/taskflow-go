package models

import (
	"time"
)

type NotificationType string

const (
	NotificationTypeASSIGNED          NotificationType = "ASSIGNED"
	NotificationTypeMENTIONED         NotificationType = "MENTIONED"
	NotificationTypeCOMMENTED         NotificationType = "COMMENTED"
	NotificationTypeSTATUSCHANGED     NotificationType = "STATUS_CHANGED"
	NotificationTypeADDEDTOPROJECT    NotificationType = "ADDED_TO_PROJECT"
	NotificationTypeADDEDTOWORKSPACE  NotificationType = "ADDED_TO_WORKSPACE"
	NotificationTypeTASKDUESOON       NotificationType = "TASK_DUE_SOON"
	NotificationTypeANNOUNCEMENT      NotificationType = "ANNOUNCEMENT"
	NotificationTypeCUSTOM            NotificationType = "CUSTOM"
)

type Notification struct {
	ID           string           `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ActorID      *string          `gorm:"type:uuid"`
	Type         NotificationType `gorm:"type:varchar(50);not null"`
	Title        string           `gorm:"type:varchar(255);not null"`
	Content      *string          `gorm:"type:text"`
	ReferenceURL *string          `gorm:"type:text"`
	CreatedAt    time.Time        `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`

	Actor      *User                    `gorm:"foreignKey:ActorID;constraint:OnDelete:SET NULL"`
	Recipients []NotificationRecipient  `gorm:"foreignKey:NotificationID;constraint:OnDelete:CASCADE"`
}

func (Notification) TableName() string { return "notifications" }

type NotificationRecipient struct {
	NotificationID string    `gorm:"type:uuid;primaryKey"`
	RecipientID    string    `gorm:"type:uuid;primaryKey"`
	IsRead         bool      `gorm:"not null;default:false"`
	ReadAt         *time.Time `gorm:"type:timestamp"`

	Notification Notification `gorm:"foreignKey:NotificationID;constraint:OnDelete:CASCADE"`
	Recipient    User         `gorm:"foreignKey:RecipientID;constraint:OnDelete:CASCADE"`
}

func (NotificationRecipient) TableName() string { return "notification_recipients" }
