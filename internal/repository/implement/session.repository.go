package implement

import (
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/models"
	_interface "TaskFlow-Go/internal/repository/interface"
)

type sessionRepository struct{ db *gorm.DB }

func NewSessionRepository(db *gorm.DB) _interface.SessionRepository {
	return &sessionRepository{db: db}
}

func (r *sessionRepository) GetByUserID(userID string) ([]models.Session, error) {
	var sessions []models.Session
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&sessions).Error
	return sessions, err
}

func (r *sessionRepository) GetByID(id string) (*models.Session, error) {
	var session models.Session
	err := r.db.Where("id = ?", id).First(&session).Error
	return &session, err
}

func (r *sessionRepository) GetCurrentSession(userID string, sessionID string) (*models.Session, error) {
	var session models.Session
	err := r.db.Where("id = ? AND user_id = ?", sessionID, userID).First(&session).Error
	return &session, err
}

func (r *sessionRepository) RevokeByID(id string) error {
	return r.db.Model(&models.Session{}).Where("id = ?", id).Update("is_revoked", true).Error
}

func (r *sessionRepository) RevokeAllExcept(userID string, excludeSessionID string) (int64, error) {
	result := r.db.Model(&models.Session{}).
		Where("user_id = ? AND id != ? AND is_revoked = false", userID, excludeSessionID).
		Update("is_revoked", true)
	return result.RowsAffected, result.Error
}

func (r *sessionRepository) CountActiveByUserID(userID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Session{}).
		Where("user_id = ? AND is_revoked = false", userID).
		Count(&count).Error
	return count, err
}

func (r *sessionRepository) RevokeOldestByUserID(userID string, keepCount int) error {
	subQuery := r.db.Model(&models.Session{}).
		Where("user_id = ? AND is_revoked = false", userID).
		Order("last_used_at ASC NULLS FIRST, created_at ASC").
		Limit(keepCount)

	return r.db.Model(&models.Session{}).
		Where("user_id = ? AND is_revoked = false AND id NOT IN (?)", userID, subQuery.Select("id")).
		Update("is_revoked", true).Error
}

func (r *sessionRepository) UpdateLastUsedAt(id string) error {
	now := time.Now()
	return r.db.Model(&models.Session{}).Where("id = ?", id).Update("last_used_at", now).Error
}

func (r *sessionRepository) Create(session *models.Session) error {
	return r.db.Create(session).Error
}
