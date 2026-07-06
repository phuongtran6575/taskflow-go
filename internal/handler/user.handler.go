package handler

import (
	"TaskFlow-Go/internal/dto"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
	"TaskFlow-Go/internal/shared/appresponse"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService    _interface.UserService
	sessionService _interface.SessionService
}

func NewUserHandler(userService _interface.UserService, sessionService _interface.SessionService) *UserHandler {
	return &UserHandler{userService: userService, sessionService: sessionService}
}

// GetProfile
// @Summary      Get user profile
// @Description  Get the current authenticated user's profile
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /users/me [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := c.GetString("user_id")
	result, err := h.userService.GetProfile(userID)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			appresponse.Fail(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		appresponse.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	appresponse.OK(c, result)
}

// UpdateProfile
// @Summary      Update user profile
// @Description  Update the current user's profile information
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.UpdateProfileRequest true "Profile update data"
// @Success      200  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /users/me [patch]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetString("user_id")
	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.userService.UpdateProfile(userID, &req)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			appresponse.Fail(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		appresponse.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	appresponse.OK(c, result)
}

// UploadAvatar
// @Summary      Upload avatar
// @Description  Upload a new avatar image for the current user
// @Tags         Users
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        file formData file true "Avatar image file"
// @Success      200  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /users/me/avatar [post]
func (h *UserHandler) UploadAvatar(c *gin.Context) {
	userID := c.GetString("user_id")
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "File is required")
		return
	}
	defer file.Close()
	result, err := h.userService.UploadAvatar(userID, header)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			appresponse.Fail(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		appresponse.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	appresponse.OK(c, result)
}

// RemoveAvatar
// @Summary      Remove avatar
// @Description  Remove the current user's avatar
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /users/me/avatar [delete]
func (h *UserHandler) RemoveAvatar(c *gin.Context) {
	userID := c.GetString("user_id")
	result, err := h.userService.RemoveAvatar(userID)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			appresponse.Fail(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		appresponse.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	appresponse.OK(c, result)
}

// ChangePassword
// @Summary      Change password
// @Description  Change the current user's password
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.ChangePasswordRequest true "Password change data"
// @Success      200  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /users/me/password [put]
func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID := c.GetString("user_id")
	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	userAgent := c.GetHeader("User-Agent")
	ipAddress := c.ClientIP()
	result, err := h.userService.ChangePassword(userID, &req, userAgent, ipAddress)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			appresponse.Fail(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		appresponse.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	appresponse.OK(c, result)
}

// GetSessions
// @Summary      List user sessions
// @Description  Get all active sessions for the current user
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /users/me/sessions [get]
func (h *UserHandler) GetSessions(c *gin.Context) {
	userID := c.GetString("user_id")
	result, err := h.sessionService.GetSessionsByUserId(userID)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			appresponse.Fail(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		appresponse.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	appresponse.OK(c, result)
}

// RevokeSession
// @Summary      Revoke a session
// @Description  Revoke a specific session by ID
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        session_id path string true "Session ID"
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Failure      404  {object}  appresponse.Response
// @Router       /users/me/sessions/{session_id} [delete]
func (h *UserHandler) RevokeSession(c *gin.Context) {
	userID := c.GetString("user_id")
	sessionID := c.Param("session_id")
	err := h.sessionService.RevokeSession(userID, sessionID)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			appresponse.Fail(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		appresponse.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	appresponse.OK(c, dto.RevokeSessionResponse{Message: "Session revoked successfully."})
}

// RevokeAllOtherSessions
// @Summary      Revoke all other sessions
// @Description  Revoke all sessions except the current one
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /users/me/sessions [delete]
func (h *UserHandler) RevokeAllOtherSessions(c *gin.Context) {
	userID := c.GetString("user_id")
	result, err := h.sessionService.RevokeAllOtherSessions(userID)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			appresponse.Fail(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		appresponse.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	appresponse.OK(c, result)
}

// DeleteAccount
// @Summary      Delete account
// @Description  Permanently delete the current user's account
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.DeleteAccountRequest true "Account deletion confirmation"
// @Success      200  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /users/me [delete]
func (h *UserHandler) DeleteAccount(c *gin.Context) {
	userID := c.GetString("user_id")
	var req dto.DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	err := h.userService.DeleteAccount(userID, &req)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			appresponse.Fail(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		appresponse.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	appresponse.OK(c, dto.DeleteAccountResponse{Message: "Your account has been deleted successfully."})
}
