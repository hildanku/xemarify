package transport

// InitializeRequest is the JSON body for POST /setup/initialize.
type InitializeRequest struct {
	Username   string `json:"username"    binding:"required,min=3,max=50"`
	Email      string `json:"email"       binding:"required,email"`
	Password   string `json:"password"    binding:"required,min=8"`
	SetupToken string `json:"setup_token" binding:"required"`
}
