package implement

import (
	"mime/multipart"
	"strings"
	"time"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/helper"
	"TaskFlow-Go/internal/models"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
	"TaskFlow-Go/internal/validator"

	"github.com/google/uuid"
)

type userService struct {
	userRepo            repoInterface.UserRepository
	sessionRepo         repoInterface.SessionRepository
	workspaceMemberRepo repoInterface.WorkspaceMemberRepository
	avatarStorage       _interface.AvatarStorageService
}

func NewUserService(
	userRepo repoInterface.UserRepository,
	sessionRepo repoInterface.SessionRepository,
	workspaceMemberRepo repoInterface.WorkspaceMemberRepository,
	avatarStorage _interface.AvatarStorageService,
) _interface.UserService {
	return &userService{
		userRepo:            userRepo,
		sessionRepo:         sessionRepo,
		workspaceMemberRepo: workspaceMemberRepo,
		avatarStorage:       avatarStorage,
	}
}

// BR-PROFILE-06: Only return safe fields, never password_hash or deleted_at
func (s *userService) GetProfile(userID string) (*dto.UserProfileResponse, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, apperror.ErrUnauthorized
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

// BR-PROFILE-01: Username policy enforcement
// BR-PROFILE-06: Safe fields only in response
func (s *userService) UpdateProfile(userID string, req *dto.UpdateProfileRequest) (*dto.UpdateProfileResponse, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, apperror.ErrUnauthorized
	}

	if req.FullName != nil {
		if strings.TrimSpace(*req.FullName) == "" {
			return nil, apperror.ErrFullNameRequired
		}
		user.FullName = *req.FullName
	}

	if req.Username != nil {
		if err := validator.ValidateUsername(*req.Username); err != nil {
			return nil, apperror.NewAppError(400, "VALIDATION_ERROR", err.Error())
		}
		existing, err := s.userRepo.GetByUsername(*req.Username)
		if err == nil && !strings.EqualFold(existing.Username, user.Username) {
			return nil, apperror.ErrUsernameAlreadyTaken
		}
		user.Username = *req.Username
	}

	updatedUser, err := s.userRepo.Update(userID, &models.User{
		FullName:  user.FullName,
		Username:  user.Username,
		UpdatedAt: time.Now(),
	})
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to update user")
	}

	return &dto.UpdateProfileResponse{
		ID:          updatedUser.ID,
		FullName:    updatedUser.FullName,
		Username:    updatedUser.Username,
		Email:       updatedUser.Email,
		PhoneNumber: updatedUser.PhoneNumber,
		AvatarURL:   updatedUser.AvatarURL,
		UpdatedAt:   updatedUser.UpdatedAt,
	}, nil
}

// BR-PROFILE-02: Avatar upload — resize 256x256, convert to PNG, upload to storage (S3 or local)
func (s *userService) UploadAvatar(userID string, fileHeader *multipart.FileHeader) (*dto.AvatarResponse, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return nil, apperror.ErrUploadFailed
	}
	defer file.Close()

	avatarURL, err := s.avatarStorage.Upload(userID, file, fileHeader)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, apperror.ErrUnauthorized
	}

	if user.AvatarURL != nil && *user.AvatarURL != "" {
		s.avatarStorage.Delete(*user.AvatarURL)
	}

	_, err = s.userRepo.Update(userID, &models.User{
		AvatarURL: &avatarURL,
		UpdatedAt: time.Now(),
	})
	if err != nil {
		return nil, apperror.ErrUploadFailed
	}

	return &dto.AvatarResponse{AvatarURL: &avatarURL}, nil
}

// BR-PROFILE-02: Remove avatar (fallback to frontend initials)
func (s *userService) RemoveAvatar(userID string) (*dto.AvatarResponse, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, apperror.ErrUnauthorized
	}

	if user.AvatarURL != nil && *user.AvatarURL != "" {
		s.avatarStorage.Delete(*user.AvatarURL)
	}

	_, err = s.userRepo.Update(userID, &models.User{
		AvatarURL: nil,
		UpdatedAt: time.Now(),
	})
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to remove avatar")
	}

	return &dto.AvatarResponse{AvatarURL: nil}, nil
}

