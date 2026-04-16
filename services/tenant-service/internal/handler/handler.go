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

// Package handler contains the HTTP layer for the tenant-service.  It depends
// only on the repository interface and the domain types — never on pgx or any
// other infrastructure package — so handlers can be unit-tested with in-memory
// fakes.
package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/SoftLaneIT/serviceforge/packages/go-common/logger"
	"github.com/SoftLaneIT/serviceforge/services/tenant-service/internal/domain"
	"github.com/SoftLaneIT/serviceforge/services/tenant-service/internal/repository"
)

// Handler holds the dependencies shared across all HTTP handler methods.
type Handler struct {
	repo repository.Repository
	log  *slog.Logger
}

// New returns a Handler wired to the given repository and logger.
func New(repo repository.Repository, log *slog.Logger) *Handler {
	return &Handler{repo: repo, log: log}
}

// RegisterRoutes registers all tenant-service routes onto mux.
// Uses the Go 1.22 "METHOD /path/{param}" pattern syntax.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("POST /v1/tenants", h.CreateTenant)
	mux.HandleFunc("GET /v1/tenants", h.ListTenants)
	mux.HandleFunc("GET /v1/tenants/{id}", h.GetTenant)
	mux.HandleFunc("PATCH /v1/tenants/{id}", h.UpdateTenant)
	mux.HandleFunc("DELETE /v1/tenants/{id}", h.DeleteTenant)
}

//  Health

// Health returns 200 when the service and its database connection are healthy,
// 503 when the database is unreachable.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{"service": "tenant-service", "status": "ok", "db": "ok"}

	if err := h.repo.Ping(r.Context()); err != nil {
		logger.FromContext(r.Context()).Warn("db ping failed", slog.Any("error", err))
		resp["status"] = "degraded"
		resp["db"] = "unreachable"
		respondJSON(w, http.StatusServiceUnavailable, resp)
		return
	}
	respondJSON(w, http.StatusOK, resp)
}

//  CreateTenant ─

type createRequest struct {
	Name     string         `json:"name"`
	Slug     string         `json:"slug"`
	Plan     domain.Plan    `json:"plan"`
	Settings map[string]any `json:"settings"`
}

func (h *Handler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	params := domain.CreateParams{
		Name:     req.Name,
		Slug:     req.Slug,
		Plan:     req.Plan,
		Settings: req.Settings,
	}
	if err := params.Validate(); err != nil {
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	tenant, err := h.repo.Create(r.Context(), params)
	if err != nil {
		if errors.Is(err, domain.ErrSlugConflict) {
			respondError(w, http.StatusConflict, "slug already taken")
			return
		}
		log.Error("create tenant", slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	log.Info("tenant created", slog.String("tenant_id", tenant.ID), slog.String("slug", tenant.Slug))
	respondJSON(w, http.StatusCreated, tenant)
}

//  ListTenants

type listResponse struct {
	Data   []domain.Tenant `json:"data"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

func (h *Handler) ListTenants(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	q := r.URL.Query()

	f := repository.ListFilter{
		Limit:  parseIntDefault(q.Get("limit"), 20),
		Offset: parseIntDefault(q.Get("offset"), 0),
	}
	if raw := q.Get("status"); raw != "" {
		s := domain.Status(raw)
		if !s.Valid() {
			respondError(w, http.StatusBadRequest,
				"status must be one of active, suspended, deleted")
			return
		}
		f.Status = &s
	}

	result, err := h.repo.List(r.Context(), f)
	if err != nil {
		log.Error("list tenants", slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	respondJSON(w, http.StatusOK, listResponse{
		Data:   result.Tenants,
		Total:  result.Total,
		Limit:  f.Limit,
		Offset: f.Offset,
	})
}

//  GetTenant ─

// GetTenant resolves by UUID {id} path parameter.  If the caller passes
// ?by=slug the value of {id} is treated as a slug instead, enabling lookups
// like GET /v1/tenants/acme-corp?by=slug.
func (h *Handler) GetTenant(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	id := r.PathValue("id")

	var (
		tenant *domain.Tenant
		err    error
	)
	if r.URL.Query().Get("by") == "slug" {
		tenant, err = h.repo.GetBySlug(r.Context(), id)
	} else {
		tenant, err = h.repo.GetByID(r.Context(), id)
	}

	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, http.StatusNotFound, "tenant not found")
			return
		}
		log.Error("get tenant", slog.String("id", id), slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	respondJSON(w, http.StatusOK, tenant)
}

//  UpdateTenant ─

type updateRequest struct {
	Name     *string        `json:"name"`
	Plan     *domain.Plan   `json:"plan"`
	Settings map[string]any `json:"settings"`
}

func (h *Handler) UpdateTenant(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	id := r.PathValue("id")

	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	params := domain.UpdateParams{
		Name:     req.Name,
		Plan:     req.Plan,
		Settings: req.Settings,
	}
	if err := params.Validate(); err != nil {
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	tenant, err := h.repo.Update(r.Context(), id, params)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, http.StatusNotFound, "tenant not found")
			return
		}
		log.Error("update tenant", slog.String("id", id), slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	log.Info("tenant updated", slog.String("tenant_id", tenant.ID))
	respondJSON(w, http.StatusOK, tenant)
}

//  DeleteTenant ─

func (h *Handler) DeleteTenant(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	id := r.PathValue("id")

	if err := h.repo.Delete(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, http.StatusNotFound, "tenant not found")
			return
		}
		log.Error("delete tenant", slog.String("id", id), slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	log.Info("tenant deleted", slog.String("tenant_id", id))
	w.WriteHeader(http.StatusNoContent)
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

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 0 {
		return def
	}
	return v
}
