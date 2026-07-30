package auth

import (
	"context"
	"net/http"
	"strings"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization := r.Header.Get("Authorization")
		parts := strings.SplitN(authorization, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" || strings.TrimSpace(parts[1]) == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		claims, err := NewService().Validate(parts[1])
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), ctxKey{}, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type ctxKey struct{}

func FromContext(ctx context.Context) map[string]any {
	if claims, ok := ctx.Value(ctxKey{}).(map[string]any); ok {
		return claims
	}
	return nil
}