// BR-PROFILE-03: Change password → revoke ALL sessions, issue new tokens, create new session
// BR-PROFILE-05: Enforce session limits after creating new session
func (s *userService) ChangePassword(userID string, req *dto.ChangePasswordRequest, userAgent string, ipAddress string) (*dto.ChangePasswordResponse, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, apperror.ErrUnauthorized
	}

	if user.AuthProvider != models.AuthProviderEmail {
		return nil, apperror.ErrOAuthAccountNoPassword
	}

	if user.PasswordHash == nil {
		return nil, apperror.ErrOAuthAccountNoPassword
	}

	if err := helper.VerifyPassword(req.CurrentPassword, *user.PasswordHash); err != nil {
		return nil, apperror.ErrWrongCurrentPassword
	}

	if !helper.CheckConfirmPassword(req.NewPassword, req.ConfirmPassword) {
		return nil, apperror.ErrPasswordMismatch
	}

	if helper.VerifyPassword(req.NewPassword, *user.PasswordHash) == nil {
		return nil, apperror.ErrSameAsCurrentPassword
	}

	if len(req.NewPassword) < 8 {
		return nil, apperror.ErrWeakPassword
	}

	hashedPassword, err := helper.HashPassword(req.NewPassword)
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to hash password")
	}

	_, err = s.userRepo.Update(userID, &models.User{
		PasswordHash: &hashedPassword,
		UpdatedAt:    time.Now(),
	})
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to update password")
	}

	// BR-PROFILE-03: Revoke ALL sessions (ALL, not just others)
	s.sessionRepo.RevokeAllExcept(userID, "")

	// BR-PROFILE-03: Issue new token pair
	accessToken, refreshToken, refreshTokenHash, err := helper.GenerateAccessAndRefreshTokens(userID, user.Email)
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to generate tokens")
	}

	// BR-PROFILE-03: Create new session record
	now := time.Now()
	session := &models.Session{
		UserID:       userID,
		RefreshToken: refreshTokenHash,
		UserAgent:    userAgent,
		IPAddress:    ipAddress,
		IsRevoked:    false,
		LastUsedAt:   &now,
		ExpiresAt:    now.Add(30 * 24 * time.Hour),
	}
	if err := s.sessionRepo.Create(session); err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to create session")
	}

	// BR-PROFILE-05: Enforce session limits based on plan
	if err := s.enforceSessionLimit(userID); err != nil {
		return nil, err
	}

	return &dto.ChangePasswordResponse{
		Message:      "Password changed successfully. Other sessions have been logged out.",
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// BR-PROFILE-04: Delete account — anonymize data, revoke sessions, soft delete
// Checks: confirmation → password (Email users) → workspace OWNER conflict → execute
func (s *userService) DeleteAccount(userID string, req *dto.DeleteAccountRequest) error {
	if req.Confirmation != "DELETE" {
		return apperror.ErrInvalidConfirmation
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return apperror.ErrUnauthorized
	}

	// Verify password for Email users
	if user.AuthProvider == models.AuthProviderEmail {
		if req.Password == nil || *req.Password == "" {
			return apperror.NewAppError(400, "VALIDATION_ERROR", "Password is required")
		}
		if user.PasswordHash == nil {
			return apperror.ErrOAuthAccountNoPassword
		}
		if err := helper.VerifyPassword(*req.Password, *user.PasswordHash); err != nil {
			return apperror.ErrWrongCurrentPassword
		}
	}

	// Check workspace OWNER conflict
	memberships, err := s.workspaceMemberRepo.ListByUserID(userID)
	if err == nil {
		for _, m := range memberships {
			if m.Role == models.WorkspaceRoleOWNER {
				return apperror.ErrWorkspaceOwnerConflict
			}
		}
	}

	// Revoke all sessions
	s.sessionRepo.RevokeAllExcept(userID, "")

	// Anonymize: full_name → "Deleted User", username → "deleted_{random}", avatar_url → null, is_active → false
	randomSuffix := uuid.New().String()[:8]
	anonymizedUsername := "deleted_" + randomSuffix
	now := time.Now()

	updates := map[string]any{
		"full_name":  "Deleted User",
		"username":   anonymizedUsername,
		"avatar_url": nil,
		"is_active":  false,
		"deleted_at": now,
		"updated_at": now,
	}

	return s.userRepo.AnonymizeDelete(userID, updates)
}

// BR-PROFILE-05: Enforce session limits.
// TODO: Inject WorkspaceRepository for plan-based limits (FREE=3, PRO=10, ENTERPRISE=unlimited).
// Currently defaults to 10 (PRO-level) as a safe limit.
func (s *userService) enforceSessionLimit(userID string) error {
	count, err := s.sessionRepo.CountActiveByUserID(userID)
	if err != nil {
		return nil
	}

	const defaultMaxSessions = 10
	if count > defaultMaxSessions {
		return s.sessionRepo.RevokeOldestByUserID(userID, defaultMaxSessions-1)
	}
	return nil
}

// compile-time interface check
var _ _interface.UserService = (*userService)(nil)
