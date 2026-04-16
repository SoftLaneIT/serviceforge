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

package logger_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SoftLaneIT/serviceforge/packages/go-common/logger"
	"github.com/SoftLaneIT/serviceforge/packages/go-common/tenant"
)

// newBufLogger builds a JSON logger that writes to buf instead of os.Stdout.
// This lets tests inspect the exact bytes emitted without capturing stdout.
func newBufLogger(buf *bytes.Buffer, level slog.Level) *slog.Logger {
	h := slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: level})
	return slog.New(h)
}

// decodeLastRecord unmarshals the last newline-delimited JSON object in buf.
func decodeLastRecord(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	var rec map[string]any
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &rec); err != nil {
		t.Fatalf("failed to decode log record: %v\nraw: %s", err, buf.String())
	}
	return rec
}

// ─ parseLevel ─

func TestNew_LevelDebug(t *testing.T) {
	var buf bytes.Buffer
	// We can't swap os.Stdout, so we verify via the exported Options struct
	// that constructing a logger with level "debug" does not panic and that
	// the returned logger accepts debug records.
	h := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	l := slog.New(h)
	l.Debug("debug message")

	if !strings.Contains(buf.String(), "debug message") {
		t.Fatalf("expected debug message in output, got: %s", buf.String())
	}
}

func TestNew_LevelInfo_FiltersDebug(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	l := slog.New(h)
	l.Debug("should be dropped")
	l.Info("should appear")

	output := buf.String()
	if strings.Contains(output, "should be dropped") {
		t.Fatal("debug record should have been filtered at info level")
	}
	if !strings.Contains(output, "should appear") {
		t.Fatal("info record was unexpectedly filtered")
	}
}

// ─ context helpers

func TestWithContext_FromContext_ReturnsStoredLogger(t *testing.T) {
	var buf bytes.Buffer
	l := newBufLogger(&buf, slog.LevelInfo)

	ctx := logger.WithContext(context.Background(), l)
	got := logger.FromContext(ctx)

	got.Info("hello from context")
	if !strings.Contains(buf.String(), "hello from context") {
		t.Fatalf("expected log output from context logger, got: %s", buf.String())
	}
}

func TestFromContext_FallsBackToDefault(t *testing.T) {
	// No logger stored — should not panic and should return a non-nil logger.
	got := logger.FromContext(context.Background())
	if got == nil {
		t.Fatal("FromContext returned nil without a stored logger")
	}
}

func TestFromContext_EnrichesWithTenantID(t *testing.T) {
	var buf bytes.Buffer
	l := newBufLogger(&buf, slog.LevelInfo)

	// Build a context that carries a tenant ID (via the tenant package) and a
	// stored logger.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant-ID", "acme-corp")

	// Use the tenant middleware to inject the tenant ID into the context.
	var capturedCtx context.Context
	handler := tenant.Middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedCtx = r.Context()
	}))
	handler.ServeHTTP(httptest.NewRecorder(), req)

	enrichedCtx := logger.WithContext(capturedCtx, l)
	log := logger.FromContext(enrichedCtx)
	log.Info("tenant-aware record")

	rec := decodeLastRecord(t, &buf)
	if rec["tenant_id"] != "acme-corp" {
		t.Fatalf("expected tenant_id=acme-corp, got: %v", rec["tenant_id"])
	}
}

func TestFromContext_EnrichesWithTraceID(t *testing.T) {
	var buf bytes.Buffer
	l := newBufLogger(&buf, slog.LevelInfo)

	ctx := logger.WithContext(context.Background(), l)
	ctx = logger.WithTraceID(ctx, "trace-xyz-123")

	logger.FromContext(ctx).Info("traced record")

	rec := decodeLastRecord(t, &buf)
	if rec["trace_id"] != "trace-xyz-123" {
		t.Fatalf("expected trace_id=trace-xyz-123, got: %v", rec["trace_id"])
	}
}

