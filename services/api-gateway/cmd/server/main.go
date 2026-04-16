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

// Command server is the entry point for the api-gateway.
//
// The gateway sits in front of all other ServiceForge microservices.  It:
//   - validates API keys by delegating to auth-service
//   - enforces per-tenant rate limits (in-memory fixed window, Phase 1)
//   - reverse-proxies authenticated requests to the appropriate downstream
//
// Routes (all authenticated unless noted):
//
//	GET  /health                  — local health check (no auth)
//	*    /v1/tenants/*            — proxied to tenant-service (no auth, admin)
//	*    /v1/keys/*               — proxied to auth-service   (no auth, admin)
//	*    /v1/bookings/*           — proxied to booking-service (auth required)
//	*    /v1/config/*             — proxied to config-service  (auth required)
//
// Environment variables:
//
//	PORT                 HTTP listen port (default: 8081)
//	AUTH_SERVICE_URL     (default: http://localhost:8082)
//	TENANT_SERVICE_URL   (default: http://localhost:8083)
//	BOOKING_SERVICE_URL  (default: http://localhost:8084)
//	CONFIG_SERVICE_URL   (default: http://localhost:8085)
//	RATE_LIMIT           requests per tenant per minute (default: 100; 0 = disabled)
//	LOG_LEVEL            debug | info | warn | error (default: info)
//	LOG_FORMAT           json | text (default: json)
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/SoftLaneIT/serviceforge/packages/go-common/config"
	"github.com/SoftLaneIT/serviceforge/packages/go-common/logger"
	"github.com/SoftLaneIT/serviceforge/packages/go-common/tenant"
	gw "github.com/SoftLaneIT/serviceforge/services/api-gateway/internal/middleware"
	"github.com/SoftLaneIT/serviceforge/services/api-gateway/internal/proxy"
	"github.com/SoftLaneIT/serviceforge/services/api-gateway/internal/registry"
)

func main() {
	log := logger.NewFromEnv("api-gateway")

	//  service registry
	reg, err := registry.Load()
	if err != nil {
		log.Error("load registry", slog.Any("error", err))
		os.Exit(1)
	}

	//  rate limiter
	rateLimit := parseIntDefault(config.GetEnv("RATE_LIMIT", "100"), 100)
	rateLimiter := gw.NewRateLimiter(rateLimit, time.Minute)

	//  auth middleware ─
	authServiceURL := config.GetEnv("AUTH_SERVICE_URL", "http://localhost:8082")
	authMiddleware := gw.AuthMiddleware(authServiceURL, log)

	//  reverse proxies ─
	authProxy := proxy.New(reg.URL(registry.Auth), log)
	tenantProxy := proxy.New(reg.URL(registry.Tenant), log)
	bookingProxy := proxy.New(reg.URL(registry.Booking), log)
	configProxy := proxy.New(reg.URL(registry.Config), log)

	//  routing ─
	//
	// Unauthenticated routes are registered first so the auth middleware only
	// wraps the paths that need it.
	mux := http.NewServeMux()

	// Health — no auth.
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusOK, map[string]any{"service": "api-gateway", "status": "ok"})
	})

	// Admin routes — no client auth (protected by network policy / internal LB).
	// /v1/tenants/{...} → tenant-service
	mux.Handle("/v1/tenants/", tenantProxy)
	mux.Handle("/v1/tenants", tenantProxy)
	// /v1/keys/{...} → auth-service (key management)
	mux.Handle("/v1/keys/", authProxy)
	mux.Handle("/v1/keys", authProxy)

	//  authenticated + rate-limited routes ─
	//
	// Middleware chain (innermost first):
	//   bookingProxy / configProxy
	//   → rateLimiter.Middleware
	//   → authMiddleware
	//   → tenant.Middleware (resolves X-Tenant-ID from header context)
	//   → logger.HTTPMiddleware (structured request logging)

	withAuth := func(h http.Handler) http.Handler {
		return authMiddleware(rateLimiter.Middleware(h))
	}

	mux.Handle("/v1/bookings/", withAuth(bookingProxy))
	mux.Handle("/v1/bookings", withAuth(bookingProxy))
	mux.Handle("/v1/config/", withAuth(configProxy))
	mux.Handle("/v1/config", withAuth(configProxy))

	// Outer middleware applied to the whole mux.
	httpHandler := tenant.Middleware(logger.HTTPMiddleware(log)(mux))

	//  HTTP server
	port := config.GetEnv("PORT", "8081")
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      httpHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 60 * time.Second, // allow slower upstreams
		IdleTimeout:  120 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info("api-gateway starting",
			slog.String("port", port),
			slog.Int("rateLimit", rateLimit),
		)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	select {
	case sig := <-quit:
		log.Info("shutdown signal received", slog.String("signal", sig.String()))
	case err := <-serverErr:
		log.Error("server error", slog.Any("error", err))
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("forced shutdown", slog.Any("error", err))
	}
	log.Info("api-gateway stopped")
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func parseIntDefault(s string, def int) int {
	v, err := strconv.Atoi(s)
	if err != nil || v < 0 {
		return def
	}
	return v
}
