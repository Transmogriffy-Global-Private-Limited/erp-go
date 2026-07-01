package tenancy

import "context"

type contextKey string

const tenantIDKey contextKey = "tenant_id"

func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantIDKey, tenantID)
}

func TenantIDFromContext(ctx context.Context) (string, bool) {
	tenantID, ok := ctx.Value(tenantIDKey).(string)
	return tenantID, ok && tenantID != ""
}
