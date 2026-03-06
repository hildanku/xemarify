package transport

import (
	"time"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/audit/domain"
)

// AuditLogResponse is the JSON representation of an audit log entry.
type AuditLogResponse struct {
	ID             uuid.UUID              `json:"id"`
	UserID         *uuid.UUID             `json:"user_id,omitempty"`
	UserIdentifier string                 `json:"user_identifier"`
	Action         string                 `json:"action"`
	ObjectType     *string                `json:"object_type,omitempty"`
	ObjectID       *uuid.UUID             `json:"object_id,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
}

// AuditLogListResponse wraps a paginated list of audit log entries.
type AuditLogListResponse struct {
	Items    []*AuditLogResponse `json:"items"`
	Total    int                 `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
}

// ToAuditLogResponse converts a domain AuditLog to its HTTP response form.
func ToAuditLogResponse(e *domain.AuditLog) *AuditLogResponse {
	return &AuditLogResponse{
		ID:             e.ID,
		UserID:         e.UserID,
		UserIdentifier: e.UserIdentifier,
		Action:         e.Action,
		ObjectType:     e.ObjectType,
		ObjectID:       e.ObjectID,
		Metadata:       e.Metadata,
		CreatedAt:      e.CreatedAt,
	}
}
