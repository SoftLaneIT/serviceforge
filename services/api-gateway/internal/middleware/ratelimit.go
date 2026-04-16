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

package middleware

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/SoftLaneIT/serviceforge/packages/go-common/tenant"
)

// windowState tracks request counts within a fixed time window.
type windowState struct {
	mu          sync.Mutex
	count       int
	windowStart time.Time
}

// RateLimiter implements a fixed-window rate limiter keyed by tenant ID.
// This is an in-memory implementation suitable for single-instance deployments
// (Phase 1).  Multi-instance deployments should replace this with a Redis-
// backed sliding window limiter.
type RateLimiter struct {
	states     sync.Map // tenantID → *windowState
	limit      int
	windowSize time.Duration
}

// NewRateLimiter returns a RateLimiter that allows at most limit requests per
// window.  A zero or negative limit disables rate limiting.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{limit: limit, windowSize: window}
}

// Allow reports whether a request from tenantID is within the rate limit.
func (rl *RateLimiter) Allow(tenantID string) bool {
	if rl.limit <= 0 {
		return true
	}

	now := time.Now()
	raw, _ := rl.states.LoadOrStore(tenantID, &windowState{windowStart: now})
	ws := raw.(*windowState)

	ws.mu.Lock()
	defer ws.mu.Unlock()

	if now.Sub(ws.windowStart) >= rl.windowSize {
		// Start a new window.
		ws.windowStart = now
		ws.count = 0
	}

	if ws.count >= rl.limit {
		return false
	}
	ws.count++
	return true
}

// Middleware wraps next with per-tenant rate limiting.  The tenant ID is read
// from the request context (populated by tenant.Middleware or AuthMiddleware).
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tid := tenant.FromContext(r.Context())
		if !rl.Allow(tid) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "rate limit exceeded"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