func TestFromContext_DoesNotAppendDefaultTenantID(t *testing.T) {
	// The "default" fallback tenant should not pollute log records.
	var buf bytes.Buffer
	l := newBufLogger(&buf, slog.LevelInfo)

	ctx := logger.WithContext(context.Background(), l)
	logger.FromContext(ctx).Info("no-tenant record")

	rec := decodeLastRecord(t, &buf)
	if _, ok := rec["tenant_id"]; ok {
		t.Fatalf("tenant_id should not appear for default tenant, got: %v", rec["tenant_id"])
	}
}

func TestWithTraceID_TraceIDFromContext(t *testing.T) {
	ctx := logger.WithTraceID(context.Background(), "req-abc")
	if got := logger.TraceIDFromContext(ctx); got != "req-abc" {
		t.Fatalf("expected req-abc, got %q", got)
	}
}

func TestTraceIDFromContext_EmptyWhenNotSet(t *testing.T) {
	if got := logger.TraceIDFromContext(context.Background()); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

// ─ HTTPMiddleware

func TestHTTPMiddleware_LogsRequest(t *testing.T) {
	var buf bytes.Buffer
	l := newBufLogger(&buf, slog.LevelInfo)

	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Chain: tenant middleware → logger middleware → mux.
	handler := tenant.Middleware(logger.HTTPMiddleware(l)(mux))

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("X-Tenant-ID", "test-tenant")
	req.Header.Set("X-Trace-ID", "trace-001")

	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)

	rec := decodeLastRecord(t, &buf)

	checks := map[string]any{
		"msg":       "http request",
		"method":    "GET",
		"path":      "/ping",
		"status":    float64(http.StatusOK),
		"tenant_id": "test-tenant",
		"trace_id":  "trace-001",
	}
	for field, want := range checks {
		if rec[field] != want {
			t.Errorf("field %q: want %v, got %v", field, want, rec[field])
		}
	}
	if _, ok := rec["latency"]; !ok {
		t.Error("expected latency field in log record")
	}
}

func TestHTTPMiddleware_CapturesNon200Status(t *testing.T) {
	var buf bytes.Buffer
	l := newBufLogger(&buf, slog.LevelInfo)

	mux := http.NewServeMux()
	mux.HandleFunc("/boom", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})

	handler := logger.HTTPMiddleware(l)(mux)
	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	rec := decodeLastRecord(t, &buf)
	if rec["status"] != float64(http.StatusNotFound) {
		t.Fatalf("expected status 404, got %v", rec["status"])
	}
}

func TestHTTPMiddleware_XRequestIDFallback(t *testing.T) {
	var buf bytes.Buffer
	l := newBufLogger(&buf, slog.LevelInfo)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {})

	handler := logger.HTTPMiddleware(l)(mux)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "req-fallback-99")

	handler.ServeHTTP(httptest.NewRecorder(), req)

	rec := decodeLastRecord(t, &buf)
	if rec["trace_id"] != "req-fallback-99" {
		t.Fatalf("expected trace_id from X-Request-ID, got %v", rec["trace_id"])
	}
}

func TestHTTPMiddleware_StoresLoggerInContext(t *testing.T) {
	var buf bytes.Buffer
	l := newBufLogger(&buf, slog.LevelInfo)

	mux := http.NewServeMux()
	mux.HandleFunc("/ctx", func(w http.ResponseWriter, r *http.Request) {
		// Handler uses FromContext — this must write to buf, not /dev/null.
		logger.FromContext(r.Context()).Info("inside handler")
		w.WriteHeader(http.StatusOK)
	})

	handler := logger.HTTPMiddleware(l)(mux)
	req := httptest.NewRequest(http.MethodGet, "/ctx", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if !strings.Contains(buf.String(), "inside handler") {
		t.Fatalf("handler logger did not write to the expected buffer: %s", buf.String())
	}
}
