package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hildanku/xemarify/config"
	jwtpkg "github.com/hildanku/xemarify/pkg/jwt"
	"github.com/hildanku/xemarify/pkg/response"
)

const UserClaimsKey = "user_claims"

// UserAuth returns a Gin middleware that validates a Bearer JWT access token.
// On success it stores *jwtpkg.Claims in the context under UserClaimsKey.
func UserAuth(jwtCfg config.JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			response.WriteWithAbort(c, http.StatusUnauthorized, "missing or invalid Authorization header", nil)
			return
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")

		claims, err := jwtpkg.ValidateAccessToken(tokenStr, jwtCfg.Secret)
		if err != nil {
			response.WriteWithAbort(c, http.StatusUnauthorized, "invalid or expired token", nil)
			return
		}

		c.Set(UserClaimsKey, claims)
		c.Next()
	}
}

// UserClaimsFromContext retrieves the JWT claims stored by UserAuth middleware.
// Returns nil if the middleware was not applied or authentication failed.
func UserClaimsFromContext(c *gin.Context) *jwtpkg.Claims {
	val, exists := c.Get(UserClaimsKey)
	if !exists {
		return nil
	}
	claims, _ := val.(*jwtpkg.Claims)
	return claims
}
