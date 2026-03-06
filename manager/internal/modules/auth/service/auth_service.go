package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/config"
	auditDomain "github.com/hildanku/xemarify/internal/modules/audit/domain"
	auditService "github.com/hildanku/xemarify/internal/modules/audit/service"
	authRepo "github.com/hildanku/xemarify/internal/modules/auth/repository"
	userRepo "github.com/hildanku/xemarify/internal/modules/user/repository"
	jwtpkg "github.com/hildanku/xemarify/pkg/jwt"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// Sentinel errors.
var (
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
)

// AuthService handles login, token refresh, and logout.
type AuthService struct {
	userRepo userRepo.UserRepository
	authRepo authRepo.AuthRepository
	auditSvc *auditService.AuditLogService
	jwtCfg   config.JWTConfig
	log      *logrus.Logger
}

// NewAuthService constructs the service with its required dependencies.
func NewAuthService(
	userRepo userRepo.UserRepository,
	authRepo authRepo.AuthRepository,
	auditSvc *auditService.AuditLogService,
	jwtCfg config.JWTConfig,
	log *logrus.Logger,
) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		authRepo: authRepo,
		auditSvc: auditSvc,
		jwtCfg:   jwtCfg,
		log:      log,
	}
}

// LoginResult holds the tokens returned after a successful login.
type LoginResult struct {
	AccessToken  string
	RefreshToken string
}

// Login authenticates the user and returns a JWT access token and a refresh token.
// It logs the LOGIN action to the audit trail.
func (s *AuthService) Login(ctx context.Context, email, password, ipAddress, userAgent string) (*LoginResult, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("user lookup failed: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	auth, err := s.authRepo.GetByUserID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("auth lookup failed: %w", err)
	}
	if auth == nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(auth.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := s.authRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		s.log.WithError(err).WithField("user_id", user.ID).Warn("failed to update last_login_at")
	}

	accessToken, err := jwtpkg.GenerateAccessToken(user.ID, user.Username, user.Role, s.jwtCfg.Secret, s.jwtCfg.AccessTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken := jwtpkg.GenerateRefreshToken()
	if err := s.authRepo.SetRefreshToken(ctx, user.ID, &refreshToken); err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	s.auditSvc.Log(ctx, &auditDomain.AuditLog{
		UserID:         &user.ID,
		UserIdentifier: user.Email,
		Action:         auditDomain.ActionLogin,
		ObjectType:     strPtr(auditDomain.ObjectTypeUser),
		ObjectID:       &user.ID,
		Metadata: map[string]interface{}{
			"ip_address": ipAddress,
			"user_agent": userAgent,
		},
	})

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// Refresh validates the given refresh token, issues a new access token, and
// rotates the refresh token (invalidating the previous one).
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (string, string, error) {
	auth, err := s.authRepo.GetByRefreshToken(ctx, refreshToken)
	if err != nil {
		return "", "", fmt.Errorf("auth lookup failed: %w", err)
	}
	if auth == nil {
		return "", "", ErrInvalidRefreshToken
	}

	user, err := s.userRepo.GetByID(ctx, auth.UserID)
	if err != nil || user == nil {
		return "", "", ErrInvalidRefreshToken
	}

	accessToken, err := jwtpkg.GenerateAccessToken(user.ID, user.Username, user.Role, s.jwtCfg.Secret, s.jwtCfg.AccessTokenTTL)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	// Rotate refresh token
	newRefresh := jwtpkg.GenerateRefreshToken()
	if err := s.authRepo.SetRefreshToken(ctx, user.ID, &newRefresh); err != nil {
		return "", "", fmt.Errorf("failed to rotate refresh token: %w", err)
	}

	return accessToken, newRefresh, nil
}

// Logout clears the stored refresh token and writes a LOGOUT audit entry.
func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID, userIdentifier, ipAddress string) error {
	if err := s.authRepo.SetRefreshToken(ctx, userID, nil); err != nil {
		return fmt.Errorf("failed to clear refresh token: %w", err)
	}

	s.auditSvc.Log(ctx, &auditDomain.AuditLog{
		UserID:         &userID,
		UserIdentifier: userIdentifier,
		Action:         auditDomain.ActionLogout,
		ObjectType:     strPtr(auditDomain.ObjectTypeUser),
		ObjectID:       &userID,
		Metadata: map[string]interface{}{
			"ip_address": ipAddress,
		},
	})
	return nil
}

func strPtr(s string) *string { return &s }
