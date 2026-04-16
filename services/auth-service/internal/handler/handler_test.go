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
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/SoftLaneIT/serviceforge/packages/go-common/tenant"
	"github.com/SoftLaneIT/serviceforge/services/auth-service/internal/cache"
	"github.com/SoftLaneIT/serviceforge/services/auth-service/internal/domain"
	"github.com/SoftLaneIT/serviceforge/services/auth-service/internal/handler"
	"github.com/SoftLaneIT/serviceforge/services/auth-service/internal/keygen"
	"github.com/SoftLaneIT/serviceforge/services/auth-service/internal/repository"
)

//  fakes

// fakeRepo is a thread-safe in-memory Repository.
type fakeRepo struct {
	mu   sync.Mutex
	keys map[string]*domain.APIKey // id → key
	// byHash maps key_hash → id
	byHash map[string]string
	// pingErr, if set, is returned by Ping.
	pingErr error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{keys: map[string]*domain.APIKey{}, byHash: map[string]string{}}
}

func (r *fakeRepo) Create(_ context.Context, p domain.CreateParams) (*domain.APIKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := &domain.APIKey{
		ID:          "key-" + p.KeyPrefix,
		TenantID:    p.TenantID,
		Name:        p.Name,
		KeyPrefix:   p.KeyPrefix,
		Environment: p.Environment,
		ModuleScope: p.ModuleScope,
		Status:      domain.KeyStatusActive,
		CreatedAt:   time.Now(),
	}
	if p.ExpiresAt != nil {
		k.ExpiresAt = p.ExpiresAt
	}
	r.keys[k.ID] = k
	r.byHash[p.KeyHash] = k.ID
	return k, nil
}

func (r *fakeRepo) GetByID(_ context.Context, id, tenantID string) (*domain.APIKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k, ok := r.keys[id]
	if !ok || k.TenantID != tenantID {
		return nil, domain.ErrNotFound
	}
	return k, nil
}

func (r *fakeRepo) List(_ context.Context, tenantID string) ([]domain.APIKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.APIKey
	for _, k := range r.keys {
		if k.TenantID == tenantID {
			out = append(out, *k)
		}
	}
	if out == nil {
		out = []domain.APIKey{}
	}
	return out, nil
}

func (r *fakeRepo) Revoke(_ context.Context, id, tenantID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k, ok := r.keys[id]
	if !ok || k.TenantID != tenantID || k.Status == domain.KeyStatusRevoked {
		return domain.ErrNotFound
	}
	k.Status = domain.KeyStatusRevoked
	now := time.Now()
	k.RevokedAt = &now
	return nil
}

func (r *fakeRepo) GetByHash(_ context.Context, hash string) (*domain.APIKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.byHash[hash]
	if !ok {
		return nil, domain.ErrInvalidKey
	}
	k := r.keys[id]
	if k.Status == domain.KeyStatusRevoked {
		return nil, domain.ErrInvalidKey
	}
	if k.ExpiresAt != nil && k.ExpiresAt.Before(time.Now()) {
		return nil, domain.ErrExpiredKey
	}
	return k, nil
}

func (r *fakeRepo) UpdateLastUsed(_ context.Context, _ string) error { return nil }

func (r *fakeRepo) Ping(_ context.Context) error { return r.pingErr }

// compile-time check
var _ repository.Repository = (*fakeRepo)(nil)

//  helpers

func newHandler(t *testing.T) (*handler.Handler, *fakeRepo) {
	t.Helper()
	repo := newFakeRepo()
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	h := handler.New(repo, cache.NoopCache{}, log)
	return h, repo
}

// withTenant injects a tenant ID into the request context via the tenant
// middleware so that tenant.FromContext works inside handlers.
func withTenant(r *http.Request, tid string) *http.Request {
	ctx := tenant.NewContext(r.Context(), tid)
	return r.WithContext(ctx)
}

