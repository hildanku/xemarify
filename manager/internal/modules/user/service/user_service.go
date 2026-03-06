package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	auditDomain "github.com/hildanku/xemarify/internal/modules/audit/domain"
	auditService "github.com/hildanku/xemarify/internal/modules/audit/service"
	"github.com/hildanku/xemarify/internal/modules/user/domain"
	userRepo "github.com/hildanku/xemarify/internal/modules/user/repository"
	jwtpkg "github.com/hildanku/xemarify/pkg/jwt"
)

// Sentinel errors.
var ErrUserNotFound = errors.New("user not found")

// UserService handles CRUD operations for manager system users.
type UserService struct {
	db       *pgxpool.Pool
	userRepo userRepo.UserRepository
	auditSvc *auditService.AuditLogService
	log      *logrus.Logger
}

// NewUserService constructs the service with its required dependencies.
func NewUserService(
	db *pgxpool.Pool,
	userRepo userRepo.UserRepository,
	auditSvc *auditService.AuditLogService,
	log *logrus.Logger,
) *UserService {
	return &UserService{db: db, userRepo: userRepo, auditSvc: auditSvc, log: log}
}

// CreateUserInput holds the data required to create a new user.
type CreateUserInput struct {
	Username string
	Email    string
	Role     string
	Password string
	Avatar   *string
}

// Create atomically inserts a user and its hashed authentication record.
func (s *UserService) Create(ctx context.Context, in CreateUserInput, actor *jwtpkg.Claims, ip string) (*domain.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now().UTC()
	userID := uuid.New()

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx, `
		INSERT INTO users (id, username, email, role, avatar, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, userID, in.Username, in.Email, in.Role, in.Avatar, now)
	if err != nil {
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO authentications (id, user_id, password_hash, created_at)
		VALUES ($1, $2, $3, $4)
	`, uuid.New(), userID, string(hash), now)
	if err != nil {
		return nil, fmt.Errorf("failed to insert authentication: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	u := &domain.User{
		ID:        userID,
		Username:  in.Username,
		Email:     in.Email,
		Role:      domain.Role(in.Role),
		Avatar:    in.Avatar,
		CreatedAt: now,
	}

	s.auditSvc.Log(ctx, &auditDomain.AuditLog{
		UserID:         &actor.UserID,
		UserIdentifier: actor.Username,
		Action:         auditDomain.ActionCreateUser,
		ObjectType:     strPtr(auditDomain.ObjectTypeUser),
		ObjectID:       &userID,
		Metadata: map[string]interface{}{
			"created_username": in.Username,
			"created_role":     in.Role,
			"ip_address":       ip,
		},
	})

	return u, nil
}

// UpdateUserInput holds patchable user fields (empty string = no change).
type UpdateUserInput struct {
	Username string
	Email    string
	Role     string
	Avatar   *string
}

// Update applies a partial update to the user and records the changed fields.
func (s *UserService) Update(ctx context.Context, id uuid.UUID, in UpdateUserInput, actor *jwtpkg.Claims, ip string) (*domain.User, error) {
	existing, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("db error: %w", err)
	}
	if existing == nil {
		return nil, ErrUserNotFound
	}

	var changedFields []string

	if in.Username != "" && in.Username != existing.Username {
		changedFields = append(changedFields, "username")
		existing.Username = in.Username
	}
	if in.Email != "" && in.Email != existing.Email {
		changedFields = append(changedFields, "email")
		existing.Email = in.Email
	}
	if in.Role != "" && domain.Role(in.Role) != existing.Role {
		changedFields = append(changedFields, "role")
		existing.Role = domain.Role(in.Role)
	}
	if in.Avatar != nil {
		changedFields = append(changedFields, "avatar")
		existing.Avatar = in.Avatar
	}

	now := time.Now().UTC()
	existing.UpdatedAt = &now

	if err := s.userRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	s.auditSvc.Log(ctx, &auditDomain.AuditLog{
		UserID:         &actor.UserID,
		UserIdentifier: actor.Username,
		Action:         auditDomain.ActionUpdateUser,
		ObjectType:     strPtr(auditDomain.ObjectTypeUser),
		ObjectID:       &id,
		Metadata: map[string]interface{}{
			"changed_fields": changedFields,
			"ip_address":     ip,
		},
	})

	return existing, nil
}

// Delete removes a user by ID and records the deletion.
func (s *UserService) Delete(ctx context.Context, id uuid.UUID, actor *jwtpkg.Claims, ip string) error {
	existing, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("db error: %w", err)
	}
	if existing == nil {
		return ErrUserNotFound
	}

	if err := s.userRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	s.auditSvc.Log(ctx, &auditDomain.AuditLog{
		UserID:         &actor.UserID,
		UserIdentifier: actor.Username,
		Action:         auditDomain.ActionDeleteUser,
		ObjectType:     strPtr(auditDomain.ObjectTypeUser),
		ObjectID:       &id,
		Metadata: map[string]interface{}{
			"deleted_username": existing.Username,
			"ip_address":       ip,
		},
	})
	return nil
}

// GetByID returns a single user or ErrUserNotFound.
func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("db error: %w", err)
	}
	if u == nil {
		return nil, ErrUserNotFound
	}
	return u, nil
}

// List returns all users.
func (s *UserService) List(ctx context.Context) ([]*domain.User, error) {
	return s.userRepo.List(ctx)
}

func strPtr(s string) *string { return &s }
