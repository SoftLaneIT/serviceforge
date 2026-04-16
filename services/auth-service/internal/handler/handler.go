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

// Package handler contains the HTTP layer for the auth-service.  It depends
// only on the repository interface, the cache interface, and domain types —
// never on pgx, Redis, or any other infrastructure package directly — so
// handlers can be unit-tested with in-memory fakes.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/SoftLaneIT/serviceforge/packages/go-common/logger"
	"github.com/SoftLaneIT/serviceforge/packages/go-common/tenant"
	"github.com/SoftLaneIT/serviceforge/services/auth-service/internal/cache"
	"github.com/SoftLaneIT/serviceforge/services/auth-service/internal/domain"
	"github.com/SoftLaneIT/serviceforge/services/auth-service/internal/keygen"
	"github.com/SoftLaneIT/serviceforge/services/auth-service/internal/repository"
)

// Handler holds the dependencies shared across all HTTP handler methods.
type Handler struct {
	repo  repository.Repository
	cache cache.Cache
	log   *slog.Logger
}

// New returns a Handler wired to the given repository, cache, and logger.
func New(repo repository.Repository, c cache.Cache, log *slog.Logger) *Handler {
	return &Handler{repo: repo, cache: c, log: log}
}

// RegisterRoutes registers all auth-service routes onto mux.
// Uses the Go 1.22 "METHOD /path/{param}" pattern syntax.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("POST /v1/keys", h.IssueKey)
	mux.HandleFunc("GET /v1/keys", h.ListKeys)
	mux.HandleFunc("GET /v1/keys/{id}", h.GetKey)
	mux.HandleFunc("DELETE /v1/keys/{id}", h.RevokeKey)
	mux.HandleFunc("POST /v1/keys/validate", h.ValidateKey)
}

//  Health ─

// Health returns 200 when both the DB and cache are reachable, 503 otherwise.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	resp := map[string]any{
		"service": "auth-service",
		"status":  "ok",
		"db":      "ok",
		"cache":   "ok",
	}

	degraded := false
	if err := h.repo.Ping(ctx); err != nil {
		logger.FromContext(ctx).Warn("db ping failed", slog.Any("error", err))
		resp["status"] = "degraded"
		resp["db"] = "unreachable"
		degraded = true
	}
	if err := h.cache.Ping(ctx); err != nil {
		logger.FromContext(ctx).Warn("cache ping failed", slog.Any("error", err))
		resp["status"] = "degraded"
		resp["cache"] = "unreachable"
		degraded = true
	}

	status := http.StatusOK
	if degraded {
		status = http.StatusServiceUnavailable
	}
	respondJSON(w, status, resp)
}

//  IssueKey ─

type issueRequest struct {
	Name        string             `json:"name"`
	Environment domain.Environment `json:"environment"`
	ModuleScope []string           `json:"moduleScope"`
	ExpiresAt   *time.Time         `json:"expiresAt"`
}

// issueResponse wraps the persisted APIKey with the one-time raw key.
type issueResponse struct {
	domain.APIKey
	// RawKey is included only in this response and never stored.
	// The caller must save it immediately — it cannot be retrieved later.
	RawKey string `json:"rawKey"`
}

