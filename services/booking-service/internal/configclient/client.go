/*
 * Copyright (c) 2026, SoftlaneIT (https://softlaneit.com/) All Rights Reserved.
 *
 * SoftlaneIT licenses this file to you under the Apache License,
 * Version 2.0 (the "LICENSE"); you may not use this file except
 * in compliance with the LICENSE.
 */

// Package configclient provides a lightweight HTTP client for fetching and
// caching per-tenant module configuration from the config-service.
//
// Typed config structs mirror the JSON Schemas in module_schemas exactly so
// that strong compile-time safety is available inside the booking-service
// without taking a dependency on the config-service codebase.
//
// Design decisions:
//   - In-memory TTL cache (60 s default) per (tenantID, module) pair keeps
//     hot-path latency sub-millisecond after the first fetch.
//   - Graceful degradation: if the config-service is unreachable the client
//     returns safe, conservative defaults so the booking flow is never blocked
//     by a transient config-service outage.
//   - All exported types use json struct tags matching the camelCase field
//     names in the stored JSONB — no custom decoder needed.
package configclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ──
// Typed config structs (one per enforced module)
// ──

// BookingCfg mirrors the "booking" module JSON Schema.
type BookingCfg struct {
	SlotDurationMinutes int                `json:"slotDurationMinutes"`
	MaxBookingsPerDay   int                `json:"maxBookingsPerDay"`
	AdvanceBookingDays  int                `json:"advanceBookingDays"`
	AutoConfirm         bool               `json:"autoConfirm"`
	BufferMinutes       int                `json:"bufferMinutes"`
	CancellationPolicy  CancellationPolicy `json:"cancellationPolicy"`
}

// CancellationPolicy is the nested object inside BookingCfg.
type CancellationPolicy struct {
	FreeCancelHoursBefore int `json:"freeCancelHoursBefore"`
	RefundPercent         int `json:"refundPercent"`
}

// DaySchedule represents operating hours for a single day of the week.
type DaySchedule struct {
	Open      bool   `json:"open"`
	OpenTime  string `json:"openTime"`  // "HH:MM" 24-hour
	CloseTime string `json:"closeTime"` // "HH:MM" 24-hour
}

// BusinessHoursCfg mirrors the "business-hours" module JSON Schema.
type BusinessHoursCfg struct {
	Timezone                  string      `json:"timezone"`
	AllowBookingsOutsideHours bool        `json:"allowBookingsOutsideHours"`
	Monday                    DaySchedule `json:"monday"`
	Tuesday                   DaySchedule `json:"tuesday"`
	Wednesday                 DaySchedule `json:"wednesday"`
	Thursday                  DaySchedule `json:"thursday"`
	Friday                    DaySchedule `json:"friday"`
	Saturday                  DaySchedule `json:"saturday"`
	Sunday                    DaySchedule `json:"sunday"`
	BreakDurationMinutes      int         `json:"breakDurationMinutes"`
	BreakStartTime            string      `json:"breakStartTime"`
}

// QueueCfg mirrors the "queue" module JSON Schema (subset used by booking-service).
type QueueCfg struct {
	MaxSeats              int    `json:"maxSeats"`
	MaxConcurrentUsers    int    `json:"maxConcurrentUsers"`
	QueueType             string `json:"queueType"`
	OverflowBehaviour     string `json:"overflowBehaviour"`
	MaxWaitTimeSeconds    int    `json:"maxWaitTimeSeconds"`
	LockingTimeoutSeconds int    `json:"lockingTimeoutSeconds"`
}

// ──
// Safe defaults — returned when config-service is unreachable or a module is
// not configured for the tenant.  Chosen to be permissive so that a config
// outage does not break the booking flow.
// ──

func defaultBookingCfg() BookingCfg {
	return BookingCfg{
		SlotDurationMinutes: 30,
		MaxBookingsPerDay:   1000,
		AdvanceBookingDays:  365,
		AutoConfirm:         false,
		BufferMinutes:       0,
		CancellationPolicy: CancellationPolicy{
			FreeCancelHoursBefore: 24,
			RefundPercent:         100,
		},
	}
}

func defaultBusinessHoursCfg() BusinessHoursCfg {
	openDay := DaySchedule{Open: true, OpenTime: "00:00", CloseTime: "23:59"}
	closedDay := DaySchedule{Open: false, OpenTime: "00:00", CloseTime: "23:59"}
	return BusinessHoursCfg{
		Timezone:                  "UTC",
		AllowBookingsOutsideHours: true, // permissive default
		Monday:                    openDay,
		Tuesday:                   openDay,
		Wednesday:                 openDay,
		Thursday:                  openDay,
		Friday:                    openDay,
		Saturday:                  closedDay,
		Sunday:                    closedDay,
	}
}

func defaultQueueCfg() QueueCfg {
	return QueueCfg{
		MaxSeats:              10000,
		MaxConcurrentUsers:    10000,
		QueueType:             "fifo",
		OverflowBehaviour:     "wait",
		MaxWaitTimeSeconds:    0,
		LockingTimeoutSeconds: 60,
	}
}

// ──
// Cache
// ──

type cacheKey struct {
	tenantID string
	module   string
}

type cacheEntry struct {
	raw       map[string]any
	isDefault bool
	expiresAt time.Time
}

// ──
// Client
// ──

// Client fetches per-tenant module configuration from the config-service and
// caches the results in memory with a configurable TTL.
type Client struct {
	baseURL    string
	httpClient *http.Client
	ttl        time.Duration

	mu    sync.RWMutex
	cache map[cacheKey]cacheEntry
}

