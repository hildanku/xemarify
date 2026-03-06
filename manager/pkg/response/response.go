package response

import (
	"github.com/gin-gonic/gin"
)

type JSONResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Write is a helper that writes a JSON response with a consistent structure.
func Write(c *gin.Context, code int, message string, data any) {
	c.JSON(code, JSONResponse{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

// WriteWithAbort is a helper that writes a JSON response and then aborts the request.
func WriteWithAbort(c *gin.Context, code int, message string, data any) {
	Write(c, code, message, data)
	c.Abort()
}
