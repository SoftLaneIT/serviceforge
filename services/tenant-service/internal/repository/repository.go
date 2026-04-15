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

// Package repository defines the persistence port for the tenant-service and
// provides the PostgreSQL adapter that implements it.
package repository

import (
	"context"

	"github.com/SoftLaneIT/serviceforge/services/tenant-service/internal/domain"
)

// ListFilter controls pagination and optional status filtering for List queries.
type ListFilter struct {
	// Status restricts results to tenants with this lifecycle state.
	// nil returns all non-deleted tenants.
	Status *domain.Status
	// Limit caps the number of rows returned.  Values <= 0 or > 100 are
	// clamped to 20 and 100 respectively inside the implementation.
	Limit int
	// Offset skips this many rows for cursor-free pagination.
	Offset int
}

// ListResult wraps the tenant slice with the total count for building
// pagination envelopes in the handler.
type ListResult struct {
	Tenants []domain.Tenant
	Total   int
}

// Repository is the persistence port.  Every storage implementation (pgx,
// in-memory test double, etc.) must satisfy this interface.
type Repository interface {
	// Create inserts a new tenant and returns the persisted aggregate.
	// Returns domain.ErrSlugConflict if the slug is already taken.
	Create(ctx context.Context, p domain.CreateParams) (*domain.Tenant, error)

	// GetByID returns a non-deleted tenant by its UUID string.
	// Returns domain.ErrNotFound if absent or soft-deleted.
	GetByID(ctx context.Context, id string) (*domain.Tenant, error)

	// GetBySlug returns a non-deleted tenant by its unique slug.
	// Returns domain.ErrNotFound if absent or soft-deleted.
	GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error)

	// List returns a filtered, paginated slice of non-deleted tenants together
	// with the total row count (before pagination) for client-side paging.
	List(ctx context.Context, f ListFilter) (ListResult, error)

	// Update applies a partial update to the identified tenant and returns the
	// updated aggregate.  Returns domain.ErrNotFound if absent or soft-deleted.
	Update(ctx context.Context, id string, p domain.UpdateParams) (*domain.Tenant, error)

	// Delete soft-deletes the identified tenant (sets status=deleted, deleted_at=NOW()).
	// Returns domain.ErrNotFound if absent or already deleted.
	Delete(ctx context.Context, id string) error

	// Ping checks the health of the underlying datastore.  Used by the
	// /health endpoint to distinguish "service up" from "service + DB up".
	Ping(ctx context.Context) error
}
