package repository

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// PageCursor is the composite pagination token used for keyset pagination.
// It encodes the last-seen (received_at, id) pair so the next query can
// resume deterministically without an OFFSET scan on the partitioned table.
//
// Both fields are required to guarantee a unique, stable position because
// received_at alone is not unique (multiple events can share the same timestamp).
type PageCursor struct {
	ReceivedAt time.Time `json:"r"`
	ID         uuid.UUID `json:"i"`
}

// EncodeCursor serialises c to a compact, URL-safe base64 (no padding) string.
// The encoding is opaque to callers; only DecodeCursor should parse it.
func EncodeCursor(c PageCursor) string {
	b, _ := json.Marshal(c) // PageCursor is always marshallable.
	return base64.RawURLEncoding.EncodeToString(b)
}

// DecodeCursor parses a cursor string produced by EncodeCursor.
// An empty string is valid and signals the first page (no prior position).
// Returns an error for any other malformed input.
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
	if c.ReceivedAt.IsZero() {
		return PageCursor{}, fmt.Errorf("cursor: missing received_at field")
	}
	if c.ID == uuid.Nil {
		return PageCursor{}, fmt.Errorf("cursor: missing id field")
	}
	return c, nil
}
