package mapper

import (
	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
)

func ToUserFromRegisterRequest(req *dto.RegisterRequest, hashedPassword string) *models.User {
	return &models.User{
		Email:        req.Email,
		PasswordHash: &hashedPassword,
		FullName:     req.FullName,
		Username:     req.Username,
		PhoneNumber:  req.PhoneNumber,
	}
}
