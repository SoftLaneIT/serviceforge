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

// Package handler contains the HTTP layer for the config-service.  It depends
// only on the repository interface, the validator, and domain types — never on
// pgx or kafka-go directly.
package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/SoftLaneIT/serviceforge/packages/go-common/logger"
	"github.com/SoftLaneIT/serviceforge/packages/go-common/tenant"
	"github.com/SoftLaneIT/serviceforge/services/config-service/internal/domain"
	"github.com/SoftLaneIT/serviceforge/services/config-service/internal/events"
	"github.com/SoftLaneIT/serviceforge/services/config-service/internal/repository"
	"github.com/SoftLaneIT/serviceforge/services/config-service/internal/validator"
)

// Handler holds shared dependencies.
type Handler struct {
	repo repository.Repository
	pub  events.Publisher
	log  *slog.Logger
}

// New returns a Handler wired to the given repository, publisher, and logger.
func New(repo repository.Repository, pub events.Publisher, log *slog.Logger) *Handler {
	return &Handler{repo: repo, pub: pub, log: log}
}

// RegisterRoutes registers all config-service routes onto mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /v1/modules", h.ListModules)
	mux.HandleFunc("GET /v1/modules/{module}/schema", h.GetSchema)
	mux.HandleFunc("GET /v1/config/{module}", h.GetConfig)
	mux.HandleFunc("PUT /v1/config/{module}", h.UpsertConfig)
	mux.HandleFunc("GET /v1/config/{module}/history", h.ListHistory)
}

//  Health ─

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{"service": "config-service", "status": "ok", "db": "ok"}
	if err := h.repo.Ping(r.Context()); err != nil {
		logger.FromContext(r.Context()).Warn("db ping failed", slog.Any("error", err))
		resp["status"] = "degraded"
		resp["db"] = "unreachable"
		respondJSON(w, http.StatusServiceUnavailable, resp)
		return
	}
	respondJSON(w, http.StatusOK, resp)
}

//  ListModules ─

func (h *Handler) ListModules(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	modules, err := h.repo.ListModules(r.Context())
	if err != nil {
		log.Error("list modules", slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": modules, "total": len(modules)})
}

//  GetSchema

func (h *Handler) GetSchema(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	module := r.PathValue("module")

	schema, err := h.repo.GetSchema(r.Context(), module)
	if err != nil {
		if errors.Is(err, domain.ErrModuleNotFound) {
			respondError(w, http.StatusNotFound, "module not found")
			return
		}
		log.Error("get schema", slog.String("module", module), slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	respondJSON(w, http.StatusOK, schema)
}

//  GetConfig

// GetConfig returns the tenant's current config for module.
// When the tenant has no saved config yet it falls back to the schema defaults.
func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	tenantID := tenant.FromContext(r.Context())
	if tenantID == "" || tenantID == tenant.DefaultTenant {
		respondError(w, http.StatusBadRequest, "X-Tenant-ID header is required")
		return
	}

	module := r.PathValue("module")

	cfg, err := h.repo.GetConfig(r.Context(), tenantID, module)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			// No config saved yet — return the schema defaults so the caller
			// knows what to start from.
			schema, schErr := h.repo.GetSchema(r.Context(), module)
			if schErr != nil {
				if errors.Is(schErr, domain.ErrModuleNotFound) {
					respondError(w, http.StatusNotFound, "module not found")
					return
				}
				log.Error("get schema for defaults", slog.String("module", module), slog.Any("error", schErr))
				respondError(w, http.StatusInternalServerError, "internal error")
				return
			}
			respondJSON(w, http.StatusOK, map[string]any{
				"tenantId":      tenantID,
				"module":        module,
				"config":        schema.Defaults,
				"schemaVersion": schema.Version,
				"isDefault":     true,
			})
			return
		}
		log.Error("get config", slog.String("module", module), slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	respondJSON(w, http.StatusOK, cfg)
}

//  UpsertConfig

func (h *Handler) UpsertConfig(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	tenantID := tenant.FromContext(r.Context())
	if tenantID == "" || tenantID == tenant.DefaultTenant {
		respondError(w, http.StatusBadRequest, "X-Tenant-ID header is required")
		return
	}

	module := r.PathValue("module")

	// Fetch the schema for this module so we can validate against it.
	schema, err := h.repo.GetSchema(r.Context(), module)
	if err != nil {
		if errors.Is(err, domain.ErrModuleNotFound) {
			respondError(w, http.StatusNotFound, "module not found")
			return
		}
		log.Error("get schema for validation", slog.String("module", module), slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Decode the incoming config.
	var incoming map[string]any
	if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Validate against the JSON Schema.
	if err := validator.Validate(schema.Schema, incoming); err != nil {
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	// changedBy defaults to "api"; the gateway can pass a richer identifier
	// via a custom header in a later increment.
	changedBy := r.Header.Get("X-Changed-By")
	if changedBy == "" {
		changedBy = "api"
	}

	result, err := h.repo.UpsertConfig(r.Context(), tenantID, module, incoming, schema.Version, changedBy)
	if err != nil {
		log.Error("upsert config", slog.String("module", module), slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	go h.publishUpdated(result)

	log.Info("config updated",
		slog.String("tenant_id", tenantID),
		slog.String("module", module),
	)
	respondJSON(w, http.StatusOK, result)
}

//  ListHistory ─

func (h *Handler) ListHistory(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	tenantID := tenant.FromContext(r.Context())
	if tenantID == "" || tenantID == tenant.DefaultTenant {
		respondError(w, http.StatusBadRequest, "X-Tenant-ID header is required")
		return
	}

	module := r.PathValue("module")
	limit := parseIntDefault(r.URL.Query().Get("limit"), 20)

	history, err := h.repo.ListHistory(r.Context(), tenantID, module, limit)
	if err != nil {
		log.Error("list config history", slog.String("module", module), slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": history, "total": len(history)})
}

//  event helpers ─

func (h *Handler) publishUpdated(cfg *domain.ModuleConfig) {
	ev := events.ConfigUpdatedEvent{
		EventID:       newEventID(),
		EventType:     "config.updated",
		OccurredAt:    time.Now().UTC(),
		TenantID:      cfg.TenantID,
		Module:        cfg.Module,
		SchemaVersion: cfg.SchemaVersion,
		Config:        cfg.Config,
	}
	payload, err := json.Marshal(ev)
	if err != nil {
		h.log.Error("marshal config.updated event", slog.Any("error", err))
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := h.pub.Publish(ctx, events.TopicConfigUpdated, []byte(cfg.TenantID), payload); err != nil {
		h.log.Error("publish config.updated",
			slog.String("tenant_id", cfg.TenantID),
			slog.String("module", cfg.Module),
			slog.Any("error", err),
		)
	}
}

func newEventID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

//  HTTP helpers

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respondJSON(w, status, map[string]string{"error": msg})
}

func parseIntDefault(s string, def int) int {
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return def
	}
	return v
}
