package domain

import (
	"time"

	"github.com/google/uuid"
)

// Action constants for common system events.
const (
	ActionLogin      = "LOGIN"
	ActionLogout     = "LOGOUT"
	ActionCreateUser = "CREATE_USER"
	ActionUpdateUser = "UPDATE_USER"
	ActionDeleteUser = "DELETE_USER"

	ObjectTypeUser = "USER"
)

// AuditLog is the internal domain representation of an audit trail entry.
type AuditLog struct {
	ID             uuid.UUID
	UserID         *uuid.UUID
	UserIdentifier string
	Action         string
	ObjectType     *string
	ObjectID       *uuid.UUID
	Metadata       map[string]interface{}
	CreatedAt      time.Time
}
