package repository

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type PageCursor struct {
	CreatedAt time.Time `json:"c"`
	ID        uuid.UUID `json:"i"`
}

func EncodeCursor(c PageCursor) string {
	b, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(b)
}

func DecodeCursor(s string) (PageCursor, error) {
	if s == "" {
		return PageCursor{}, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return PageCursor{}, fmt.Errorf("cursor: invalid base64: %w", err)
	}
	var c PageCursor
	if err := json.Unmarshal(b, &c); err != nil {
		return PageCursor{}, fmt.Errorf("cursor: invalid payload: %w", err)
	}
	if c.CreatedAt.IsZero() {
		return PageCursor{}, fmt.Errorf("cursor: missing created_at field")
	}
	if c.ID == uuid.Nil {
		return PageCursor{}, fmt.Errorf("cursor: missing id field")
	}
	return c, nil
}