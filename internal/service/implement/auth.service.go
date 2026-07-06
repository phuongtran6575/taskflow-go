package implement

import (
	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/helper"
	"TaskFlow-Go/internal/models"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"errors"
)

type authService struct {
	userRepo repoInterface.UserRepository
}

func NewAuthService(userRepo repoInterface.UserRepository) _interface.AuthService {
	return &authService{userRepo: userRepo}
}

func (s *authService) Register(req *dto.RegisterRequest) (*dto.AuthResponse, error) {
	user, err := s.userRepo.GetByEmail(req.Email)
	if err == nil && user != nil {
		return nil, errors.New("user already exists")
	}
	if !helper.CheckConfirmPassword(req.Password, req.ConfirmPassword) {
		return nil, errors.New("passwords do not match")
	}
	hashedPassword, err := helper.HashPassword(req.Password)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}
	user = &models.User{
		Email:        req.Email,
		PasswordHash: &hashedPassword,
		FullName:     req.FullName,
		Username:     req.Username,
		PhoneNumber:  req.PhoneNumber,
	}
	err = s.userRepo.Create(user)
	if err != nil {
		return nil, errors.New("failed to create user")
	}
	accessToken, err := helper.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, errors.New("failed to generate token")
	}
	return &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: "",
		User: dto.AuthUserInfo{
			ID:           user.ID,
			Email:        user.Email,
			FullName:     user.FullName,
			Username:     user.Username,
			AvatarURL:    user.AvatarURL,
			AuthProvider: string(user.AuthProvider),
		},
	}, nil
}

func (s *authService) Login(req *dto.LoginRequest) (*dto.AuthResponse, error) {
	user, err := s.userRepo.GetByEmail(req.Email)
	if err != nil {
		return nil, errors.New("user not found")
	}
	if err := helper.VerifyPassword(req.Password, *user.PasswordHash); err != nil {
		return nil, errors.New("invalid password")
	}
	accessToken, err := helper.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, errors.New("failed to generate token")
	}
	return &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: "",
		User: dto.AuthUserInfo{
			ID:           user.ID,
			Email:        user.Email,
			FullName:     user.FullName,
			Username:     user.Username,
			AvatarURL:    user.AvatarURL,
			AuthProvider: string(user.AuthProvider),
		},
	}, nil
}

func (s *authService) Logout(userID string, req *dto.LogoutRequest) error {
	return errors.New("not implemented")
}

func (s *authService) RefreshToken(req *dto.RefreshTokenRequest) (*dto.TokenResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *authService) ForgotPassword(req *dto.ForgotPasswordRequest) (*dto.ForgotPasswordResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *authService) ResetPassword(req *dto.ResetPasswordRequest) (*dto.ResetPasswordResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *authService) GetGoogleOAuthUrl() (string, error) {
	return "", errors.New("not implemented")
}

func (s *authService) ProcessGoogleCallback(code string) (*dto.AuthResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *authService) GetGithubOAuthUrl() (string, error) {
	return "", errors.New("not implemented")
}

func (s *authService) ProcessGithubCallback(code string) (*dto.AuthResponse, error) {
	return nil, errors.New("not implemented")
}
