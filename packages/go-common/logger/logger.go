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

// Package logger provides a structured, context-aware logger for all ServiceForge
// services.  It is built on top of the standard library's log/slog (Go 1.21+)
// and therefore carries zero third-party dependencies.
//
// # Typical service startup
//
//	func main() {
//	    log := logger.NewFromEnv("booking-service")
//	    log.Info("service starting", slog.String("port", port))
//	    ...
//	    handler := logger.HTTPMiddleware(log)(mux)
//	}
//
// # Context propagation
//
// Every outgoing log record is enriched with the tenant_id and trace_id that
// are already stored in the request context.  Use FromContext inside handlers:
//
//	func (h *Handler) CreateBooking(w http.ResponseWriter, r *http.Request) {
//	    log := logger.FromContext(r.Context())
//	    log.Info("creating booking", slog.String("customer", id))
//	}
//
// # Environment variables
//
//   - LOG_LEVEL  – debug | info | warn | error  (default: info)
//   - LOG_FORMAT – json | text                  (default: json)
package logger

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/SoftLaneIT/serviceforge/packages/go-common/tenant"
)

// ─── context keys ────────────────────────────────────────────────────────────

type ctxKey string

const (
	loggerKey  ctxKey = "sf_logger"
	traceIDKey ctxKey = "sf_trace_id"
)

// ─── Options ─────────────────────────────────────────────────────────────────

// Options controls how a logger is constructed.
type Options struct {
	// Level is the minimum log level that will be emitted.
	// Accepted values (case-insensitive): "debug", "info", "warn", "error".
	// Empty string defaults to "info".
	Level string

	// Format selects the output encoding.
	// "text" emits human-readable key=value lines; everything else emits JSON.
	// JSON is the default because it is the format consumed by log aggregators
	// (Loki, CloudWatch, ELK).
	Format string

	// Service is emitted as a fixed "service" field on every log record so that
	// log aggregation queries can filter by service name without parsing the
	// message.
	Service string
}

// ─── Constructors ─────────────────────────────────────────────────────────────

// New builds a *slog.Logger from opts, installs it as the process-wide default
// (slog.SetDefault), and returns it.  Services that want an isolated logger
// without touching the default should use the returned value directly.
func New(opts Options) *slog.Logger {
	handlerOpts := &slog.HandlerOptions{
		Level:     parseLevel(opts.Level),
		AddSource: parseLevel(opts.Level) == slog.LevelDebug,
	}

	var handler slog.Handler
	if strings.ToLower(opts.Format) == "text" {
		handler = slog.NewTextHandler(os.Stdout, handlerOpts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, handlerOpts)
	}

	l := slog.New(handler)
	if opts.Service != "" {
		l = l.With(slog.String("service", opts.Service))
	}

	slog.SetDefault(l)
	return l
}

// NewFromEnv is the standard entry-point for every ServiceForge binary.  It
// reads LOG_LEVEL and LOG_FORMAT from the environment, sets the named service
// attribute, and registers the logger as the process-wide slog default.
//
//	log := logger.NewFromEnv("tenant-service")
func NewFromEnv(service string) *slog.Logger {
	return New(Options{
		Level:   os.Getenv("LOG_LEVEL"),
		Format:  os.Getenv("LOG_FORMAT"),
		Service: service,
	})
}

// ─── Context helpers ──────────────────────────────────────────────────────────

// WithContext stores l in ctx and returns the enriched context.  Call this once
// per request (typically inside HTTPMiddleware) so that handler code can
// retrieve a pre-enriched logger via FromContext.
func WithContext(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

// FromContext retrieves the logger stored by WithContext and returns a copy
// pre-enriched with any tenant_id and trace_id already present in ctx.
// If no logger was stored, the process-wide slog default is used as the base.
//
// This is the only logger retrieval function handler code should ever need.
func FromContext(ctx context.Context) *slog.Logger {
	l, _ := ctx.Value(loggerKey).(*slog.Logger)
	if l == nil {
		l = slog.Default()
	}

	var attrs []any

	if tid := tenant.FromContext(ctx); tid != "" && tid != "default" {
		attrs = append(attrs, slog.String("tenant_id", tid))
	}
	if traceID := TraceIDFromContext(ctx); traceID != "" {
		attrs = append(attrs, slog.String("trace_id", traceID))
	}

	if len(attrs) > 0 {
		return l.With(attrs...)
	}
	return l
}

// WithTraceID stores traceID in ctx so that subsequent calls to FromContext
// will automatically include it in every log record.  The gateway sets this
// from the incoming X-Trace-ID header; downstream services propagate it
// by forwarding the header on outgoing calls.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// TraceIDFromContext returns the trace ID stored by WithTraceID, or empty
// string if none was set.
func TraceIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(traceIDKey).(string); ok {
		return v
	}
	return ""
}

// ─── HTTP middleware ───────────────────────────────────────────────────────────

// HTTPMiddleware returns an http.Handler middleware that:
//
//  1. Extracts (or notes the absence of) a trace ID from the X-Trace-ID or
//     X-Request-ID request header.
//  2. Stores the service logger and trace ID in the request context so that
//     handler code can call FromContext to obtain a pre-enriched logger.
//  3. After the handler returns, emits a single INFO-level "http request"
//     record containing: method, path, status, latency, remote_addr,
//     tenant_id (when present), trace_id (when present).
//
// Chain this after the tenant.Middleware so that tenant_id is available:
//
//	http.ListenAndServe(addr, tenant.Middleware(logger.HTTPMiddleware(log)(mux)))
func HTTPMiddleware(l *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Prefer X-Trace-ID; fall back to X-Request-ID for compatibility
			// with AWS ALB / GCP load balancers that inject X-Request-ID.
			traceID := r.Header.Get("X-Trace-ID")
			if traceID == "" {
				traceID = r.Header.Get("X-Request-ID")
			}

			ctx := r.Context()
			ctx = WithContext(ctx, l)
			if traceID != "" {
				ctx = WithTraceID(ctx, traceID)
			}

			// Wrap the ResponseWriter so we can capture the status code written
			// by the handler without interfering with its normal operation.
			rw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, r.WithContext(ctx))

			attrs := []any{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rw.status),
				slog.Duration("latency", time.Since(start)),
				slog.String("remote_addr", r.RemoteAddr),
			}
			if tid := tenant.FromContext(ctx); tid != "" && tid != "default" {
				attrs = append(attrs, slog.String("tenant_id", tid))
			}
			if traceID != "" {
				attrs = append(attrs, slog.String("trace_id", traceID))
			}

			l.InfoContext(ctx, "http request", attrs...)
		})
	}
}

// ─── internal helpers ─────────────────────────────────────────────────────────

// statusRecorder wraps http.ResponseWriter to capture the HTTP status code.
// The zero value's status field must be initialised to 200 before use.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if r.wroteHeader {
		return
	}
	r.status = code
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(b)
}

// Unwrap allows net/http internals (e.g. http.Flusher, http.Hijacker) to reach
// the underlying ResponseWriter through the wrapper.
func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

// parseLevel converts a string level to slog.Level.
// Unknown / empty values default to slog.LevelInfo.
func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
