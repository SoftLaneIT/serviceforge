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

// Package handler contains the HTTP layer for the booking-service.  It depends
// only on the repository interface, the events.Publisher interface, and domain
// types — never on pgx or kafka-go directly.
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
	"github.com/SoftLaneIT/serviceforge/services/booking-service/internal/domain"
	"github.com/SoftLaneIT/serviceforge/services/booking-service/internal/events"
	"github.com/SoftLaneIT/serviceforge/services/booking-service/internal/repository"
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

// RegisterRoutes registers all booking-service routes onto mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("POST /v1/bookings", h.CreateBooking)
	mux.HandleFunc("GET /v1/bookings", h.ListBookings)
	mux.HandleFunc("GET /v1/bookings/{id}", h.GetBooking)
	mux.HandleFunc("PATCH /v1/bookings/{id}", h.UpdateStatus)
	mux.HandleFunc("DELETE /v1/bookings/{id}", h.CancelBooking)
}

// ── Health ──────

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{"service": "booking-service", "status": "ok", "db": "ok"}
	if err := h.repo.Ping(r.Context()); err != nil {
		logger.FromContext(r.Context()).Warn("db ping failed", slog.Any("error", err))
		resp["status"] = "degraded"
		resp["db"] = "unreachable"
		respondJSON(w, http.StatusServiceUnavailable, resp)
		return
	}
	respondJSON(w, http.StatusOK, resp)
}

// ── CreateBooking ─────────────────────────────────────────────────────────────

type createRequest struct {
	CustomerRef string         `json:"customerRef"`
	ServiceRef  string         `json:"serviceRef"`
	SlotStart   time.Time      `json:"slotStart"`
	SlotEnd     time.Time      `json:"slotEnd"`
	Metadata    map[string]any `json:"metadata"`
}

