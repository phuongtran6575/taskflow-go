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

type AuthHandler struct {
	authService _interface.AuthService
}

func NewAuthHandler(authService _interface.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Register
// @Summary      Register a new user
// @Description  Create a new user account with email and password
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body dto.RegisterRequest true "Registration details"
// @Success      201  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      409  {object}  appresponse.Response
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.authService.Register(&req)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			appresponse.Fail(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		appresponse.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	appresponse.Created(c, result)
}

// Login
// @Summary      Login user
// @Description  Authenticate user with email and password, returns JWT tokens
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body dto.LoginRequest true "Login credentials"
// @Success      200  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.authService.Login(&req)
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

// Logout
// @Summary      Logout user
// @Description  Invalidate refresh token to log out
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.LogoutRequest true "Refresh token to invalidate"
// @Success      200  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	userID := c.GetString("user_id")
	var req dto.LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	err := h.authService.Logout(userID, &req)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			appresponse.Fail(c, appErr.Status, appErr.Code, appErr.Message)
			return
		}
		appresponse.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	appresponse.OK(c, dto.LogoutResponse{Message: "Logged out successfully"})
}

// RefreshToken
// @Summary      Refresh access token
// @Description  Get a new access token using a valid refresh token
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body dto.RefreshTokenRequest true "Refresh token"
// @Success      200  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.authService.RefreshToken(&req)
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

// ForgotPassword
// @Summary      Request password reset
// @Description  Send password reset email to the given email address
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body dto.ForgotPasswordRequest true "Registered email"
// @Success      200  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Router       /auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.authService.ForgotPassword(&req)
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

// ResetPassword
// @Summary      Reset password
// @Description  Reset password using the token from the reset email
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body dto.ResetPasswordRequest true "Reset token and new password"
// @Success      200  {object}  appresponse.Response
// @Failure      400  {object}  appresponse.Response
// @Failure      401  {object}  appresponse.Response
// @Router       /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.authService.ResetPassword(&req)
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

// InitiateGoogleOAuth
// @Summary      Initiate Google OAuth login
// @Description  Redirect user to Google OAuth consent screen
// @Tags         Auth
// @Produce      json
// @Success      302
// @Router       /auth/oauth/google [get]
func (h *AuthHandler) InitiateGoogleOAuth(c *gin.Context) {
	url, err := h.authService.GetGoogleOAuthUrl()
	if err != nil {
		appresponse.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	c.Redirect(http.StatusFound, url)
}

// HandleGoogleCallback
// @Summary      Google OAuth callback
// @Description  Handle Google OAuth callback and return JWT tokens
// @Tags         Auth
// @Produce      json
// @Param        code query string true "Authorization code from Google"
// @Success      302
// @Failure      400  {object}  appresponse.Response
// @Router       /auth/oauth/google/callback [get]
func (h *AuthHandler) HandleGoogleCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "Authorization code is required")
		return
	}
	result, err := h.authService.ProcessGoogleCallback(code)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			c.Redirect(http.StatusFound, "/oauth/error?code="+appErr.Code)
			return
		}
		c.Redirect(http.StatusFound, "/oauth/error?code=INTERNAL_ERROR")
		return
	}
	c.Redirect(http.StatusFound, "/oauth/success?access_token="+result.AccessToken+"&refresh_token="+result.RefreshToken)
}

// InitiateGithubOAuth
// @Summary      Initiate GitHub OAuth login
// @Description  Redirect user to GitHub OAuth consent screen
// @Tags         Auth
// @Produce      json
// @Success      302
// @Router       /auth/oauth/github [get]
func (h *AuthHandler) InitiateGithubOAuth(c *gin.Context) {
	url, err := h.authService.GetGithubOAuthUrl()
	if err != nil {
		appresponse.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	c.Redirect(http.StatusFound, url)
}

// HandleGithubCallback
// @Summary      GitHub OAuth callback
// @Description  Handle GitHub OAuth callback and return JWT tokens
// @Tags         Auth
// @Produce      json
// @Param        code query string true "Authorization code from GitHub"
// @Success      302
// @Failure      400  {object}  appresponse.Response
// @Router       /auth/oauth/github/callback [get]
func (h *AuthHandler) HandleGithubCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "Authorization code is required")
		return
	}
	result, err := h.authService.ProcessGithubCallback(code)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			c.Redirect(http.StatusFound, "/oauth/error?code="+appErr.Code)
			return
		}
		c.Redirect(http.StatusFound, "/oauth/error?code=INTERNAL_ERROR")
		return
	}
	c.Redirect(http.StatusFound, "/oauth/success?access_token="+result.AccessToken+"&refresh_token="+result.RefreshToken)
}
