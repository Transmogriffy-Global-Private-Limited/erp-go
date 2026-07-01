package auth

import "context"

type platformContextKey string

const platformUserIDKey platformContextKey = "platform_user_id"

// WithPlatformUserID stores the authenticated platform user ID in request context.
//
// This is currently backed by the temporary X-Platform-User-ID header.
// Later, it must come from real platform authentication/session state.
func WithPlatformUserID(ctx context.Context, platformUserID string) context.Context {
	return context.WithValue(ctx, platformUserIDKey, platformUserID)
}

// PlatformUserIDFromContext reads the authenticated platform user ID from request context.
func PlatformUserIDFromContext(ctx context.Context) (string, bool) {
	platformUserID, ok := ctx.Value(platformUserIDKey).(string)
	return platformUserID, ok && platformUserID != ""
}