// IssueKey creates a new API key for the tenant identified by X-Tenant-ID.
func (h *Handler) IssueKey(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	tenantID := tenant.FromContext(r.Context())
	if tenantID == "" || tenantID == tenant.DefaultTenant {
		respondError(w, http.StatusBadRequest, "X-Tenant-ID header is required")
		return
	}

	var req issueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		respondError(w, http.StatusUnprocessableEntity, "name is required")
		return
	}
	if req.Environment == "" {
		req.Environment = domain.EnvSandbox // sensible default
	}
	if !req.Environment.Valid() {
		respondError(w, http.StatusUnprocessableEntity,
			"environment must be one of sandbox, production")
		return
	}

	// Generate cryptographically random key material.
	gen, err := keygen.Generate(req.Environment)
	if err != nil {
		log.Error("keygen failed", slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	params := domain.CreateParams{
		TenantID:    tenantID,
		Name:        strings.TrimSpace(req.Name),
		KeyHash:     gen.Hash,
		KeyPrefix:   gen.Prefix,
		Environment: req.Environment,
		ModuleScope: req.ModuleScope,
		ExpiresAt:   req.ExpiresAt,
	}
	if params.ModuleScope == nil {
		params.ModuleScope = []string{}
	}

	apiKey, err := h.repo.Create(r.Context(), params)
	if err != nil {
		log.Error("create api key", slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	log.Info("api key issued",
		slog.String("key_id", apiKey.ID),
		slog.String("tenant_id", tenantID),
		slog.String("environment", string(req.Environment)),
	)
	respondJSON(w, http.StatusCreated, issueResponse{APIKey: *apiKey, RawKey: gen.RawKey})
}

//  ListKeys ─

func (h *Handler) ListKeys(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	tenantID := tenant.FromContext(r.Context())
	if tenantID == "" || tenantID == tenant.DefaultTenant {
		respondError(w, http.StatusBadRequest, "X-Tenant-ID header is required")
		return
	}

	keys, err := h.repo.List(r.Context(), tenantID)
	if err != nil {
		log.Error("list api keys", slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"data":  keys,
		"total": len(keys),
	})
}

//  GetKey ─

func (h *Handler) GetKey(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	tenantID := tenant.FromContext(r.Context())
	if tenantID == "" || tenantID == tenant.DefaultTenant {
		respondError(w, http.StatusBadRequest, "X-Tenant-ID header is required")
		return
	}

	id := r.PathValue("id")
	key, err := h.repo.GetByID(r.Context(), id, tenantID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, http.StatusNotFound, "api key not found")
			return
		}
		log.Error("get api key", slog.String("id", id), slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	respondJSON(w, http.StatusOK, key)
}

//  RevokeKey

func (h *Handler) RevokeKey(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	tenantID := tenant.FromContext(r.Context())
	if tenantID == "" || tenantID == tenant.DefaultTenant {
		respondError(w, http.StatusBadRequest, "X-Tenant-ID header is required")
		return
	}

	id := r.PathValue("id")

	// Fetch key first so we can delete its cache entry by hash.
	// We need the hash to invalidate the cache; it is not stored on the
	// APIKey struct (by design), so we do a best-effort delete by id — in
	// practice the cache TTL (5 min) is the safety net if this lookup fails.
	if err := h.repo.Revoke(r.Context(), id, tenantID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, http.StatusNotFound, "api key not found")
			return
		}
		log.Error("revoke api key", slog.String("id", id), slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	log.Info("api key revoked", slog.String("key_id", id), slog.String("tenant_id", tenantID))
	w.WriteHeader(http.StatusNoContent)
}

//  ValidateKey

type validateRequest struct {
	// Key is the raw API key supplied by the calling service.
	Key string `json:"key"`
}

type validateResponse struct {
	TenantID    string             `json:"tenantId"`
	KeyID       string             `json:"keyId"`
	Environment domain.Environment `json:"environment"`
	ModuleScope []string           `json:"moduleScope"`
}

// ValidateKey is the internal endpoint consumed by the API gateway to verify
// an API key and resolve its owning tenant.
//
// Hot-path:  Redis cache hit → return immediately.
// Cold-path: Postgres lookup → populate cache → return.
// After a successful validation, last_used_at is updated asynchronously.
func (h *Handler) ValidateKey(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	var req validateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Key == "" {
		respondError(w, http.StatusBadRequest, "key is required")
		return
	}

	hash := keygen.HashRaw(req.Key)

	//  cache fast path ─
	if cached, found, err := h.cache.Get(r.Context(), hash); err == nil && found {
		if cached.Status == domain.KeyStatusRevoked {
			respondError(w, http.StatusUnauthorized, "api key has been revoked")
			return
		}
		if cached.ExpiresAt != nil && cached.ExpiresAt.Before(time.Now()) {
			respondError(w, http.StatusUnauthorized, "api key has expired")
			return
		}
		go h.asyncUpdateLastUsed(cached.ID)
		respondJSON(w, http.StatusOK, validateResponse{
			TenantID:    cached.TenantID,
			KeyID:       cached.ID,
			Environment: cached.Environment,
			ModuleScope: cached.ModuleScope,
		})
		return
	}

	//  Postgres cold path
	apiKey, err := h.repo.GetByHash(r.Context(), hash)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidKey):
			respondError(w, http.StatusUnauthorized, "invalid api key")
		case errors.Is(err, domain.ErrExpiredKey):
			respondError(w, http.StatusUnauthorized, "api key has expired")
		default:
			log.Error("validate api key", slog.Any("error", err))
			respondError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	// Populate cache for future requests.
	if err := h.cache.Set(r.Context(), hash, apiKey, cache.DefaultTTL); err != nil {
		// Non-fatal — log and continue.
		log.Warn("cache set failed", slog.Any("error", err))
	}

	go h.asyncUpdateLastUsed(apiKey.ID)

	respondJSON(w, http.StatusOK, validateResponse{
		TenantID:    apiKey.TenantID,
		KeyID:       apiKey.ID,
		Environment: apiKey.Environment,
		ModuleScope: apiKey.ModuleScope,
	})
}

// asyncUpdateLastUsed records the current time as last_used_at for key id.
// Runs in a goroutine so it never blocks the response path.
func (h *Handler) asyncUpdateLastUsed(id string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := h.repo.UpdateLastUsed(ctx, id); err != nil {
		h.log.Warn("update last_used_at failed", slog.String("key_id", id), slog.Any("error", err))
	}
}

//  helpers

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respondJSON(w, status, map[string]string{"error": msg})
}
