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

// Package repository defines the storage contract for the booking-service.
package repository

import (
	"context"

	"github.com/SoftLaneIT/serviceforge/services/booking-service/internal/domain"
)

// ListFilter controls pagination and optional status filtering.
type ListFilter struct {
	Status *domain.Status
	Limit  int
	Offset int
}

// ListResult bundles a page of bookings with the total unpaginated count.
type ListResult struct {
	Bookings []domain.Booking
	Total    int
}

// Repository is the persistence contract for the booking-service.
// Every mutating method accepts tenantID explicitly so the implementation can
// arm PostgreSQL Row-Level Security via set_tenant_context().
type Repository interface {
	// Create inserts a new booking.  Returns ErrSlotConflict when the
	// unique-slot index would be violated.
	Create(ctx context.Context, tenantID string, p domain.CreateParams) (*domain.Booking, error)

	// GetByID returns the booking if it belongs to tenantID, else ErrNotFound.
	GetByID(ctx context.Context, tenantID, id string) (*domain.Booking, error)

	// List returns a paginated page of bookings for tenantID.
	List(ctx context.Context, tenantID string, f ListFilter) (ListResult, error)

	// UpdateStatus transitions the booking to status.  Returns ErrNotFound if
	// the booking does not exist for tenantID, or ErrTerminalStatus if the
	// current status is already terminal.
	UpdateStatus(ctx context.Context, tenantID, id string, status domain.Status) (*domain.Booking, error)

	// Cancel soft-deletes the booking (status=cancelled, cancelled_at=NOW()).
	// Returns ErrNotFound or ErrTerminalStatus.
	Cancel(ctx context.Context, tenantID, id string) (*domain.Booking, error)

	// Ping verifies the underlying connection is alive.
	Ping(ctx context.Context) error
}
