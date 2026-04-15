package tenant

import (
	"context"
	"net/http"
)

type ctxKey string

const (
	tenantHeader      = "X-Tenant-ID"
	tenantContextKey  = ctxKey("tenant_id")
	defaultTenantName = "default"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.Header.Get(tenantHeader)
		if tenantID == "" {
			tenantID = defaultTenantName
		}
		ctx := context.WithValue(r.Context(), tenantContextKey, tenantID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func FromContext(ctx context.Context) string {
	if value, ok := ctx.Value(tenantContextKey).(string); ok && value != "" {
		return value
	}
	return defaultTenantName
}
