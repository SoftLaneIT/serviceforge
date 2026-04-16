/*
 * Copyright (c) 2026, SoftlaneIT (https://softlaneit.com/) All Rights Reserved.
 *
 * SoftlaneIT licenses this file to you under the Apache License,
 * Version 2.0 (the "LICENSE"); you may not use this file except
 * in compliance with the LICENSE.
 * You may obtain a copy of the LICENSE at
 *
 * https://softlaneit.com/LICENSE.txt
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the LICENSE is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the LICENSE for the
 * specific language governing permissions and limitations
 * under the LICENSE.
 */

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

	// DefaultTenant is the sentinel value returned by FromContext when no
	// X-Tenant-ID header was present on the request.  Handlers that require a
	// real tenant ID should compare against this value to detect unauthenticated
	// calls, e.g.:
	//
	//	if tid := tenant.FromContext(ctx); tid == tenant.DefaultTenant { ... }
	DefaultTenant = defaultTenantName
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

// NewContext returns a copy of ctx with tenantID stored under the tenant key.
// Useful in tests and internal callers that construct contexts directly instead
// of going through Middleware.
func NewContext(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantContextKey, tenantID)
}
