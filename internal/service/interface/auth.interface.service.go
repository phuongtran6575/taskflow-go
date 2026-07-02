package _interface

import "TaskFlow-Go/internal/dto"

type AuthService interface {
	Register(req *dto.RegisterRequest) (*dto.AuthResponse, error)
	Login(req *dto.LoginRequest) (*dto.AuthResponse, error)
	Logout(userID string, req *dto.LogoutRequest) error
	RefreshToken(req *dto.RefreshTokenRequest) (*dto.TokenResponse, error)
	ForgotPassword(req *dto.ForgotPasswordRequest) (*dto.ForgotPasswordResponse, error)
	ResetPassword(req *dto.ResetPasswordRequest) (*dto.ResetPasswordResponse, error)
	GetGoogleOAuthUrl() (string, error)
	ProcessGoogleCallback(code string) (*dto.AuthResponse, error)
	GetGithubOAuthUrl() (string, error)
	ProcessGithubCallback(code string) (*dto.AuthResponse, error)
}
