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

// Command server is the entry point for the tenant-service.
//
// Environment variables:
//
//	PORT           HTTP listen port (default: 8083)
//	DATABASE_URL   PostgreSQL connection string (required)
//	LOG_LEVEL      debug | info | warn | error (default: info)
//	LOG_FORMAT     json | text (default: json)
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SoftLaneIT/serviceforge/packages/go-common/config"
	"github.com/SoftLaneIT/serviceforge/packages/go-common/logger"
	commidware "github.com/SoftLaneIT/serviceforge/packages/go-common/middleware"
	"github.com/SoftLaneIT/serviceforge/packages/go-common/tenant"
	"github.com/SoftLaneIT/serviceforge/services/tenant-service/internal/handler"
	"github.com/SoftLaneIT/serviceforge/services/tenant-service/internal/repository"
)

func main() {
	//  logger
	log := logger.NewFromEnv("tenant-service")

	//  database
	dsn := config.GetEnv("DATABASE_URL",
		"postgres://serviceforge:serviceforge@localhost:5432/serviceforge?sslmode=disable")

	pool := mustConnectPool(log, dsn)
	defer pool.Close()

	//  repository + handler ─
	repo := repository.NewPostgres(pool)
	h := handler.New(repo, log)

	//  HTTP server
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Middleware chain (outermost first):
	//   tenant.Middleware  → injects tenant_id from X-Tenant-ID header
	//   logger.HTTPMiddleware → structured request logging with tenant_id + trace_id
	corsOrigins := config.GetEnv("CORS_ORIGINS", "http://localhost:3000")
	httpHandler := commidware.CORS(corsOrigins)(
		tenant.Middleware(logger.HTTPMiddleware(log)(mux)),
	)

	port := config.GetEnv("PORT", "8083")
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      httpHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	//  graceful shutdown ─
	// Start the HTTP server in a goroutine so the main goroutine can block on
	// the signal channel.
	serverErr := make(chan error, 1)
	go func() {
		log.Info("tenant-service starting", slog.String("port", port))
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

	// Allow up to 30 s for in-flight requests to complete.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("forced shutdown", slog.Any("error", err))
	}
	log.Info("tenant-service stopped")
}

// mustConnectPool attempts to connect to PostgreSQL with exponential back-off
// retries.  It calls os.Exit(1) if the database is unreachable after all
// attempts, because a tenant-service without a database is not useful.
func mustConnectPool(log *slog.Logger, dsn string) *pgxpool.Pool {
	const maxAttempts = 5

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Error("invalid DATABASE_URL", slog.Any("error", err))
		os.Exit(1)
	}

	// Reasonable pool limits for a Phase 1 single-replica deployment.
	cfg.MaxConns = 10
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 1 * time.Hour
	cfg.MaxConnIdleTime = 5 * time.Minute

	ctx := context.Background()
	var pool *pgxpool.Pool

	for attempt := range maxAttempts {
		pool, err = pgxpool.NewWithConfig(ctx, cfg)
		if err == nil {
			if pingErr := pool.Ping(ctx); pingErr == nil {
				log.Info("database connected", slog.Int("attempt", attempt+1))
				return pool
			} else {
				pool.Close()
				err = pingErr
			}
		}

		wait := time.Duration(1<<attempt) * time.Second // 1, 2, 4, 8, 16 s
		log.Warn("database not ready, retrying",
			slog.Int("attempt", attempt+1),
			slog.Int("maxAttempts", maxAttempts),
			slog.Duration("retryIn", wait),
			slog.Any("error", err),
		)
		time.Sleep(wait)
	}

	log.Error("could not connect to database after retries", slog.Any("error", err))
	os.Exit(1)
	return nil // unreachable — satisfies the compiler
}
