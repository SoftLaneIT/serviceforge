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
// only on the repository interface, the events.Publisher interface, the config
// client, and domain types — never on pgx or kafka-go directly.
package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/SoftLaneIT/serviceforge/packages/go-common/logger"
	"github.com/SoftLaneIT/serviceforge/packages/go-common/tenant"
	"github.com/SoftLaneIT/serviceforge/services/booking-service/internal/configclient"
	"github.com/SoftLaneIT/serviceforge/services/booking-service/internal/domain"
	"github.com/SoftLaneIT/serviceforge/services/booking-service/internal/events"
	"github.com/SoftLaneIT/serviceforge/services/booking-service/internal/repository"
)

// Handler holds shared dependencies.
type Handler struct {
	repo repository.Repository
	pub  events.Publisher
	cfg  *configclient.Client
	log  *slog.Logger
}

// New returns a Handler wired to the given repository, publisher, config client, and logger.
func New(repo repository.Repository, pub events.Publisher, cfg *configclient.Client, log *slog.Logger) *Handler {
	return &Handler{repo: repo, pub: pub, cfg: cfg, log: log}
}

// RegisterRoutes registers all booking-service routes onto mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("POST /v1/bookings", h.CreateBooking)
	mux.HandleFunc("GET /v1/bookings", h.ListBookings)
	mux.HandleFunc("GET /v1/bookings/{id}", h.GetBooking)
	mux.HandleFunc("PUT /v1/bookings/{id}", h.UpdateBooking)
	mux.HandleFunc("PATCH /v1/bookings/{id}", h.UpdateStatus)
	mux.HandleFunc("DELETE /v1/bookings/{id}", h.CancelBooking)
}

// ──
// Health
// ──

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

