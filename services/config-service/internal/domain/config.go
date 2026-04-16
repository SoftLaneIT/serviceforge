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

// Package domain defines the module-config aggregate and business rules.
// No external dependencies.
package domain

import (
	"errors"
	"time"
)

// ModuleSchema is the platform-level JSON Schema definition for a module.
// Stored in module_schemas; not tenant-scoped.
type ModuleSchema struct {
	Module    string         `json:"module"`
	Version   int            `json:"version"`
	Schema    map[string]any `json:"schema"`
	Defaults  map[string]any `json:"defaults"`
	CreatedAt time.Time      `json:"createdAt"`
}

// ModuleConfig is the tenant-specific configuration for one module.
// Stored in module_configs with RLS.
type ModuleConfig struct {
	ID            string         `json:"id"`
	TenantID      string         `json:"tenantId"`
	Module        string         `json:"module"`
	Config        map[string]any `json:"config"`
	SchemaVersion int            `json:"schemaVersion"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
}

// ConfigHistory is one row in the append-only audit log.
type ConfigHistory struct {
	ID        string         `json:"id"`
	TenantID  string         `json:"tenantId"`
	Module    string         `json:"module"`
	Config    map[string]any `json:"config"`
	ChangedBy string         `json:"changedBy"`
	ChangedAt time.Time      `json:"changedAt"`
}

// Sentinel errors.
var (
	ErrNotFound       = errors.New("config not found")
	ErrModuleNotFound = errors.New("module schema not found")
	ErrValidation     = errors.New("config validation failed")
)
