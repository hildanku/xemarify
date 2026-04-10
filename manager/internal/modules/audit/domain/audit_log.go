package domain

import (
	"time"

	"github.com/google/uuid"
)

// Action constants for common system events.
const (
	ActionLogin                 = "LOGIN"
	ActionLogout                = "LOGOUT"
	ActionCreateUser            = "CREATE_USER"
	ActionUpdateUser            = "UPDATE_USER"
	ActionDeleteUser            = "DELETE_USER"
	ActionCreateRule            = "CREATE_RULE"
	ActionUpdateRule            = "UPDATE_RULE"
	ActionDeleteRule            = "DELETE_RULE"
	ActionCreateAgent           = "CREATE_AGENT"
	ActionRegisterAgent         = "REGISTER_AGENT"
	ActionUpdateAgent           = "UPDATE_AGENT"
	ActionDeleteAgent           = "DELETE_AGENT"
	ActionUpdateAlertStatus     = "UPDATE_ALERT_STATUS"
	ActionGenerateEnrollmentKey = "GENERATE_ENROLLMENT_KEY"

	ObjectTypeUser          = "USER"
	ObjectTypeRule          = "RULE"
	ObjectTypeAgent         = "AGENT"
	ObjectTypeAlert         = "ALERT"
	ObjectTypeEnrollmentKey = "ENROLLMENT_KEY"
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
