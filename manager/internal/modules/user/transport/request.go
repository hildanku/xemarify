package transport

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
