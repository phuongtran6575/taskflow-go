package dto

import "time"

type UpdateProfileRequest struct {
	FullName *string `json:"full_name" binding:"omitempty,min=1"`
	Username *string `json:"username" binding:"omitempty,min=3,max=30"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" binding:"required,eqfield=NewPassword"`
}

type DeleteAccountRequest struct {
	Confirmation string  `json:"confirmation" binding:"required"`
	Password     *string `json:"password,omitempty"`
}

type UserProfileResponse struct {
	ID           string     `json:"id"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	PhoneNumber  string     `json:"phone_number"`
	FullName     string     `json:"full_name"`
	AvatarURL    *string    `json:"avatar_url"`
	AuthProvider string     `json:"auth_provider"`
	IsActive     bool       `json:"is_active"`
	LastLogin    *time.Time `json:"last_login"`
	CreatedAt    time.Time  `json:"created_at"`
}

type UpdateProfileResponse struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	PhoneNumber string    `json:"phone_number"`
	FullName    string    `json:"full_name"`
	AvatarURL   *string   `json:"avatar_url"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AvatarResponse struct {
	AvatarURL *string `json:"avatar_url"`
}

type ChangePasswordResponse struct {
	Message      string `json:"message"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

type SessionInfo struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	UserAgent string    `json:"user_agent"`
	IPAddress string    `json:"ip_address"`
	IsCurrent bool      `json:"is_current"`
}

type SessionListResponse struct {
	CurrentSessionID string        `json:"current_session_id"`
	Sessions         []SessionInfo `json:"sessions"`
}

type RevokeSessionResponse struct {
	Message string `json:"message"`
}

type RevokeAllSessionsResponse struct {
	Message      string `json:"message"`
	RevokedCount int    `json:"revoked_count"`
}

type DeleteAccountResponse struct {
	Message string `json:"message"`
}
