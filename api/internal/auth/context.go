package auth

import (
	"context"

	"github.com/google/uuid"
)

type contextKey string

const userIDKey contextKey = "user_id"

// WithUserID returns a new context carrying the user ID.
func WithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserIDFromContext extracts the user ID from the context.
// Returns uuid.Nil and false if not present.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDKey).(uuid.UUID)
	return id, ok
}
