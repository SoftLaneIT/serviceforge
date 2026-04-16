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

// Package repository defines the storage contract for the config-service.
package repository

import (
	"context"

	"github.com/SoftLaneIT/serviceforge/services/config-service/internal/domain"
)

// Repository is the persistence contract for the config-service.
type Repository interface {
	// GetSchema returns the latest schema for the given module.
	GetSchema(ctx context.Context, module string) (*domain.ModuleSchema, error)

	// ListModules returns all modules that have a schema defined.
	ListModules(ctx context.Context) ([]domain.ModuleSchema, error)

	// GetConfig returns the tenant's config for module, or ErrNotFound if
	// the tenant has not yet set a config (they should start from Defaults).
	GetConfig(ctx context.Context, tenantID, module string) (*domain.ModuleConfig, error)

	// UpsertConfig creates or updates the tenant's config for module.
	// It also appends a config_history row inside the same transaction.
	// changedBy identifies the caller (e.g. the key ID from the auth middleware).
	UpsertConfig(ctx context.Context, tenantID, module string, cfg map[string]any, schemaVersion int, changedBy string) (*domain.ModuleConfig, error)

	// ListHistory returns the change history for the tenant's module config,
	// newest first, up to limit rows.
	ListHistory(ctx context.Context, tenantID, module string, limit int) ([]domain.ConfigHistory, error)

	// Ping verifies the underlying connection is alive.
	Ping(ctx context.Context) error
}
