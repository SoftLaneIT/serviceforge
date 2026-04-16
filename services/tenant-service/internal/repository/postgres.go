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
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SoftLaneIT/serviceforge/services/tenant-service/internal/domain"
)

// pgUniqueViolation is the PostgreSQL error code for a unique-constraint breach.
const pgUniqueViolation = "23505"

// PostgresRepo is the pgx-backed implementation of Repository.
// It requires that every DB call sets app.current_tenant_id via
// set_tenant_context() when RLS is relevant.  For the tenant-service itself
// (which reads/writes the tenants table without RLS), this is not necessary.
type PostgresRepo struct {
	pool *pgxpool.Pool
}

// NewPostgres creates a PostgresRepo backed by pool.  pool must already be
// connected; call Ping after construction to verify liveness.
func NewPostgres(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

// Ping delegates to the underlying pool.
func (r *PostgresRepo) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

//  Create

func (r *PostgresRepo) Create(ctx context.Context, p domain.CreateParams) (*domain.Tenant, error) {
	settingsJSON, err := marshalSettings(p.Settings)
	if err != nil {
		return nil, err
	}

	const q = `
		INSERT INTO tenants (name, slug, plan, settings)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, name, slug, plan, status, settings,
		          created_at, updated_at, deleted_at`

	row := r.pool.QueryRow(ctx, q,
		strings.TrimSpace(p.Name),
		strings.TrimSpace(p.Slug),
		string(p.Plan),
		settingsJSON,
	)
	t, err := scanTenant(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return nil, domain.ErrSlugConflict
		}
		return nil, fmt.Errorf("create tenant: %w", err)
	}
	return t, nil
}

//  GetByID

func (r *PostgresRepo) GetByID(ctx context.Context, id string) (*domain.Tenant, error) {
	const q = `
		SELECT id::text, name, slug, plan, status, settings,
		       created_at, updated_at, deleted_at
		FROM   tenants
		WHERE  id = $1::uuid AND deleted_at IS NULL`

	row := r.pool.QueryRow(ctx, q, id)
	t, err := scanTenant(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get tenant by id: %w", err)
	}
	return t, nil
}

//  GetBySlug ─

func (r *PostgresRepo) GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	const q = `
		SELECT id::text, name, slug, plan, status, settings,
		       created_at, updated_at, deleted_at
		FROM   tenants
		WHERE  slug = $1 AND deleted_at IS NULL`

	row := r.pool.QueryRow(ctx, q, slug)
	t, err := scanTenant(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get tenant by slug: %w", err)
	}
	return t, nil
}

//  List

