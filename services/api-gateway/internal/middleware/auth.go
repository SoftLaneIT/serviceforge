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

// Package middleware provides HTTP middleware for the api-gateway.
package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/SoftLaneIT/serviceforge/packages/go-common/logger"
	"github.com/SoftLaneIT/serviceforge/packages/go-common/tenant"
)

// validateResponse is the expected payload from auth-service POST /v1/keys/validate.
type validateResponse struct {
	TenantID    string   `json:"tenantId"`
	KeyID       string   `json:"keyId"`
	Environment string   `json:"environment"`
	ModuleScope []string `json:"moduleScope"`
}

// AuthMiddleware validates the Bearer token in the Authorization header by
// calling auth-service.  On success it injects X-Tenant-ID into the request
// context and forwards the original request.  On failure it returns 401/403.
//
// authServiceURL is the base URL of the auth-service,
// e.g. "http://auth-service:8082".
func AuthMiddleware(authServiceURL string, log *slog.Logger) func(http.Handler) http.Handler {
	validateURL := strings.TrimRight(authServiceURL, "/") + "/v1/keys/validate"

	client := &http.Client{Timeout: 3 * time.Second}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawKey := bearerToken(r)
			if rawKey == "" {
				respondAuthError(w, http.StatusUnauthorized, "missing or malformed Authorization header")
				return
			}

			apiKey, err := validateKey(r.Context(), client, validateURL, rawKey)
			if err != nil {
				logger.FromContext(r.Context()).Warn("auth validation failed", slog.Any("error", err))
				respondAuthError(w, http.StatusUnauthorized, "invalid api key")
				return
			}

			// Inject the resolved tenant ID so downstream proxies can forward it.
			ctx := tenant.NewContext(r.Context(), apiKey.TenantID)
			r = r.WithContext(ctx)
			r.Header.Set("X-Tenant-ID", apiKey.TenantID)
			// Surface the key environment to downstream services.
			r.Header.Set("X-Key-Environment", apiKey.Environment)

			next.ServeHTTP(w, r)
		})
	}
}

// bearerToken extracts the token from an "Authorization: Bearer <token>" header.
// Returns "" if the header is absent or malformed.
func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return ""
	}
	t := strings.TrimSpace(h[len(prefix):])
	return t
}

// validateKey calls the auth-service validate endpoint and returns the
// resolved key metadata.
func validateKey(ctx context.Context, client *http.Client, validateURL, rawKey string) (*validateResponse, error) {
	body, err := json.Marshal(map[string]string{"key": rawKey})
	if err != nil {
		return nil, fmt.Errorf("marshal validate body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, validateURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build validate request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call auth-service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auth-service returned %d", resp.StatusCode)
	}

	var vr validateResponse
	if err := json.NewDecoder(resp.Body).Decode(&vr); err != nil {
		return nil, fmt.Errorf("decode validate response: %w", err)
	}
	return &vr, nil
}

func respondAuthError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
