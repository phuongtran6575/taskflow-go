package _interface

import (
	"TaskFlow-Go/internal/dto"
	"mime/multipart"
)

type UserService interface {
	GetProfile(userID string) (*dto.UserProfileResponse, error)
	UpdateProfile(userID string, req *dto.UpdateProfileRequest) (*dto.UpdateProfileResponse, error)
	UploadAvatar(userID string, fileHeader *multipart.FileHeader) (*dto.AvatarResponse, error)
	RemoveAvatar(userID string) (*dto.AvatarResponse, error)
	ChangePassword(userID string, req *dto.ChangePasswordRequest, userAgent string, ipAddress string) (*dto.ChangePasswordResponse, error)
	DeleteAccount(userID string, req *dto.DeleteAccountRequest) error
}
