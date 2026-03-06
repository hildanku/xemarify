package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	userDomain "github.com/hildanku/xemarify/internal/modules/user/domain"
)

// Claims is the JWT payload for authenticated manager users.
type Claims struct {
	UserID   uuid.UUID       `json:"user_id"`
	Username string          `json:"username"`
	Role     userDomain.Role `json:"role"`
	jwt.RegisteredClaims
}

// GenerateAccessToken mints a signed HS256 JWT access token.
func GenerateAccessToken(userID uuid.UUID, username string, role userDomain.Role, secret string, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}

// GenerateRefreshToken returns a random opaque UUID string used as a refresh token.
func GenerateRefreshToken() string {
	return uuid.New().String()
}

// ValidateAccessToken parses and validates an access token, returning its Claims.
func ValidateAccessToken(tokenStr, secret string) (*Claims, error) {
	t, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := t.Claims.(*Claims)
	if !ok || !t.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
