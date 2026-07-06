package _interface

import "TaskFlow-Go/internal/models"

type SessionRepository interface {
	GetByUserID(userID string) ([]models.Session, error)
	GetByID(id string) (*models.Session, error)
	GetCurrentSession(userID string, sessionID string) (*models.Session, error)
	RevokeByID(id string) error
	RevokeAllExcept(userID string, excludeSessionID string) (int64, error)
	CountActiveByUserID(userID string) (int64, error)
	RevokeOldestByUserID(userID string, keepCount int) error
	UpdateLastUsedAt(id string) error
	Create(session *models.Session) error
}
