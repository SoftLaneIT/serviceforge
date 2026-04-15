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

package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"log/slog"

	"github.com/SoftLaneIT/serviceforge/services/tenant-service/internal/domain"
	"github.com/SoftLaneIT/serviceforge/services/tenant-service/internal/handler"
	"github.com/SoftLaneIT/serviceforge/services/tenant-service/internal/repository"
)

// ─── stub repository ──────────────────────────────────────────────────────────

// stubRepo is a configurable test double for repository.Repository.  Each
// field is a function that the handler under test will call; set only the
// functions relevant to a particular test case.
type stubRepo struct {
	createFn    func(context.Context, domain.CreateParams) (*domain.Tenant, error)
	getByIDFn   func(context.Context, string) (*domain.Tenant, error)
	getBySlugFn func(context.Context, string) (*domain.Tenant, error)
	listFn      func(context.Context, repository.ListFilter) (repository.ListResult, error)
	updateFn    func(context.Context, string, domain.UpdateParams) (*domain.Tenant, error)
	deleteFn    func(context.Context, string) error
	pingFn      func(context.Context) error
}

func (s *stubRepo) Create(ctx context.Context, p domain.CreateParams) (*domain.Tenant, error) {
	return s.createFn(ctx, p)
}
func (s *stubRepo) GetByID(ctx context.Context, id string) (*domain.Tenant, error) {
	return s.getByIDFn(ctx, id)
}
func (s *stubRepo) GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	return s.getBySlugFn(ctx, slug)
}
func (s *stubRepo) List(ctx context.Context, f repository.ListFilter) (repository.ListResult, error) {
	return s.listFn(ctx, f)
}
func (s *stubRepo) Update(ctx context.Context, id string, p domain.UpdateParams) (*domain.Tenant, error) {
	return s.updateFn(ctx, id, p)
}
func (s *stubRepo) Delete(ctx context.Context, id string) error {
	return s.deleteFn(ctx, id)
}
func (s *stubRepo) Ping(ctx context.Context) error {
	if s.pingFn != nil {
		return s.pingFn(ctx)
	}
	return nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func silentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), &slog.HandlerOptions{
		Level: slog.LevelError + 10, // effectively discard all output
	}))
}

func fixedTenant() *domain.Tenant {
	return &domain.Tenant{
		ID:        "550e8400-e29b-41d4-a716-446655440000",
		Name:      "Acme Corp",
		Slug:      "acme-corp",
		Plan:      domain.PlanStarter,
		Status:    domain.StatusActive,
		Settings:  map[string]any{},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func ptr[T any](v T) *T { return &v }

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func decodeBody(t *testing.T, body *bytes.Buffer) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.NewDecoder(body).Decode(&m); err != nil {
		t.Fatalf("decode response: %v\nbody: %s", err, body.String())
	}
	return m
}

// newMux builds and registers a Handler onto a fresh ServeMux.
func newMux(repo repository.Repository) *http.ServeMux {
	mux := http.NewServeMux()
	handler.New(repo, silentLogger()).RegisterRoutes(mux)
	return mux
}

// ─── Health ───────────────────────────────────────────────────────────────────

func TestHealth_OK(t *testing.T) {
	mux := newMux(&stubRepo{pingFn: func(context.Context) error { return nil }})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rw := httptest.NewRecorder()
	mux.ServeHTTP(rw, req)

	if rw.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rw.Code)
	}
	body := decodeBody(t, rw.Body)
	if body["status"] != "ok" {
		t.Errorf("want status=ok, got %v", body["status"])
	}
}

func TestHealth_DBDown(t *testing.T) {
	mux := newMux(&stubRepo{pingFn: func(context.Context) error {
		return context.DeadlineExceeded
	}})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rw := httptest.NewRecorder()
	mux.ServeHTTP(rw, req)

	if rw.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d", rw.Code)
	}
	body := decodeBody(t, rw.Body)
	if body["db"] != "unreachable" {
		t.Errorf("want db=unreachable, got %v", body["db"])
	}
}

// ─── CreateTenant ─────────────────────────────────────────────────────────────