func (h *Handler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	tenantID := tenant.FromContext(r.Context())
	if tenantID == "" || tenantID == tenant.DefaultTenant {
		respondError(w, http.StatusBadRequest, "X-Tenant-ID header is required")
		return
	}

	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	params := domain.CreateParams{
		CustomerRef: req.CustomerRef,
		ServiceRef:  req.ServiceRef,
		SlotStart:   req.SlotStart,
		SlotEnd:     req.SlotEnd,
		Metadata:    req.Metadata,
	}
	if err := params.Validate(); err != nil {
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	booking, err := h.repo.Create(r.Context(), tenantID, params)
	if err != nil {
		if errors.Is(err, domain.ErrSlotConflict) {
			respondError(w, http.StatusConflict, "time slot already booked")
			return
		}
		log.Error("create booking", slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Publish booking.created event asynchronously so it never blocks the response.
	go h.publishCreated(booking)

	log.Info("booking created",
		slog.String("booking_id", booking.ID),
		slog.String("tenant_id", tenantID),
	)
	respondJSON(w, http.StatusCreated, booking)
}

// ── ListBookings

type listResponse struct {
	Data   []domain.Booking `json:"data"`
	Total  int              `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}

func (h *Handler) ListBookings(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	tenantID := tenant.FromContext(r.Context())
	if tenantID == "" || tenantID == tenant.DefaultTenant {
		respondError(w, http.StatusBadRequest, "X-Tenant-ID header is required")
		return
	}

	q := r.URL.Query()
	f := repository.ListFilter{
		Limit:  parseIntDefault(q.Get("limit"), 20),
		Offset: parseIntDefault(q.Get("offset"), 0),
	}
	if raw := q.Get("status"); raw != "" {
		s := domain.Status(raw)
		if !s.Valid() {
			respondError(w, http.StatusBadRequest,
				"status must be one of pending, confirmed, completed, cancelled, no_show")
			return
		}
		f.Status = &s
	}

	result, err := h.repo.List(r.Context(), tenantID, f)
	if err != nil {
		log.Error("list bookings", slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	respondJSON(w, http.StatusOK, listResponse{
		Data:   result.Bookings,
		Total:  result.Total,
		Limit:  f.Limit,
		Offset: f.Offset,
	})
}

// ── GetBooking ──

func (h *Handler) GetBooking(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	tenantID := tenant.FromContext(r.Context())
	if tenantID == "" || tenantID == tenant.DefaultTenant {
		respondError(w, http.StatusBadRequest, "X-Tenant-ID header is required")
		return
	}

	id := r.PathValue("id")
	booking, err := h.repo.GetByID(r.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, http.StatusNotFound, "booking not found")
			return
		}
		log.Error("get booking", slog.String("id", id), slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	respondJSON(w, http.StatusOK, booking)
}

// ── UpdateStatus

type updateStatusRequest struct {
	Status domain.Status `json:"status"`
}

func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	tenantID := tenant.FromContext(r.Context())
	if tenantID == "" || tenantID == tenant.DefaultTenant {
		respondError(w, http.StatusBadRequest, "X-Tenant-ID header is required")
		return
	}

	id := r.PathValue("id")

	var req updateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if !req.Status.Valid() {
		respondError(w, http.StatusUnprocessableEntity,
			"status must be one of pending, confirmed, completed, cancelled, no_show")
		return
	}

	booking, err := h.repo.UpdateStatus(r.Context(), tenantID, id, req.Status)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			respondError(w, http.StatusNotFound, "booking not found")
		case errors.Is(err, domain.ErrTerminalStatus):
			respondError(w, http.StatusConflict, "booking is in a terminal state")
		default:
			log.Error("update booking status", slog.String("id", id), slog.Any("error", err))
			respondError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	go h.publishUpdated(booking)

	log.Info("booking status updated",
		slog.String("booking_id", id),
		slog.String("status", string(req.Status)),
	)
	respondJSON(w, http.StatusOK, booking)
}

// ── CancelBooking ─────────────────────────────────────────────────────────────

func (h *Handler) CancelBooking(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	tenantID := tenant.FromContext(r.Context())
	if tenantID == "" || tenantID == tenant.DefaultTenant {
		respondError(w, http.StatusBadRequest, "X-Tenant-ID header is required")
		return
	}

	id := r.PathValue("id")
	booking, err := h.repo.Cancel(r.Context(), tenantID, id)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			respondError(w, http.StatusNotFound, "booking not found")
		case errors.Is(err, domain.ErrTerminalStatus):
			respondError(w, http.StatusConflict, "booking is in a terminal state")
		default:
			log.Error("cancel booking", slog.String("id", id), slog.Any("error", err))
			respondError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	go h.publishCancelled(booking)

	log.Info("booking cancelled", slog.String("booking_id", id))
	respondJSON(w, http.StatusOK, booking)
}

// ── event helpers ─────────────────────────────────────────────────────────────

// newEventID returns a random 32-hex-char string used as a unique event ID.
// Using crypto/rand avoids the google/uuid dependency.
func newEventID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (h *Handler) publishCreated(b *domain.Booking) {
	ev := events.BookingCreatedEvent{
		EventID:     newEventID(),
		EventType:   "booking.created",
		OccurredAt:  time.Now().UTC(),
		TenantID:    b.TenantID,
		BookingID:   b.ID,
		CustomerRef: b.CustomerRef,
		ServiceRef:  b.ServiceRef,
		SlotStart:   b.SlotStart,
		SlotEnd:     b.SlotEnd,
	}
	payload, err := json.Marshal(ev)
	if err != nil {
		h.log.Error("marshal booking.created event", slog.Any("error", err))
		return
	}
	ctx, cancel := timeoutCtx(5)
	defer cancel()
	if err := h.pub.Publish(ctx, events.TopicBookingCreated, []byte(b.TenantID), payload); err != nil {
		h.log.Error("publish booking.created", slog.String("booking_id", b.ID), slog.Any("error", err))
	}
}

func (h *Handler) publishCancelled(b *domain.Booking) {
	ev := events.BookingCancelledEvent{
		EventID:    newEventID(),
		EventType:  "booking.cancelled",
		OccurredAt: time.Now().UTC(),
		TenantID:   b.TenantID,
		BookingID:  b.ID,
	}
	payload, err := json.Marshal(ev)
	if err != nil {
		h.log.Error("marshal booking.cancelled event", slog.Any("error", err))
		return
	}
	ctx, cancel := timeoutCtx(5)
	defer cancel()
	if err := h.pub.Publish(ctx, events.TopicBookingCancelled, []byte(b.TenantID), payload); err != nil {
		h.log.Error("publish booking.cancelled", slog.String("booking_id", b.ID), slog.Any("error", err))
	}
}

func (h *Handler) publishUpdated(b *domain.Booking) {
	ev := events.BookingUpdatedEvent{
		EventID:    newEventID(),
		EventType:  "booking.updated",
		OccurredAt: time.Now().UTC(),
		TenantID:   b.TenantID,
		BookingID:  b.ID,
		NewStatus:  string(b.Status),
	}
	payload, err := json.Marshal(ev)
	if err != nil {
		h.log.Error("marshal booking.updated event", slog.Any("error", err))
		return
	}
	ctx, cancel := timeoutCtx(5)
	defer cancel()
	if err := h.pub.Publish(ctx, events.TopicBookingUpdated, []byte(b.TenantID), payload); err != nil {
		h.log.Error("publish booking.updated", slog.String("booking_id", b.ID), slog.Any("error", err))
	}
}

func timeoutCtx(secs int) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(secs)*time.Second)
}

// ── HTTP helpers

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
