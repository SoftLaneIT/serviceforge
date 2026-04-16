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
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SoftLaneIT/serviceforge/services/auth-service/internal/domain"
)

// PostgresRepo is the pgx-backed implementation of Repository.
type PostgresRepo struct {
	pool *pgxpool.Pool
}

// NewPostgres returns a PostgresRepo backed by pool.
func NewPostgres(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

// Ping delegates to the underlying pool.
func (r *PostgresRepo) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

// columns returned by every SELECT / RETURNING query in this file.
// Order must match the scan call in scanKey().
const selectCols = `
	id::text, tenant_id::text, name, key_prefix, environment,
	module_scope, status, last_used_at, expires_at, created_at, revoked_at`

// Create inserts a new API key and returns the persisted view (without the
// raw key or hash).
func (r *PostgresRepo) Create(ctx context.Context, p domain.CreateParams) (*domain.APIKey, error) {
	var expiresAt *time.Time
	if p.ExpiresAt != nil {
		expiresAt = p.ExpiresAt
	}

	const q = `
		INSERT INTO api_keys
			(tenant_id, name, key_hash, key_prefix, environment, module_scope, expires_at)
		VALUES
			($1::uuid, $2, $3, $4, $5, $6, $7)
		RETURNING ` + selectCols

	row := r.pool.QueryRow(ctx, q,
		p.TenantID,
		p.Name,
		p.KeyHash,
		p.KeyPrefix,
		string(p.Environment),
		p.ModuleScope,
		expiresAt,
	)

	k, err := scanKey(row)
	if err != nil {
		return nil, fmt.Errorf("create api key: %w", err)
	}
	return k, nil
}

// GetByID returns the key only when it belongs to tenantID.
func (r *PostgresRepo) GetByID(ctx context.Context, id, tenantID string) (*domain.APIKey, error) {
	const q = `
		SELECT ` + selectCols + `
		FROM   api_keys
		WHERE  id = $1::uuid AND tenant_id = $2::uuid`

	row := r.pool.QueryRow(ctx, q, id, tenantID)
	k, err := scanKey(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get api key by id: %w", err)
	}
	return k, nil
}

// List returns all keys (active and revoked) for tenantID, newest first.
func (r *PostgresRepo) List(ctx context.Context, tenantID string) ([]domain.APIKey, error) {
	const q = `
		SELECT ` + selectCols + `
		FROM   api_keys
		WHERE  tenant_id = $1::uuid
		ORDER  BY created_at DESC`

	rows, err := r.pool.Query(ctx, q, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()

	var keys []domain.APIKey
	for rows.Next() {
		k, err := scanKey(rows)
		if err != nil {
			return nil, fmt.Errorf("scan api key row: %w", err)
		}
		keys = append(keys, *k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list api keys rows: %w", err)
	}
	if keys == nil {
		keys = []domain.APIKey{}
	}
	return keys, nil
}

// Revoke sets status=revoked and revoked_at=NOW() for the key identified by
// id + tenantID.
func (r *PostgresRepo) Revoke(ctx context.Context, id, tenantID string) error {
	const q = `
		UPDATE api_keys
		SET    status = 'revoked', revoked_at = NOW()
		WHERE  id = $1::uuid AND tenant_id = $2::uuid AND status = 'active'`

	tag, err := r.pool.Exec(ctx, q, id, tenantID)
	if err != nil {
		return fmt.Errorf("revoke api key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// GetByHash looks up a key by its SHA-256 hash.
// Returns ErrInvalidKey when the hash does not exist or the key is revoked.
// Returns ErrExpiredKey when the key has a non-nil expires_at in the past.
func (r *PostgresRepo) GetByHash(ctx context.Context, hash string) (*domain.APIKey, error) {
	const q = `
		SELECT ` + selectCols + `
		FROM   api_keys
		WHERE  key_hash = $1`

	row := r.pool.QueryRow(ctx, q, hash)
	k, err := scanKey(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInvalidKey
		}
		return nil, fmt.Errorf("get api key by hash: %w", err)
	}

	if k.Status == domain.KeyStatusRevoked {
		return nil, domain.ErrInvalidKey
	}
	if k.ExpiresAt != nil && k.ExpiresAt.Before(time.Now()) {
		return nil, domain.ErrExpiredKey
	}
	return k, nil
}

// UpdateLastUsed sets last_used_at = NOW() for the given key id.  This is
// best-effort and typically called from a background goroutine.
func (r *PostgresRepo) UpdateLastUsed(ctx context.Context, id string) error {
	const q = `UPDATE api_keys SET last_used_at = NOW() WHERE id = $1::uuid`
	_, err := r.pool.Exec(ctx, q, id)
	return err
}

//  scan helpers

// scanner is satisfied by both pgx.Row and pgx.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanKey(s scanner) (*domain.APIKey, error) {
	var (
		k           domain.APIKey
		env         string
		status      string
		moduleScope []string
		lastUsedAt  pgtype.Timestamptz
		expiresAt   pgtype.Timestamptz
		revokedAt   pgtype.Timestamptz
	)

	if err := s.Scan(
		&k.ID,
		&k.TenantID,
		&k.Name,
		&k.KeyPrefix,
		&env,
		&moduleScope,
		&status,
		&lastUsedAt,
		&expiresAt,
		&k.CreatedAt,
		&revokedAt,
	); err != nil {
		return nil, err
	}

	k.Environment = domain.Environment(env)
	k.Status = domain.KeyStatus(status)
	k.ModuleScope = moduleScope
	if k.ModuleScope == nil {
		k.ModuleScope = []string{}
	}

	if lastUsedAt.Valid {
		t := lastUsedAt.Time
		k.LastUsedAt = &t
	}
	if expiresAt.Valid {
		t := expiresAt.Time
		k.ExpiresAt = &t
	}
	if revokedAt.Valid {
		t := revokedAt.Time
		k.RevokedAt = &t
	}

	return &k, nil
}
