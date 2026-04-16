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

// Package repository defines the storage contract for auth-service API keys.
package repository

import (
	"context"

	"github.com/SoftLaneIT/serviceforge/services/auth-service/internal/domain"
)

// Repository is the persistence contract for the auth-service.
// Implementations must be safe for concurrent use.
type Repository interface {
	// Create inserts a new API key row.  The RawKey field in CreateParams is
	// never passed here — only the hash and prefix derived from it.
	Create(ctx context.Context, p domain.CreateParams) (*domain.APIKey, error)

	// GetByID returns the key visible to tenantID, or ErrNotFound.
	GetByID(ctx context.Context, id, tenantID string) (*domain.APIKey, error)

	// List returns all non-revoked keys that belong to tenantID.
	List(ctx context.Context, tenantID string) ([]domain.APIKey, error)

	// Revoke soft-deletes a key by setting status=revoked and revoked_at=NOW().
	// Returns ErrNotFound if the key does not belong to tenantID or is already
	// revoked.
	Revoke(ctx context.Context, id, tenantID string) error

	// GetByHash looks up a key by its SHA-256 hash regardless of tenant.
	// Used on the validate hot-path before the cache is populated.
	// Returns ErrInvalidKey if not found or already revoked.
	// Returns ErrExpiredKey if found but expires_at has passed.
	GetByHash(ctx context.Context, hash string) (*domain.APIKey, error)

	// UpdateLastUsed records the current timestamp on the key row.  This is
	// called asynchronously (best-effort) after a successful validation so it
	// never blocks the response path.
	UpdateLastUsed(ctx context.Context, id string) error

	// Ping verifies the underlying connection is alive.
	Ping(ctx context.Context) error
}
