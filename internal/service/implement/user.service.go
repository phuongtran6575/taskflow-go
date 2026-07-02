package implement

import (
	"TaskFlow-Go/internal/dto"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"errors"
	"mime/multipart"
)

type userService struct {
	userRepo repoInterface.UserRepository
}

func NewUserService(userRepo repoInterface.UserRepository) _interface.UserService {
	return &userService{userRepo: userRepo}
}

func (s *userService) GetProfile(userID string) (*dto.UserProfileResponse, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}
	return &dto.UserProfileResponse{
		ID:           user.ID,
		FullName:     user.FullName,
		Username:     user.Username,
		Email:        user.Email,
		PhoneNumber:  user.PhoneNumber,
		AvatarURL:    user.AvatarURL,
		AuthProvider: string(user.AuthProvider),
		IsActive:     user.IsActive,
		LastLogin:    user.LastLogin,
		CreatedAt:    user.CreatedAt,
	}, nil
}

func (s *userService) UpdateProfile(userID string, req *dto.UpdateProfileRequest) (*dto.UpdateProfileResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *userService) UploadAvatar(userID string, fileHeader *multipart.FileHeader) (*dto.AvatarResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *userService) RemoveAvatar(userID string) (*dto.AvatarResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *userService) ChangePassword(userID string, req *dto.ChangePasswordRequest) (*dto.ChangePasswordResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *userService) DeleteAccount(userID string, req *dto.DeleteAccountRequest) error {
	return errors.New("not implemented")
}
