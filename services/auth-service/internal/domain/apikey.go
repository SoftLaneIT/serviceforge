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

// Package domain defines the APIKey aggregate and the business rules that
// govern it.  This package has zero external dependencies.
package domain

import (
	"errors"
	"time"
)

// Environment is the deployment context for an API key.
type Environment string

const (
	EnvSandbox    Environment = "sandbox"
	EnvProduction Environment = "production"
)

// Valid reports whether e is a recognised environment value.
func (e Environment) Valid() bool {
	switch e {
	case EnvSandbox, EnvProduction:
		return true
	}
	return false
}

// KeyStatus is the lifecycle state of an API key.
type KeyStatus string

const (
	KeyStatusActive  KeyStatus = "active"
	KeyStatusRevoked KeyStatus = "revoked"
)

// Valid reports whether s is a recognised status value.
func (s KeyStatus) Valid() bool {
	switch s {
	case KeyStatusActive, KeyStatusRevoked:
		return true
	}
	return false
}

// APIKey is the view of an API key that is safe to return in list/get
// responses.  The raw key and key_hash are never included.
type APIKey struct {
	ID          string      `json:"id"`
	TenantID    string      `json:"tenantId"`
	Name        string      `json:"name"`
	KeyPrefix   string      `json:"keyPrefix"`
	Environment Environment `json:"environment"`
	ModuleScope []string    `json:"moduleScope"`
	Status      KeyStatus   `json:"status"`
	LastUsedAt  *time.Time  `json:"lastUsedAt,omitempty"`
	ExpiresAt   *time.Time  `json:"expiresAt,omitempty"`
	CreatedAt   time.Time   `json:"createdAt"`
	RevokedAt   *time.Time  `json:"revokedAt,omitempty"`
}

// CreateParams is the validated input for issuing a new API key.
// KeyHash and KeyPrefix are computed by the keygen package before calling
// the repository; they are not supplied directly by the HTTP caller.
type CreateParams struct {
	TenantID    string
	Name        string
	KeyHash     string      // SHA-256 hex of the raw key
	KeyPrefix   string      // first 12 chars of the raw key (display only)
	Environment Environment
	ModuleScope []string
	ExpiresAt   *time.Time
}

// Sentinel errors returned by the repository.
var (
	// ErrNotFound is returned when an API key does not exist for the given
	// tenant, or has already been revoked.
	ErrNotFound = errors.New("api key not found")

	// ErrInvalidKey is returned by the validate path when the supplied raw key
	// is not found in the store or is in the revoked state.
	ErrInvalidKey = errors.New("invalid or revoked api key")

	// ErrExpiredKey is returned when a key exists but its expires_at has passed.
	ErrExpiredKey = errors.New("api key has expired")
)