func do(t *testing.T, h *handler.Handler, method, path string, body any, tenantID string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if tenantID != "" {
		req = withTenant(req, tenantID)
	}
	rr := httptest.NewRecorder()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rr, req)
	return rr
}

func decodeJSON(t *testing.T, rr *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.NewDecoder(rr.Body).Decode(dst); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

//  IssueKey ─

func TestIssueKey_Created(t *testing.T) {
	h, _ := newHandler(t)
	rr := do(t, h, http.MethodPost, "/v1/keys", map[string]any{
		"name":        "my-key",
		"environment": "sandbox",
	}, "tenant-abc")

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", rr.Code, rr.Body)
	}

	var resp struct {
		ID       string `json:"id"`
		RawKey   string `json:"rawKey"`
		Name     string `json:"name"`
		TenantID string `json:"tenantId"`
	}
	decodeJSON(t, rr, &resp)

	if resp.RawKey == "" {
		t.Error("rawKey should be present in create response")
	}
	if resp.Name != "my-key" {
		t.Errorf("name = %q, want %q", resp.Name, "my-key")
	}
	if resp.TenantID != "tenant-abc" {
		t.Errorf("tenantId = %q, want %q", resp.TenantID, "tenant-abc")
	}
}

func TestIssueKey_DefaultsToSandbox(t *testing.T) {
	h, _ := newHandler(t)
	rr := do(t, h, http.MethodPost, "/v1/keys", map[string]any{"name": "x"}, "t1")
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d; body: %s", rr.Code, rr.Body)
	}
	var resp struct {
		Environment string `json:"environment"`
	}
	decodeJSON(t, rr, &resp)
	if resp.Environment != "sandbox" {
		t.Errorf("environment = %q, want sandbox", resp.Environment)
	}
}

func TestIssueKey_MissingName(t *testing.T) {
	h, _ := newHandler(t)
	rr := do(t, h, http.MethodPost, "/v1/keys", map[string]any{}, "t1")
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rr.Code)
	}
}

func TestIssueKey_InvalidEnvironment(t *testing.T) {
	h, _ := newHandler(t)
	rr := do(t, h, http.MethodPost, "/v1/keys", map[string]any{
		"name": "k", "environment": "nope",
	}, "t1")
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rr.Code)
	}
}

func TestIssueKey_MissingTenantID(t *testing.T) {
	h, _ := newHandler(t)
	rr := do(t, h, http.MethodPost, "/v1/keys", map[string]any{"name": "k"}, "")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

//  ListKeys ─

func TestListKeys_Empty(t *testing.T) {
	h, _ := newHandler(t)
	rr := do(t, h, http.MethodGet, "/v1/keys", nil, "t1")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", rr.Code, rr.Body)
	}
	var resp struct {
		Data  []any `json:"data"`
		Total int   `json:"total"`
	}
	decodeJSON(t, rr, &resp)
	if resp.Total != 0 || len(resp.Data) != 0 {
		t.Errorf("expected empty list, got %+v", resp)
	}
}

func TestListKeys_ReturnsOwnTenantKeys(t *testing.T) {
	h, _ := newHandler(t)
	// Issue two keys for different tenants.
	do(t, h, http.MethodPost, "/v1/keys", map[string]any{"name": "a"}, "t1")
	do(t, h, http.MethodPost, "/v1/keys", map[string]any{"name": "b"}, "t1")
	do(t, h, http.MethodPost, "/v1/keys", map[string]any{"name": "c"}, "t2")

	rr := do(t, h, http.MethodGet, "/v1/keys", nil, "t1")
	var resp struct {
		Total int `json:"total"`
	}
	decodeJSON(t, rr, &resp)
	if resp.Total != 2 {
		t.Errorf("total = %d, want 2", resp.Total)
	}
}

//  GetKey ─

func TestGetKey_NotFound(t *testing.T) {
	h, _ := newHandler(t)
	rr := do(t, h, http.MethodGet, "/v1/keys/nonexistent", nil, "t1")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}