func TestCreateTenant_Created(t *testing.T) {
	want := fixedTenant()
	mux := newMux(&stubRepo{createFn: func(_ context.Context, p domain.CreateParams) (*domain.Tenant, error) {
		if p.Name != "Acme Corp" || p.Slug != "acme-corp" {
			t.Errorf("unexpected params: %+v", p)
		}
		return want, nil
	}})

	body := mustMarshal(t, map[string]any{"name": "Acme Corp", "slug": "acme-corp", "plan": "starter"})
	req := httptest.NewRequest(http.MethodPost, "/v1/tenants", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()
	mux.ServeHTTP(rw, req)

	if rw.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d\nbody: %s", rw.Code, rw.Body.String())
	}
	resp := decodeBody(t, rw.Body)
	if resp["id"] != want.ID {
		t.Errorf("want id=%s, got %v", want.ID, resp["id"])
	}
}

func TestCreateTenant_ValidationError(t *testing.T) {
	mux := newMux(&stubRepo{})
	body := mustMarshal(t, map[string]any{"name": "", "slug": "acme-corp", "plan": "starter"})
	req := httptest.NewRequest(http.MethodPost, "/v1/tenants", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()
	mux.ServeHTTP(rw, req)

	if rw.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422, got %d", rw.Code)
	}
}

func TestCreateTenant_SlugConflict(t *testing.T) {
	mux := newMux(&stubRepo{createFn: func(_ context.Context, _ domain.CreateParams) (*domain.Tenant, error) {
		return nil, domain.ErrSlugConflict
	}})
	body := mustMarshal(t, map[string]any{"name": "Acme", "slug": "acme-corp", "plan": "starter"})
	req := httptest.NewRequest(http.MethodPost, "/v1/tenants", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()
	mux.ServeHTTP(rw, req)

	if rw.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d", rw.Code)
	}
}

func TestCreateTenant_BadJSON(t *testing.T) {
	mux := newMux(&stubRepo{})
	req := httptest.NewRequest(http.MethodPost, "/v1/tenants", bytes.NewReader([]byte("not-json")))
	rw := httptest.NewRecorder()
	mux.ServeHTTP(rw, req)

	if rw.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rw.Code)
	}
}

// ─── GetTenant ────────────────────────────────────────────────────────────────

func TestGetTenant_ByID(t *testing.T) {
	want := fixedTenant()
	mux := newMux(&stubRepo{getByIDFn: func(_ context.Context, id string) (*domain.Tenant, error) {
		if id != want.ID {
			t.Errorf("unexpected id %s", id)
		}
		return want, nil
	}})

	req := httptest.NewRequest(http.MethodGet, "/v1/tenants/"+want.ID, nil)
	rw := httptest.NewRecorder()
	mux.ServeHTTP(rw, req)

	if rw.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rw.Code)
	}
}

func TestGetTenant_BySlug(t *testing.T) {
	want := fixedTenant()
	mux := newMux(&stubRepo{getBySlugFn: func(_ context.Context, slug string) (*domain.Tenant, error) {
		if slug != want.Slug {
			t.Errorf("unexpected slug %s", slug)
		}
		return want, nil
	}})

	req := httptest.NewRequest(http.MethodGet, "/v1/tenants/acme-corp?by=slug", nil)
	rw := httptest.NewRecorder()
	mux.ServeHTTP(rw, req)

	if rw.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rw.Code)
	}
	body := decodeBody(t, rw.Body)
	if body["slug"] != want.Slug {
		t.Errorf("want slug=%s, got %v", want.Slug, body["slug"])
	}
}

func TestGetTenant_NotFound(t *testing.T) {
	mux := newMux(&stubRepo{getByIDFn: func(_ context.Context, _ string) (*domain.Tenant, error) {
		return nil, domain.ErrNotFound
	}})
	req := httptest.NewRequest(http.MethodGet, "/v1/tenants/does-not-exist", nil)
	rw := httptest.NewRecorder()
	mux.ServeHTTP(rw, req)

	if rw.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", rw.Code)
	}
}

// ─── ListTenants ──────────────────────────────────────────────────────────────