// New returns a Client that fetches from baseURL (e.g. "http://config-service:8085").
func New(baseURL string, ttl time.Duration) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 3 * time.Second},
		ttl:        ttl,
		cache:      make(map[cacheKey]cacheEntry),
	}
}

// configResponse is the shape returned by GET /v1/config/{module}.
type configResponse struct {
	Module    string         `json:"module"`
	TenantID  string         `json:"tenantId"`
	Config    map[string]any `json:"config"`
	IsDefault bool           `json:"isDefault"`
}

// fetchFull retrieves the full config response for (tenantID, module), using the
// cache when the entry is still fresh.  It returns both the raw config map and
// the isDefault flag so callers can decide whether to fall back to permissive
// defaults for modules the tenant has never explicitly configured.
func (c *Client) fetchFull(ctx context.Context, tenantID, module string) (map[string]any, bool, error) {
	key := cacheKey{tenantID: tenantID, module: module}

	// Fast-path: read from cache.
	c.mu.RLock()
	if entry, ok := c.cache[key]; ok && time.Now().Before(entry.expiresAt) {
		c.mu.RUnlock()
		return entry.raw, entry.isDefault, nil
	}
	c.mu.RUnlock()

	// Slow-path: call config-service.
	url := fmt.Sprintf("%s/v1/config/%s", c.baseURL, module)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, fmt.Errorf("configclient: build request: %w", err)
	}
	req.Header.Set("X-Tenant-ID", tenantID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("configclient: http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("configclient: unexpected status %d for module %s", resp.StatusCode, module)
	}

	var body configResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, false, fmt.Errorf("configclient: decode response: %w", err)
	}

	// Write to cache.
	c.mu.Lock()
	c.cache[key] = cacheEntry{raw: body.Config, isDefault: body.IsDefault, expiresAt: time.Now().Add(c.ttl)}
	c.mu.Unlock()

	return body.Config, body.IsDefault, nil
}

// fetch is a convenience wrapper that discards the isDefault flag.
func (c *Client) fetch(ctx context.Context, tenantID, module string) (map[string]any, error) {
	raw, _, err := c.fetchFull(ctx, tenantID, module)
	return raw, err
}

// decode re-marshals raw into dst using JSON round-trip so that struct tags
// handle all camelCase ↔ Go-field mapping automatically.
func decode(raw map[string]any, dst any) error {
	b, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("configclient: marshal: %w", err)
	}
	return json.Unmarshal(b, dst)
}

// ──
// Public typed accessors
// ──

// GetBookingConfig returns the booking module config for tenantID.
// Falls back to safe defaults on any error so the booking flow is not blocked.
func (c *Client) GetBookingConfig(ctx context.Context, tenantID string) BookingCfg {
	raw, err := c.fetch(ctx, tenantID, "booking")
	if err != nil {
		return defaultBookingCfg()
	}
	var cfg BookingCfg
	if err := decode(raw, &cfg); err != nil {
		return defaultBookingCfg()
	}
	// Fill zero-value fields with defaults so callers never see a zero MaxBookingsPerDay etc.
	if cfg.MaxBookingsPerDay == 0 {
		cfg.MaxBookingsPerDay = defaultBookingCfg().MaxBookingsPerDay
	}
	if cfg.AdvanceBookingDays == 0 {
		cfg.AdvanceBookingDays = defaultBookingCfg().AdvanceBookingDays
	}
	if cfg.SlotDurationMinutes == 0 {
		cfg.SlotDurationMinutes = defaultBookingCfg().SlotDurationMinutes
	}
	return cfg
}

// GetBusinessHoursConfig returns the business-hours module config for tenantID.
// Falls back to permissive defaults (all hours allowed) on any error.
//
// If the config-service indicates that the tenant is still using the module
// schema defaults (isDefault=true), this method also returns the permissive
// default rather than enforcing the schema's example hours.  Business-hours
// enforcement only activates once a tenant has explicitly saved their own
// schedule via PUT /v1/config/business-hours.
func (c *Client) GetBusinessHoursConfig(ctx context.Context, tenantID string) BusinessHoursCfg {
	raw, isDefault, err := c.fetchFull(ctx, tenantID, "business-hours")
	if err != nil || isDefault {
		// Either config-service unreachable OR tenant hasn't configured a
		// custom schedule — either way use the permissive in-process default.
		return defaultBusinessHoursCfg()
	}
	var cfg BusinessHoursCfg
	if err := decode(raw, &cfg); err != nil {
		return defaultBusinessHoursCfg()
	}
	if cfg.Timezone == "" {
		cfg.Timezone = "UTC"
	}
	return cfg
}

// GetQueueConfig returns the queue module config for tenantID.
func (c *Client) GetQueueConfig(ctx context.Context, tenantID string) QueueCfg {
	raw, err := c.fetch(ctx, tenantID, "queue")
	if err != nil {
		return defaultQueueCfg()
	}
	var cfg QueueCfg
	if err := decode(raw, &cfg); err != nil {
		return defaultQueueCfg()
	}
	if cfg.MaxSeats == 0 {
		cfg.MaxSeats = defaultQueueCfg().MaxSeats
	}
	return cfg
}

// Invalidate removes the cached entry for (tenantID, module) so the next
// fetch will hit the config-service.  Useful after a known config change.
func (c *Client) Invalidate(tenantID, module string) {
	c.mu.Lock()
	delete(c.cache, cacheKey{tenantID: tenantID, module: module})
	c.mu.Unlock()
}
