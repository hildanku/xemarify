package transport

// ListUsersQuery holds the query parameters for GET /api/v1/users.
type ListUsersQuery struct {
	// Search performs a case-insensitive partial match on username and email.
	Search string `form:"search"`

	// SortBy is the column to sort by: username, email, role, created_at.
	SortBy string `form:"sort_by,default=created_at"`

	// Order is the sort direction: asc or desc.
	Order string `form:"order,default=asc" binding:"omitempty,oneof=asc desc"`

	// Limit is the maximum number of rows to return (1-100).
	Limit int `form:"limit,default=10" binding:"omitempty,min=1,max=100"`

	// Offset is the number of rows to skip.
	Offset int `form:"offset,default=0" binding:"omitempty,min=0"`
}

// CreateUserRequest is the JSON body for POST /api/v1/users.
type CreateUserRequest struct {
	Username string  `json:"username" binding:"required,min=3,max=50"`
	Email    string  `json:"email"    binding:"required,email"`
	Role     string  `json:"role"     binding:"required,oneof=MANAGER ANALYST VIEWER"`
	Password string  `json:"password" binding:"required,min=8"`
	Avatar   *string `json:"avatar"`
}

// UpdateUserRequest is the JSON body for PUT /api/v1/users/:id.
// All fields are optional (omitempty) — only provided fields are applied.
type UpdateUserRequest struct {
	Username string  `json:"username" binding:"omitempty,min=3,max=50"`
	Email    string  `json:"email"    binding:"omitempty,email"`
	Role     string  `json:"role"     binding:"omitempty,oneof=MANAGER ANALYST VIEWER"`
	Avatar   *string `json:"avatar"`
}
