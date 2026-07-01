package auth

import "context"

type contextKey string

const userIDKey contextKey = "user_id"

// WithUserID stores the authenticated user ID in request context.
//
// This is currently backed by the temporary X-User-ID header.
// Later, it must come from real authentication/session/JWT state.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserIDFromContext reads the authenticated user ID from request context.
func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDKey).(string)
	return userID, ok && userID != ""
}
