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

// Package domain defines the Booking aggregate and business rules.
// No external dependencies.
package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Status is the lifecycle state of a booking.
type Status string

const (
	StatusPending   Status = "pending"
	StatusConfirmed Status = "confirmed"
	StatusCompleted Status = "completed"
	StatusCancelled Status = "cancelled"
	StatusNoShow    Status = "no_show"
)

// Valid reports whether s is a recognised status value.
func (s Status) Valid() bool {
	switch s {
	case StatusPending, StatusConfirmed, StatusCompleted, StatusCancelled, StatusNoShow:
		return true
	}
	return false
}

// Terminal reports whether the status is a terminal state (no further
// transitions allowed).
func (s Status) Terminal() bool {
	switch s {
	case StatusCompleted, StatusCancelled, StatusNoShow:
		return true
	}
	return false
}

// Booking is the root aggregate for the booking-service.
type Booking struct {
	ID          string         `json:"id"`
	TenantID    string         `json:"tenantId"`
	CustomerRef string         `json:"customerRef"`
	ServiceRef  string         `json:"serviceRef"`
	SlotStart   time.Time      `json:"slotStart"`
	SlotEnd     time.Time      `json:"slotEnd"`
	Status      Status         `json:"status"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	CancelledAt *time.Time     `json:"cancelledAt,omitempty"`
}

// CreateParams is the validated input for creating a new booking.
type CreateParams struct {
	CustomerRef string
	ServiceRef  string
	SlotStart   time.Time
	SlotEnd     time.Time
	Metadata    map[string]any
}

// Validate returns a combined error if any field is invalid.
func (p CreateParams) Validate() error {
	var errs []string

	if strings.TrimSpace(p.CustomerRef) == "" {
		errs = append(errs, "customerRef is required")
	}
	if strings.TrimSpace(p.ServiceRef) == "" {
		errs = append(errs, "serviceRef is required")
	}
	if p.SlotStart.IsZero() {
		errs = append(errs, "slotStart is required")
	}
	if p.SlotEnd.IsZero() {
		errs = append(errs, "slotEnd is required")
	}
	if !p.SlotStart.IsZero() && !p.SlotEnd.IsZero() && !p.SlotEnd.After(p.SlotStart) {
		errs = append(errs, "slotEnd must be after slotStart")
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation: %s", strings.Join(errs, "; "))
	}
	return nil
}

// Sentinel errors.
var (
	ErrNotFound       = errors.New("booking not found")
	ErrSlotConflict   = errors.New("time slot already booked")
	ErrTerminalStatus = errors.New("booking is in a terminal state")
)
