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

package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SoftLaneIT/serviceforge/services/booking-service/internal/domain"
)

const pgUniqueViolation = "23505"

// PostgresRepo is the pgx-backed implementation of Repository.
//
// Every mutating method wraps its work in a transaction and calls
// set_tenant_context() as the first statement.  This arms the Row-Level
// Security policy on the bookings table so that only rows belonging to the
// current tenant are visible and writable.
type PostgresRepo struct {
	pool *pgxpool.Pool
}

// NewPostgres returns a PostgresRepo backed by pool.
func NewPostgres(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

// Ping delegates to the pool.
func (r *PostgresRepo) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

// columns returned by every SELECT / RETURNING in this file.
const selectCols = `
	id::text, tenant_id::text, customer_ref, service_ref,
	slot_start, slot_end, status, metadata,
	created_at, updated_at, cancelled_at`

// setTenantContext executes set_tenant_context() within tx to arm RLS.
func setTenantContext(ctx context.Context, tx pgx.Tx, tenantID string) error {
	_, err := tx.Exec(ctx, "SELECT set_tenant_context($1::uuid)", tenantID)
	return err
}

// Create inserts a new booking inside a transaction with RLS armed.
// If p.InitialStatus is set it overrides the DB default ("pending"); this is
// used by the handler to honour the autoConfirm booking-module policy.
func (r *PostgresRepo) Create(ctx context.Context, tenantID string, p domain.CreateParams) (*domain.Booking, error) {
	settingsJSON, err := marshalMetadata(p.Metadata)
	if err != nil {
		return nil, err
	}

	// Determine initial status.
	initialStatus := string(domain.StatusPending)
	if p.InitialStatus != "" {
		initialStatus = string(p.InitialStatus)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := setTenantContext(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("set tenant context: %w", err)
	}

	const q = `
		INSERT INTO bookings
			(tenant_id, customer_ref, service_ref, slot_start, slot_end, status, metadata)
		VALUES
			($1::uuid, $2, $3, $4, $5, $6, $7)
		RETURNING ` + selectCols

	row := tx.QueryRow(ctx, q,
		tenantID,
		p.CustomerRef,
		p.ServiceRef,
		p.SlotStart,
		p.SlotEnd,
		initialStatus,
		settingsJSON,
	)

	b, err := scanBooking(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return nil, domain.ErrSlotConflict
		}
		return nil, fmt.Errorf("create booking: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create booking: %w", err)
	}
	return b, nil
}

// CountForDate returns the number of active bookings whose slot_start falls on
// the same UTC calendar day as date.  Used for the maxBookingsPerDay policy.
func (r *PostgresRepo) CountForDate(ctx context.Context, tenantID string, date time.Time) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := setTenantContext(ctx, tx, tenantID); err != nil {
		return 0, fmt.Errorf("set tenant context: %w", err)
	}

	const q = `
		SELECT COUNT(*)
		FROM   bookings
		WHERE  DATE(slot_start AT TIME ZONE 'UTC') = DATE($1 AT TIME ZONE 'UTC')
		AND    status NOT IN ('cancelled', 'no_show')`

	var count int
	if err := tx.QueryRow(ctx, q, date).Scan(&count); err != nil {
		return 0, fmt.Errorf("count bookings for date: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit count for date: %w", err)
	}
	return count, nil
}

// Update replaces the editable fields of an existing booking.  Status is
// intentionally not changed here — use UpdateStatus / Cancel for that.
func (r *PostgresRepo) Update(ctx context.Context, tenantID, id string, p domain.UpdateParams) (*domain.Booking, error) {
	existing, err := r.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if existing.Status.Terminal() {
		return nil, domain.ErrTerminalStatus
	}

	metaJSON, err := marshalMetadata(p.Metadata)
	if err != nil {
		return nil, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := setTenantContext(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("set tenant context: %w", err)
	}

	const q = `
		UPDATE bookings
		SET    customer_ref = $2,
		       service_ref  = $3,
		       slot_start   = $4,
		       slot_end     = $5,
		       metadata     = $6,
		       updated_at   = NOW()
		WHERE  id = $1::uuid
		RETURNING ` + selectCols

	row := tx.QueryRow(ctx, q, id, p.CustomerRef, p.ServiceRef, p.SlotStart, p.SlotEnd, metaJSON)
	b, err := scanBooking(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return nil, domain.ErrSlotConflict
		}
		return nil, fmt.Errorf("update booking: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit update booking: %w", err)
	}
	return b, nil
}

// HasBufferConflict returns true when any active booking for the same
// tenantID + serviceRef overlaps with the expanded window:
//
//	[slotStart − bufferMinutes, slotEnd + bufferMinutes]
//
// excludeID exempts a specific booking ID from the check (pass the booking's
// own ID when re-validating an edit so it doesn't conflict with itself).
// A bufferMinutes value of 0 makes this a plain overlap check.
func (r *PostgresRepo) HasBufferConflict(
	ctx context.Context,
	tenantID, serviceRef, excludeID string,
	slotStart, slotEnd time.Time,
	bufferMinutes int,
) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := setTenantContext(ctx, tx, tenantID); err != nil {
		return false, fmt.Errorf("set tenant context: %w", err)
	}

	// An existing booking B conflicts when:
	//   B.slot_end   > slotStart − buffer  (B finishes too close to the new start)
	//   B.slot_start < slotEnd   + buffer  (B starts too close to the new end)
	// excludeID (if non-empty) exempts the booking being edited.
	const q = `
		SELECT COUNT(*) FROM bookings
		WHERE  service_ref = $1
		AND    status NOT IN ('cancelled', 'no_show')
		AND    ($5 = '' OR id::text != $5)
		AND    slot_end   > $2::timestamptz - ($3 * interval '1 minute')
		AND    slot_start < $4::timestamptz + ($3 * interval '1 minute')`

	var count int
	if err := tx.QueryRow(ctx, q, serviceRef, slotStart, bufferMinutes, slotEnd, excludeID).Scan(&count); err != nil {
		return false, fmt.Errorf("buffer conflict check: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit buffer check: %w", err)
	}
	return count > 0, nil
}

