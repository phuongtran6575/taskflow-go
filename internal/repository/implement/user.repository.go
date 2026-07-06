package implement

import (
	"gorm.io/gorm"

	"TaskFlow-Go/internal/models"
	_interface "TaskFlow-Go/internal/repository/interface"
)

type userRepository struct{ db *gorm.DB }

// Update implements [_interface.UserRepository].

func NewUserRepository(db *gorm.DB) _interface.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Update(id string, user *models.User) (*models.User, error) {
	var updatedUser models.User
	if err := r.db.Where("id = ?", id).Updates(user).First(&updatedUser).Error; err != nil {
		return nil, err
	}
	return &updatedUser, nil
}

func (r *userRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *userRepository) GetByID(id string) (*models.User, error) {
	var user models.User
	err := r.db.Where("id = ?", id).First(&user).Error
	return &user, err
}

func (r *userRepository) GetByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.Where("email = ?", email).First(&user).Error
	return &user, err
}

func (r *userRepository) GetByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.db.Where("username = ?", username).First(&user).Error
	return &user, err
}

func (r *userRepository) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.User{}).Error
}

func (r *userRepository) AnonymizeDelete(id string, updates map[string]interface{}) error {
	return r.db.Model(&models.User{}).Where("id = ?", id).Updates(updates).Error
}
