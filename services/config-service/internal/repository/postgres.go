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

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SoftLaneIT/serviceforge/services/config-service/internal/domain"
)

// PostgresRepo is the pgx-backed implementation of Repository.
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

// setTenantContext arms RLS for the given transaction.
func setTenantContext(ctx context.Context, tx pgx.Tx, tenantID string) error {
	_, err := tx.Exec(ctx, "SELECT set_tenant_context($1::uuid)", tenantID)
	return err
}

//  GetSchema

// GetSchema returns the latest (highest version) schema for module.
func (r *PostgresRepo) GetSchema(ctx context.Context, module string) (*domain.ModuleSchema, error) {
	const q = `
		SELECT module, version, schema, defaults, created_at
		FROM   module_schemas
		WHERE  module = $1
		ORDER  BY version DESC
		LIMIT  1`

	row := r.pool.QueryRow(ctx, q, module)
	return scanSchema(row)
}

//  ListModules ─

// ListModules returns the latest schema row per module.
func (r *PostgresRepo) ListModules(ctx context.Context) ([]domain.ModuleSchema, error) {
	const q = `
		SELECT DISTINCT ON (module) module, version, schema, defaults, created_at
		FROM   module_schemas
		ORDER  BY module, version DESC`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list modules: %w", err)
	}
	defer rows.Close()

	var out []domain.ModuleSchema
	for rows.Next() {
		s, err := scanSchema(rows)
		if err != nil {
			return nil, fmt.Errorf("scan module schema: %w", err)
		}
		out = append(out, *s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list modules rows: %w", err)
	}
	if out == nil {
		out = []domain.ModuleSchema{}
	}
	return out, nil
}

//  GetConfig

// GetConfig returns the tenant's config for module.
func (r *PostgresRepo) GetConfig(ctx context.Context, tenantID, module string) (*domain.ModuleConfig, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := setTenantContext(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("set tenant context: %w", err)
	}

	const q = `
		SELECT id::text, tenant_id::text, module, config, schema_version,
		       created_at, updated_at
		FROM   module_configs
		WHERE  tenant_id = $1::uuid AND module = $2`

	row := tx.QueryRow(ctx, q, tenantID, module)
	cfg, err := scanConfig(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get config: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit get config: %w", err)
	}
	return cfg, nil
}

//  UpsertConfig

// UpsertConfig creates or updates the tenant's module config and appends a
// history row, all within a single transaction.
func (r *PostgresRepo) UpsertConfig(ctx context.Context, tenantID, module string, cfg map[string]any, schemaVersion int, changedBy string) (*domain.ModuleConfig, error) {
	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := setTenantContext(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("set tenant context: %w", err)
	}

	const upsertQ = `
		INSERT INTO module_configs (tenant_id, module, config, schema_version)
		VALUES ($1::uuid, $2, $3, $4)
		ON CONFLICT (tenant_id, module) DO UPDATE
		SET config = EXCLUDED.config,
		    schema_version = EXCLUDED.schema_version,
		    updated_at = NOW()
		RETURNING id::text, tenant_id::text, module, config, schema_version,
		          created_at, updated_at`

	row := tx.QueryRow(ctx, upsertQ, tenantID, module, cfgJSON, schemaVersion)
	result, err := scanConfig(row)
	if err != nil {
		return nil, fmt.Errorf("upsert config: %w", err)
	}

	// Append history entry.
	const histQ = `
		INSERT INTO config_history (config_id, tenant_id, module, config, changed_by)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5)`

	if _, err := tx.Exec(ctx, histQ, result.ID, tenantID, module, cfgJSON, changedBy); err != nil {
		return nil, fmt.Errorf("insert config history: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit upsert config: %w", err)
	}
	return result, nil
}

//  ListHistory ─

// ListHistory returns the most recent limit history rows for tenant + module.
func (r *PostgresRepo) ListHistory(ctx context.Context, tenantID, module string, limit int) ([]domain.ConfigHistory, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
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
		SELECT id::text, config_id::text, tenant_id::text, module,
		       config, changed_by, changed_at
		FROM   config_history
		WHERE  tenant_id = $1::uuid AND module = $2
		ORDER  BY changed_at DESC
		LIMIT  $3`

	rows, err := tx.Query(ctx, q, tenantID, module, limit)
	if err != nil {
		return nil, fmt.Errorf("list config history: %w", err)
	}
	defer rows.Close()

	var out []domain.ConfigHistory
	for rows.Next() {
		h, err := scanHistory(rows)
		if err != nil {
			return nil, fmt.Errorf("scan history row: %w", err)
		}
		out = append(out, *h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list history rows: %w", err)
	}
	if out == nil {
		out = []domain.ConfigHistory{}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit list history: %w", err)
	}
	return out, nil
}

//  scan helpers ─

type scanner interface{ Scan(dest ...any) error }

func scanSchema(s scanner) (*domain.ModuleSchema, error) {
	var (
		ms           domain.ModuleSchema
		schemaJSON   []byte
		defaultsJSON []byte
	)
	if err := s.Scan(&ms.Module, &ms.Version, &schemaJSON, &defaultsJSON, &ms.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrModuleNotFound
		}
		return nil, err
	}
	if err := json.Unmarshal(schemaJSON, &ms.Schema); err != nil {
		return nil, fmt.Errorf("unmarshal schema: %w", err)
	}
	if err := json.Unmarshal(defaultsJSON, &ms.Defaults); err != nil {
		return nil, fmt.Errorf("unmarshal defaults: %w", err)
	}
	return &ms, nil
}

func scanConfig(s scanner) (*domain.ModuleConfig, error) {
	var (
		mc      domain.ModuleConfig
		cfgJSON []byte
	)
	if err := s.Scan(&mc.ID, &mc.TenantID, &mc.Module, &cfgJSON, &mc.SchemaVersion, &mc.CreatedAt, &mc.UpdatedAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(cfgJSON, &mc.Config); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	if mc.Config == nil {
		mc.Config = map[string]any{}
	}
	return &mc, nil
}

func scanHistory(s scanner) (*domain.ConfigHistory, error) {
	var (
		h       domain.ConfigHistory
		cfgJSON []byte
	)
	if err := s.Scan(&h.ID, &h.ConfigID, &h.TenantID, &h.Module, &cfgJSON, &h.ChangedBy, &h.ChangedAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(cfgJSON, &h.Config); err != nil {
		return nil, fmt.Errorf("unmarshal history config: %w", err)
	}
	if h.Config == nil {
		h.Config = map[string]any{}
	}
	return &h, nil
}