// GetByID returns the booking if it exists for tenantID (RLS enforces the
// tenant restriction; ErrNotFound is returned for rows belonging to other
// tenants just as for non-existent rows).
func (r *PostgresRepo) GetByID(ctx context.Context, tenantID, id string) (*domain.Booking, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := setTenantContext(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("set tenant context: %w", err)
	}

	const q = `
		SELECT ` + selectCols + `
		FROM   bookings
		WHERE  id = $1::uuid`

	row := tx.QueryRow(ctx, q, id)
	b, err := scanBooking(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get booking by id: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit get booking: %w", err)
	}
	return b, nil
}

// List returns a paginated page of bookings for tenantID.
func (r *PostgresRepo) List(ctx context.Context, tenantID string, f ListFilter) (ListResult, error) {
	limit := f.Limit
	switch {
	case limit <= 0:
		limit = 20
	case limit > 100:
		limit = 100
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	var statusArg *string
	if f.Status != nil {
		s := string(*f.Status)
		statusArg = &s
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ListResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := setTenantContext(ctx, tx, tenantID); err != nil {
		return ListResult{}, fmt.Errorf("set tenant context: %w", err)
	}

	const countQ = `
		SELECT COUNT(*)
		FROM   bookings
		WHERE  ($1::text IS NULL OR status = $1)`

	var total int
	if err := tx.QueryRow(ctx, countQ, statusArg).Scan(&total); err != nil {
		return ListResult{}, fmt.Errorf("count bookings: %w", err)
	}

	const listQ = `
		SELECT ` + selectCols + `
		FROM   bookings
		WHERE  ($1::text IS NULL OR status = $1)
		ORDER  BY slot_start ASC
		LIMIT  $2 OFFSET $3`

	rows, err := tx.Query(ctx, listQ, statusArg, limit, offset)
	if err != nil {
		return ListResult{}, fmt.Errorf("list bookings: %w", err)
	}
	defer rows.Close()

	var bookings []domain.Booking
	for rows.Next() {
		b, err := scanBooking(rows)
		if err != nil {
			return ListResult{}, fmt.Errorf("scan booking row: %w", err)
		}
		bookings = append(bookings, *b)
	}
	if err := rows.Err(); err != nil {
		return ListResult{}, fmt.Errorf("list bookings rows: %w", err)
	}
	if bookings == nil {
		bookings = []domain.Booking{}
	}

	if err := tx.Commit(ctx); err != nil {
		return ListResult{}, fmt.Errorf("commit list bookings: %w", err)
	}
	return ListResult{Bookings: bookings, Total: total}, nil
}

// UpdateStatus transitions the booking to the new status.
func (r *PostgresRepo) UpdateStatus(ctx context.Context, tenantID, id string, status domain.Status) (*domain.Booking, error) {
	// Read current status first to guard against terminal-state transitions.
	existing, err := r.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if existing.Status.Terminal() {
		return nil, domain.ErrTerminalStatus
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := setTenantContext(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("set tenant context: %w", err)
	}

	const q = `
		UPDATE bookings
		SET    status = $2, updated_at = NOW()
		WHERE  id = $1::uuid
		RETURNING ` + selectCols

	row := tx.QueryRow(ctx, q, id, string(status))
	b, err := scanBooking(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("update booking status: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit update status: %w", err)
	}
	return b, nil
}

// Cancel sets status=cancelled and cancelled_at=NOW().
func (r *PostgresRepo) Cancel(ctx context.Context, tenantID, id string) (*domain.Booking, error) {
	existing, err := r.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if existing.Status.Terminal() {
		return nil, domain.ErrTerminalStatus
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := setTenantContext(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("set tenant context: %w", err)
	}

	const q = `
		UPDATE bookings
		SET    status = 'cancelled', cancelled_at = NOW(), updated_at = NOW()
		WHERE  id = $1::uuid
		RETURNING ` + selectCols

	row := tx.QueryRow(ctx, q, id)
	b, err := scanBooking(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("cancel booking: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit cancel: %w", err)
	}
	return b, nil
}

//  scan helpers ─

type scanner interface{ Scan(dest ...any) error }

func scanBooking(s scanner) (*domain.Booking, error) {
	var (
		b            domain.Booking
		status       string
		metadataJSON []byte
		cancelledAt  pgtype.Timestamptz
	)

	if err := s.Scan(
		&b.ID,
		&b.TenantID,
		&b.CustomerRef,
		&b.ServiceRef,
		&b.SlotStart,
		&b.SlotEnd,
		&status,
		&metadataJSON,
		&b.CreatedAt,
		&b.UpdatedAt,
		&cancelledAt,
	); err != nil {
		return nil, err
	}

	b.Status = domain.Status(status)

	if len(metadataJSON) > 0 && string(metadataJSON) != "null" {
		if err := json.Unmarshal(metadataJSON, &b.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}
	}
	if b.Metadata == nil {
		b.Metadata = map[string]any{}
	}

	if cancelledAt.Valid {
		t := cancelledAt.Time
		b.CancelledAt = &t
	}

	return &b, nil
}

func marshalMetadata(m map[string]any) ([]byte, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}
	return b, nil
}
