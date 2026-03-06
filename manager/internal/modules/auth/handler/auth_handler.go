package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hildanku/xemarify/internal/infrastructure/middleware"
	authService "github.com/hildanku/xemarify/internal/modules/auth/service"
	"github.com/hildanku/xemarify/internal/modules/auth/transport"
	"github.com/hildanku/xemarify/pkg/response"
	"github.com/sirupsen/logrus"
)

// AuthHandler handles HTTP requests for authentication endpoints.
type AuthHandler struct {
	svc *authService.AuthService
	log *logrus.Logger
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(svc *authService.AuthService, log *logrus.Logger) *AuthHandler {
	return &AuthHandler{svc: svc, log: log}
}

// Register wires the public auth routes (login, refresh) onto the given router group.
// Call RegisterProtected for routes that require JWT authentication.
func (h *AuthHandler) Register(rg *gin.RouterGroup) {
	rg.POST("/login", h.Login)
	rg.POST("/refresh", h.Refresh)
}

// RegisterProtected wires the authenticated auth routes (logout) onto the given router group.
// The group must already have UserAuth middleware applied.
func (h *AuthHandler) RegisterProtected(rg *gin.RouterGroup) {
	rg.POST("/logout", h.Logout)
}

// Login handles POST /auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req transport.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Write(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	result, err := h.svc.Login(
		c.Request.Context(),
		req.Email,
		req.Password,
		c.ClientIP(),
		c.GetHeader("User-Agent"),
	)
	if err != nil {
		if errors.Is(err, authService.ErrInvalidCredentials) {
			response.Write(c, http.StatusUnauthorized, "invalid email or password", nil)
			return
		}
		h.log.WithError(err).Error("login failed")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	response.Write(c, http.StatusOK, "Login successful", gin.H{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
	})
}

// Refresh handles POST /auth/refresh.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req transport.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Write(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	accessToken, newRefresh, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, authService.ErrInvalidRefreshToken) {
			response.Write(c, http.StatusUnauthorized, "invalid or expired refresh token", nil)
			return
		}
		h.log.WithError(err).Error("refresh failed")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	response.Write(c, http.StatusOK, "Token refreshed", gin.H{
		"access_token":  accessToken,
		"refresh_token": newRefresh,
	})
}

// Logout handles POST /auth/logout (requires JWT middleware).
func (h *AuthHandler) Logout(c *gin.Context) {
	claims := middleware.UserClaimsFromContext(c)
	if claims == nil {
		response.Write(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	if err := h.svc.Logout(c.Request.Context(), claims.UserID, claims.Username, c.ClientIP()); err != nil {
		h.log.WithError(err).Error("logout failed")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	response.Write(c, http.StatusOK, "Logged out successfully", nil)
}