// ──
// CreateBooking — policy enforcement pipeline
// ──

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

	// ── Fetch tenant configuration ─────────────────
	// Both fetches degrade gracefully to permissive defaults if config-service
	// is unavailable, so a config outage never breaks the booking flow.
	bookCfg := h.cfg.GetBookingConfig(r.Context(), tenantID)
	bizCfg := h.cfg.GetBusinessHoursConfig(r.Context(), tenantID)

	// ── Policy: advance booking limit ──────────────
	maxFuture := time.Now().UTC().AddDate(0, 0, bookCfg.AdvanceBookingDays)
	if req.SlotStart.After(maxFuture) {
		respondError(w, http.StatusUnprocessableEntity, fmt.Sprintf(
			"slot is too far in the future: bookings may only be created up to %d day(s) ahead",
			bookCfg.AdvanceBookingDays,
		))
		return
	}

	// ── Policy: slot must not be in the past ───────
	if req.SlotStart.Before(time.Now().UTC()) {
		respondError(w, http.StatusUnprocessableEntity, "slot start must be in the future")
		return
	}

	// ── Policy: slot must meet the minimum configured duration ─────────────────
	slotMinutes := int(req.SlotEnd.Sub(req.SlotStart).Minutes())
	if bookCfg.SlotDurationMinutes > 0 && slotMinutes < bookCfg.SlotDurationMinutes {
		respondError(w, http.StatusUnprocessableEntity, fmt.Sprintf(
			"slot duration (%d min) is shorter than the minimum slot duration configured for this tenant (%d min)",
			slotMinutes, bookCfg.SlotDurationMinutes,
		))
		return
	}

	// ── Policy: business hours ─────────────────────
	if !bizCfg.AllowBookingsOutsideHours {
		if err := checkBusinessHours(bizCfg, req.SlotStart, req.SlotEnd); err != nil {
			respondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
	}

	// ── Policy: max bookings per day ───────────────
	dayCount, err := h.repo.CountForDate(r.Context(), tenantID, req.SlotStart)
	if err != nil {
		log.Error("count bookings for date", slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if dayCount >= bookCfg.MaxBookingsPerDay {
		respondError(w, http.StatusConflict, fmt.Sprintf(
			"daily booking limit reached (%d/%d) — no further bookings accepted on this date",
			dayCount, bookCfg.MaxBookingsPerDay,
		))
		return
	}

	// ── Policy: buffer between consecutive bookings
	if bookCfg.BufferMinutes > 0 {
		conflict, err := h.repo.HasBufferConflict(
			r.Context(), tenantID, req.ServiceRef, "", // "" = no exclusion for new bookings
			req.SlotStart, req.SlotEnd, bookCfg.BufferMinutes,
		)
		if err != nil {
			log.Error("buffer conflict check", slog.Any("error", err))
			respondError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if conflict {
			respondError(w, http.StatusConflict, fmt.Sprintf(
				"a %d-minute buffer is required between consecutive bookings for this service",
				bookCfg.BufferMinutes,
			))
			return
		}
	}

	// ── Policy: autoConfirm ────────────────────────
	if bookCfg.AutoConfirm {
		params.InitialStatus = domain.StatusConfirmed
	}

	// ── Persist ───────────
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

	go h.publishCreated(booking)

	log.Info("booking created",
		slog.String("booking_id", booking.ID),
		slog.String("tenant_id", tenantID),
		slog.String("status", string(booking.Status)),
		slog.Bool("auto_confirmed", bookCfg.AutoConfirm),
	)
	respondJSON(w, http.StatusCreated, booking)
}

// ─────────────────────────────────────────────────────────────────────────────
// UpdateBooking — full edit of slot times, refs, and metadata
// ─────────────────────────────────────────────────────────────────────────────

type updateBookingRequest struct {
	CustomerRef string         `json:"customerRef"`
	ServiceRef  string         `json:"serviceRef"`
	SlotStart   time.Time      `json:"slotStart"`
	SlotEnd     time.Time      `json:"slotEnd"`
	Metadata    map[string]any `json:"metadata"`
}

func (h *Handler) UpdateBooking(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	tenantID := tenant.FromContext(r.Context())
	if tenantID == "" || tenantID == tenant.DefaultTenant {
		respondError(w, http.StatusBadRequest, "X-Tenant-ID header is required")
		return
	}

	id := r.PathValue("id")

	var req updateBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	params := domain.UpdateParams{
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

	// ── Fetch tenant configuration ──────────────────────────────────────────
	bookCfg := h.cfg.GetBookingConfig(r.Context(), tenantID)
	bizCfg := h.cfg.GetBusinessHoursConfig(r.Context(), tenantID)

	// ── Policy: advance booking limit ───────────────────────────────────────
	maxFuture := time.Now().UTC().AddDate(0, 0, bookCfg.AdvanceBookingDays)
	if req.SlotStart.After(maxFuture) {
		respondError(w, http.StatusUnprocessableEntity, fmt.Sprintf(
			"slot is too far in the future: bookings may only be created up to %d day(s) ahead",
			bookCfg.AdvanceBookingDays,
		))
		return
	}

	// ── Policy: minimum slot duration ───────────────────────────────────────
	slotMinutes := int(req.SlotEnd.Sub(req.SlotStart).Minutes())
	if bookCfg.SlotDurationMinutes > 0 && slotMinutes < bookCfg.SlotDurationMinutes {
		respondError(w, http.StatusUnprocessableEntity, fmt.Sprintf(
			"slot duration (%d min) is shorter than the minimum slot duration configured for this tenant (%d min)",
			slotMinutes, bookCfg.SlotDurationMinutes,
		))
		return
	}

	// ── Policy: business hours ──────────────────────────────────────────────
	if !bizCfg.AllowBookingsOutsideHours {
		if err := checkBusinessHours(bizCfg, req.SlotStart, req.SlotEnd); err != nil {
			respondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
	}

	// ── Policy: buffer (exclude self so it doesn't conflict with its old slot)
	if bookCfg.BufferMinutes > 0 {
		conflict, err := h.repo.HasBufferConflict(
			r.Context(), tenantID, req.ServiceRef, id,
			req.SlotStart, req.SlotEnd, bookCfg.BufferMinutes,
		)
		if err != nil {
			log.Error("buffer conflict check (update)", slog.Any("error", err))
			respondError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if conflict {
			respondError(w, http.StatusConflict, fmt.Sprintf(
				"a %d-minute buffer is required between consecutive bookings for this service",
				bookCfg.BufferMinutes,
			))
			return
		}
	}

	// ── Persist ─────────────────────────────────────────────────────────────
	booking, err := h.repo.Update(r.Context(), tenantID, id, params)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			respondError(w, http.StatusNotFound, "booking not found")
		case errors.Is(err, domain.ErrTerminalStatus):
			respondError(w, http.StatusConflict, "cannot edit a booking in a terminal state")
		case errors.Is(err, domain.ErrSlotConflict):
			respondError(w, http.StatusConflict, "time slot already booked")
		default:
			log.Error("update booking", slog.String("id", id), slog.Any("error", err))
			respondError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	go h.publishUpdated(booking)

	log.Info("booking updated",
		slog.String("booking_id", id),
		slog.String("tenant_id", tenantID),
	)
	respondJSON(w, http.StatusOK, booking)
}

// ──
// ListBookings
// ──

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

// ──
// GetBooking
// ──

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

// ──
// UpdateStatus
// ──

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

// ──
// CancelBooking
// ──

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

// ──
// Business-hours enforcement helper
// ──

// checkBusinessHours returns a descriptive error when slotStart or slotEnd
// falls outside the tenant's configured operating hours for that day.
//
// For multi-day bookings the start and end are checked against their respective
// day schedules independently, so a booking that genuinely spans into the next
// day is not incorrectly rejected by comparing the end time to the start day's
// closing time.
func checkBusinessHours(cfg configclient.BusinessHoursCfg, slotStart, slotEnd time.Time) error {
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		loc = time.UTC // unknown timezone → allow
	}

	start := slotStart.In(loc)
	end := slotEnd.In(loc)

	// ── Check the start day
	startDayName := strings.ToLower(start.Weekday().String())
	startSched := dayScheduleFor(cfg, startDayName)

	if !startSched.Open {
		return fmt.Errorf("business is closed on %s", startDayName)
	}

	sy, smo, sd := start.Date()
	openH, openM := parseHHMM(startSched.OpenTime)
	openTime := time.Date(sy, smo, sd, openH, openM, 0, 0, loc)

	if start.Before(openTime) {
		return fmt.Errorf("slot starts before opening time (%s)", startSched.OpenTime)
	}

	// ── Check the end time against the end day's schedule ───────────────────
	// Using the end day (not start day) means multi-day bookings are evaluated
	// correctly; the close time is anchored to the day on which the slot ends.
	endDayName := strings.ToLower(end.Weekday().String())
	endSched := dayScheduleFor(cfg, endDayName)

	ey, emo, ed := end.Date()
	closeH, closeM := parseHHMM(endSched.CloseTime)
	closeTime := time.Date(ey, emo, ed, closeH, closeM, 0, 0, loc)

	if end.After(closeTime) {
		return fmt.Errorf("slot ends after closing time (%s)", endSched.CloseTime)
	}

	// ── Break-time check (same-day bookings only) ───
	startDate := time.Date(sy, smo, sd, 0, 0, 0, 0, loc)
	endDate := time.Date(ey, emo, ed, 0, 0, 0, 0, loc)
	if startDate.Equal(endDate) && cfg.BreakDurationMinutes > 0 && cfg.BreakStartTime != "" {
		breakH, breakM := parseHHMM(cfg.BreakStartTime)
		breakStart := time.Date(sy, smo, sd, breakH, breakM, 0, 0, loc)
		breakEnd := breakStart.Add(time.Duration(cfg.BreakDurationMinutes) * time.Minute)

		// The slot overlaps the break if it starts before the break ends
		// AND ends after the break starts.
		if start.Before(breakEnd) && end.After(breakStart) {
			return fmt.Errorf("slot overlaps with the business break (%s – %s min)",
				cfg.BreakStartTime, strconv.Itoa(cfg.BreakDurationMinutes))
		}
	}

	return nil
}

// dayScheduleFor returns the DaySchedule for the given lowercase weekday name.
func dayScheduleFor(cfg configclient.BusinessHoursCfg, day string) configclient.DaySchedule {
	switch day {
	case "monday":
		return cfg.Monday
	case "tuesday":
		return cfg.Tuesday
	case "wednesday":
		return cfg.Wednesday
	case "thursday":
		return cfg.Thursday
	case "friday":
		return cfg.Friday
	case "saturday":
		return cfg.Saturday
	case "sunday":
		return cfg.Sunday
	default:
		return configclient.DaySchedule{Open: true, OpenTime: "00:00", CloseTime: "23:59"}
	}
}

// parseHHMM parses an "HH:MM" string into hours and minutes.
// Returns 0, 0 on any parse error (permissive).
func parseHHMM(s string) (int, int) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return 0, 0
	}
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	return h, m
}

// ──
// Event helpers
// ──

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

// ──
// HTTP helpers
// ──

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