func (r *PostgresRepo) List(ctx context.Context, f ListFilter) (ListResult, error) {
	// Clamp pagination bounds.
	limit := f.Limit
	switch {
	case limit <= 0:
		limit = 20
	case limit > 100:
		limit = 100
	}
	offset := max(f.Offset, 0)

	// Encode optional status filter as a nullable string so the SQL can use
	// ($1::text IS NULL OR status = $1) without branching on the Go side.
	var statusArg *string
	if f.Status != nil {
		s := string(*f.Status)
		statusArg = &s
	}

	// Total count (before pagination) — required for paging envelopes.
	const countQ = `
		SELECT COUNT(*)
		FROM   tenants
		WHERE  ($1::text IS NULL OR status = $1)
		  AND  deleted_at IS NULL`

	var total int
	if err := r.pool.QueryRow(ctx, countQ, statusArg).Scan(&total); err != nil {
		return ListResult{}, fmt.Errorf("count tenants: %w", err)
	}

	const listQ = `
		SELECT id::text, name, slug, plan, status, settings,
		       created_at, updated_at, deleted_at
		FROM   tenants
		WHERE  ($1::text IS NULL OR status = $1)
		  AND  deleted_at IS NULL
		ORDER  BY created_at DESC
		LIMIT  $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, listQ, statusArg, limit, offset)
	if err != nil {
		return ListResult{}, fmt.Errorf("list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []domain.Tenant
	for rows.Next() {
		t, err := scanTenantRow(rows)
		if err != nil {
			return ListResult{}, fmt.Errorf("scan tenant row: %w", err)
		}
		tenants = append(tenants, *t)
	}
	if err := rows.Err(); err != nil {
		return ListResult{}, fmt.Errorf("list tenants rows: %w", err)
	}

	// Ensure JSON array is never null for empty results.
	if tenants == nil {
		tenants = []domain.Tenant{}
	}
	return ListResult{Tenants: tenants, Total: total}, nil
}

//  Update

// Update builds a dynamic SET clause so that only the fields present in p are
// modified.  This prevents a PATCH from inadvertently clearing fields that
// were not included in the request body.
func (r *PostgresRepo) Update(ctx context.Context, id string, p domain.UpdateParams) (*domain.Tenant, error) {
	// If nothing is set, skip the DB round-trip.
	if p.Name == nil && p.Plan == nil && p.Settings == nil {
		return r.GetByID(ctx, id)
	}

	// $1 is always the tenant ID used in the WHERE clause.
	args := []any{id}
	setClauses := []string{"updated_at = NOW()"}

	if p.Name != nil {
		args = append(args, strings.TrimSpace(*p.Name))
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", len(args)))
	}
	if p.Plan != nil {
		args = append(args, string(*p.Plan))
		setClauses = append(setClauses, fmt.Sprintf("plan = $%d", len(args)))
	}
	if p.Settings != nil {
		settingsJSON, err := marshalSettings(p.Settings)
		if err != nil {
			return nil, err
		}
		args = append(args, settingsJSON)
		setClauses = append(setClauses, fmt.Sprintf("settings = $%d", len(args)))
	}

	q := fmt.Sprintf(`
		UPDATE tenants
		SET    %s
		WHERE  id = $1::uuid AND deleted_at IS NULL
		RETURNING id::text, name, slug, plan, status, settings,
		          created_at, updated_at, deleted_at`,
		strings.Join(setClauses, ", "))

	row := r.pool.QueryRow(ctx, q, args...)
	t, err := scanTenant(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("update tenant: %w", err)
	}
	return t, nil
}

//  Delete

func (r *PostgresRepo) Delete(ctx context.Context, id string) error {
	const q = `
		UPDATE tenants
		SET    status = 'deleted', deleted_at = NOW(), updated_at = NOW()
		WHERE  id = $1::uuid AND deleted_at IS NULL`

	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("soft-delete tenant: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

//  scan helpers ─

// scanner is satisfied by both pgx.Row (QueryRow) and pgx.Rows (Query) so
// that a single scan function handles both call sites.
type scanner interface {
	Scan(dest ...any) error
}

func scanTenant(row pgx.Row) (*domain.Tenant, error)      { return scan(row) }
func scanTenantRow(rows pgx.Rows) (*domain.Tenant, error) { return scan(rows) }

func scan(s scanner) (*domain.Tenant, error) {
	var (
		t            domain.Tenant
		plan         string
		status       string
		settingsJSON []byte
		deletedAt    pgtype.Timestamptz
	)

	if err := s.Scan(
		&t.ID,
		&t.Name,
		&t.Slug,
		&plan,
		&status,
		&settingsJSON,
		&t.CreatedAt,
		&t.UpdatedAt,
		&deletedAt,
	); err != nil {
		return nil, err
	}

	t.Plan = domain.Plan(plan)
	t.Status = domain.Status(status)

	if len(settingsJSON) > 0 && string(settingsJSON) != "null" {
		if err := json.Unmarshal(settingsJSON, &t.Settings); err != nil {
			return nil, fmt.Errorf("unmarshal settings: %w", err)
		}
	}
	if t.Settings == nil {
		t.Settings = map[string]any{}
	}

	if deletedAt.Valid {
		ts := deletedAt.Time
		t.DeletedAt = &ts
	}

	return &t, nil
}

//  internal helpers

func marshalSettings(s map[string]any) ([]byte, error) {
	if s == nil {
		return []byte("{}"), nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("marshal settings: %w", err)
	}
	return b, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
