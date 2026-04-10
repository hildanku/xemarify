package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/config"
	userDomain "github.com/hildanku/xemarify/internal/modules/user/domain"
	jwtpkg "github.com/hildanku/xemarify/pkg/jwt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

const bootstrapAdvisoryLockKey int64 = 2026041001

var (
	ErrAlreadyInitialized    = errors.New("system already initialized")
	ErrInvalidSetupToken     = errors.New("invalid setup token")
	ErrSetupTokenUnavailable = errors.New("setup token is not configured")
)

// InitializeResult returns the session minted for the first manager.
type InitializeResult struct {
	AccessToken  string
	RefreshToken string
}

// SetupService handles first-run bootstrap for the initial manager account.
type SetupService struct {
	db         *pgxpool.Pool
	jwtCfg     config.JWTConfig
	setupToken string
	log        *logrus.Logger
}

func NewSetupService(db *pgxpool.Pool, jwtCfg config.JWTConfig, setupToken string, log *logrus.Logger) *SetupService {
	return &SetupService{
		db:         db,
		jwtCfg:     jwtCfg,
		setupToken: setupToken,
		log:        log,
	}
}

// IsInitialized reports whether at least one manager account already exists.
func (s *SetupService) IsInitialized(ctx context.Context) (bool, error) {
	const q = `
		SELECT EXISTS (
			SELECT 1
			FROM users
			WHERE role = $1
		)
	`

	var initialized bool
	if err := s.db.QueryRow(ctx, q, userDomain.RoleManager).Scan(&initialized); err != nil {
		return false, fmt.Errorf("failed to query setup status: %w", err)
	}

	return initialized, nil
}

// InitializeFirstManager creates the initial manager account exactly once.
func (s *SetupService) InitializeFirstManager(ctx context.Context, username, email, password, setupToken string) (*InitializeResult, error) {
	if s.setupToken == "" {
		return nil, ErrSetupTokenUnavailable
	}
	if setupToken != s.setupToken {
		return nil, ErrInvalidSetupToken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now().UTC()
	userID := uuid.New()
	refreshToken := jwtpkg.GenerateRefreshToken()

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin bootstrap transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, bootstrapAdvisoryLockKey); err != nil {
		return nil, fmt.Errorf("failed to acquire bootstrap lock: %w", err)
	}

	const existsQ = `
		SELECT EXISTS (
			SELECT 1
			FROM users
			WHERE role = $1
		)
	`

	var initialized bool
	if err := tx.QueryRow(ctx, existsQ, userDomain.RoleManager).Scan(&initialized); err != nil {
		return nil, fmt.Errorf("failed to re-check setup status: %w", err)
	}
	if initialized {
		return nil, ErrAlreadyInitialized
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO users (id, username, email, role, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, username, email, userDomain.RoleManager, now); err != nil {
		return nil, fmt.Errorf("failed to insert initial manager: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO authentications (id, user_id, password_hash, refresh_token, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, uuid.New(), userID, string(hash), refreshToken, now); err != nil {
		return nil, fmt.Errorf("failed to insert initial manager authentication: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit bootstrap transaction: %w", err)
	}

	accessToken, err := jwtpkg.GenerateAccessToken(userID, username, userDomain.RoleManager, s.jwtCfg.Secret, s.jwtCfg.AccessTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	s.log.WithFields(logrus.Fields{
		"user_id": userID,
		"email":   email,
	}).Info("initial manager bootstrapped")

	return &InitializeResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