func TestGetKey_WrongTenant(t *testing.T) {
	h, _ := newHandler(t)
	// Issue a key for t1.
	rr1 := do(t, h, http.MethodPost, "/v1/keys", map[string]any{"name": "k"}, "t1")
	var created struct {
		ID string `json:"id"`
	}
	decodeJSON(t, rr1, &created)

	// Try to fetch with t2.
	rr2 := do(t, h, http.MethodGet, "/v1/keys/"+created.ID, nil, "t2")
	if rr2.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 for cross-tenant access", rr2.Code)
	}
}

//  RevokeKey

func TestRevokeKey_NoContent(t *testing.T) {
	h, _ := newHandler(t)
	rr1 := do(t, h, http.MethodPost, "/v1/keys", map[string]any{"name": "k"}, "t1")
	var created struct {
		ID string `json:"id"`
	}
	decodeJSON(t, rr1, &created)

	rr2 := do(t, h, http.MethodDelete, "/v1/keys/"+created.ID, nil, "t1")
	if rr2.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body: %s", rr2.Code, rr2.Body)
	}
}

func TestRevokeKey_NotFound(t *testing.T) {
	h, _ := newHandler(t)
	rr := do(t, h, http.MethodDelete, "/v1/keys/ghost", nil, "t1")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}

//  ValidateKey

func TestValidateKey_Valid(t *testing.T) {
	h, repo := newHandler(t)
	// Issue a key and capture the raw value.
	rr1 := do(t, h, http.MethodPost, "/v1/keys", map[string]any{"name": "k"}, "t1")
	var created struct {
		ID     string `json:"id"`
		RawKey string `json:"rawKey"`
	}
	decodeJSON(t, rr1, &created)

	// Manually insert the hash into the fake repo so GetByHash works.
	hash := keygen.HashRaw(created.RawKey)
	repo.mu.Lock()
	repo.byHash[hash] = created.ID
	repo.mu.Unlock()

	rr2 := do(t, h, http.MethodPost, "/v1/keys/validate", map[string]any{"key": created.RawKey}, "")
	if rr2.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rr2.Code, rr2.Body)
	}
	var resp struct {
		TenantID string `json:"tenantId"`
		KeyID    string `json:"keyId"`
	}
	decodeJSON(t, rr2, &resp)
	if resp.TenantID != "t1" {
		t.Errorf("tenantId = %q, want %q", resp.TenantID, "t1")
	}
	if resp.KeyID != created.ID {
		t.Errorf("keyId = %q, want %q", resp.KeyID, created.ID)
	}
}

func TestValidateKey_InvalidKey(t *testing.T) {
	h, _ := newHandler(t)
	rr := do(t, h, http.MethodPost, "/v1/keys/validate", map[string]any{"key": "sf_test_bogus"}, "")
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestValidateKey_RevokedKey(t *testing.T) {
	h, repo := newHandler(t)
	rr1 := do(t, h, http.MethodPost, "/v1/keys", map[string]any{"name": "k"}, "t1")
	var created struct {
		ID     string `json:"id"`
		RawKey string `json:"rawKey"`
	}
	decodeJSON(t, rr1, &created)

	hash := keygen.HashRaw(created.RawKey)
	repo.mu.Lock()
	repo.byHash[hash] = created.ID
	repo.mu.Unlock()

	// Revoke the key.
	do(t, h, http.MethodDelete, "/v1/keys/"+created.ID, nil, "t1")

	rr2 := do(t, h, http.MethodPost, "/v1/keys/validate", map[string]any{"key": created.RawKey}, "")
	if rr2.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 for revoked key", rr2.Code)
	}
}

func TestValidateKey_MissingKey(t *testing.T) {
	h, _ := newHandler(t)
	rr := do(t, h, http.MethodPost, "/v1/keys/validate", map[string]any{}, "")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

//  Health ─

func TestHealth_OK(t *testing.T) {
	h, _ := newHandler(t)
	rr := do(t, h, http.MethodGet, "/health", nil, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
}
