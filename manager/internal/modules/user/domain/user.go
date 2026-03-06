package domain

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleManager Role = "MANAGER"
	RoleAnalyst Role = "ANALYST"
	RoleViewer  Role = "VIEWER"
)

// User is the internal domain representation of a manager system user.
type User struct {
	ID        uuid.UUID
	Username  string
	Email     string
	Role      Role
	Avatar    *string
	CreatedAt time.Time
	UpdatedAt *time.Time
}
