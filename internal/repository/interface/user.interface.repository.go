package _interface

import (
	"TaskFlow-Go/internal/models"
)

type UserRepository interface {
	Create(user *models.User) error
	GetByID(id string) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	GetByUsername(username string) (*models.User, error)
	Update(id string, user *models.User) (*models.User, error)
	Delete(id string) error
	AnonymizeDelete(id string, updates map[string]interface{}) error
}
