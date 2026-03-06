package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	userDomain "github.com/hildanku/xemarify/internal/modules/user/domain"
	"github.com/hildanku/xemarify/pkg/response"
)

// RequireRole returns a Gin middleware that verifies the authenticated user
// holds one of the allowed roles. Must be placed after UserAuth middleware.
func RequireRole(roles ...userDomain.Role) gin.HandlerFunc {
	allowed := make(map[userDomain.Role]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}

	return func(c *gin.Context) {
		claims := UserClaimsFromContext(c)
		if claims == nil {
			response.WriteWithAbort(c, http.StatusUnauthorized, "unauthorized", nil)
			return
		}

		if _, ok := allowed[claims.Role]; !ok {
			response.WriteWithAbort(c, http.StatusForbidden, "insufficient permissions", nil)
			return
		}

		c.Next()
	}
}