func TestListTenants_DefaultPagination(t *testing.T) {
	mux := newMux(&stubRepo{listFn: func(_ context.Context, f repository.ListFilter) (repository.ListResult, error) {
		if f.Limit != 20 || f.Offset != 0 {
			t.Errorf("unexpected pagination: limit=%d offset=%d", f.Limit, f.Offset)
		}
		return repository.ListResult{Tenants: []domain.Tenant{*fixedTenant()}, Total: 1}, nil
	}})

	req := httptest.NewRequest(http.MethodGet, "/v1/tenants", nil)
	rw := httptest.NewRecorder()
	mux.ServeHTTP(rw, req)

	if rw.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rw.Code)
	}
	body := decodeBody(t, rw.Body)
	if body["total"] != float64(1) {
		t.Errorf("want total=1, got %v", body["total"])
	}
}

func TestListTenants_CustomPagination(t *testing.T) {
	mux := newMux(&stubRepo{listFn: func(_ context.Context, f repository.ListFilter) (repository.ListResult, error) {
		if f.Limit != 5 || f.Offset != 10 {
			t.Errorf("unexpected pagination: limit=%d offset=%d", f.Limit, f.Offset)
		}
		return repository.ListResult{Tenants: []domain.Tenant{}, Total: 0}, nil
	}})

	req := httptest.NewRequest(http.MethodGet, "/v1/tenants?limit=5&offset=10", nil)
	rw := httptest.NewRecorder()
	mux.ServeHTTP(rw, req)

	if rw.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rw.Code)
	}
}

func TestListTenants_InvalidStatus(t *testing.T) {
	mux := newMux(&stubRepo{})
	req := httptest.NewRequest(http.MethodGet, "/v1/tenants?status=unknown", nil)
	rw := httptest.NewRecorder()
	mux.ServeHTTP(rw, req)

	if rw.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rw.Code)
	}
}

// ─── UpdateTenant ─────────────────────────────────────────────────────────────

func TestUpdateTenant_OK(t *testing.T) {
	want := fixedTenant()
	want.Name = "New Name"
	mux := newMux(&stubRepo{updateFn: func(_ context.Context, id string, p domain.UpdateParams) (*domain.Tenant, error) {
		if id != want.ID {
			t.Errorf("unexpected id %s", id)
		}
		if p.Name == nil || *p.Name != "New Name" {
			t.Errorf("unexpected name %v", p.Name)
		}
		return want, nil
	}})

	body := mustMarshal(t, map[string]any{"name": "New Name"})
	req := httptest.NewRequest(http.MethodPatch, "/v1/tenants/"+want.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()
	mux.ServeHTTP(rw, req)

	if rw.Code != http.StatusOK {
		t.Fatalf("want 200, got %d\nbody: %s", rw.Code, rw.Body.String())
	}
}

func TestUpdateTenant_EmptyBodyIsValidationError(t *testing.T) {
	mux := newMux(&stubRepo{})
	req := httptest.NewRequest(http.MethodPatch, "/v1/tenants/some-id", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()
	mux.ServeHTTP(rw, req)

	if rw.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422, got %d", rw.Code)
	}
}

func TestUpdateTenant_NotFound(t *testing.T) {
	mux := newMux(&stubRepo{updateFn: func(_ context.Context, _ string, _ domain.UpdateParams) (*domain.Tenant, error) {
		return nil, domain.ErrNotFound
	}})
	body := mustMarshal(t, map[string]any{"name": "X"})
	req := httptest.NewRequest(http.MethodPatch, "/v1/tenants/missing", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()
	mux.ServeHTTP(rw, req)

	if rw.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", rw.Code)
	}
}

// ─── DeleteTenant ─────────────────────────────────────────────────────────────

func TestDeleteTenant_NoContent(t *testing.T) {
	mux := newMux(&stubRepo{deleteFn: func(_ context.Context, id string) error {
		if id != "some-uuid" {
			t.Errorf("unexpected id %s", id)
		}
		return nil
	}})

	req := httptest.NewRequest(http.MethodDelete, "/v1/tenants/some-uuid", nil)
	rw := httptest.NewRecorder()
	mux.ServeHTTP(rw, req)

	if rw.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d", rw.Code)
	}
}

func TestDeleteTenant_NotFound(t *testing.T) {
	mux := newMux(&stubRepo{deleteFn: func(_ context.Context, _ string) error {
		return domain.ErrNotFound
	}})
	req := httptest.NewRequest(http.MethodDelete, "/v1/tenants/ghost", nil)
	rw := httptest.NewRecorder()
	mux.ServeHTTP(rw, req)

	if rw.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", rw.Code)
	}
}
