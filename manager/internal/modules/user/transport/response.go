package transport

import (
	"time"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/user/domain"
)

// UserResponse is the JSON representation of a user returned to the client.
type UserResponse struct {
	ID        uuid.UUID  `json:"id"`
	Username  string     `json:"username"`
	Email     string     `json:"email"`
	Role      string     `json:"role"`
	Avatar    *string    `json:"avatar,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

// ListUsersMetadata carries pagination and count information for a list response.
type ListUsersMetadata struct {
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
	Limit      int `json:"limit"`
	Offset     int `json:"offset"`
}

// ListUsersResponse wraps a paginated list of users with metadata.
type ListUsersResponse struct {
	Items    []*UserResponse   `json:"items"`
	Metadata ListUsersMetadata `json:"metadata"`
}

// ToUserResponse converts a domain User to its HTTP response representation.
func ToUserResponse(u *domain.User) *UserResponse {
	return &UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Role:      string(u.Role),
		Avatar:    u.Avatar,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
