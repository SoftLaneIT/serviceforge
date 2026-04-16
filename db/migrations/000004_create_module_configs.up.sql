-- Copyright (c) 2026, SoftlaneIT (https://softlaneit.com/) All Rights Reserved.
--
-- SoftlaneIT licenses this file to you under the Apache License,
-- Version 2.0 (the "LICENSE"); you may not use this file except
-- in compliance with the LICENSE.
-- You may obtain a copy of the LICENSE at
--
-- https://softlaneit.com/LICENSE.txt
--
-- Unless required by applicable law or agreed to in writing,
-- software distributed under the LICENSE is distributed on an
-- "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
-- KIND, either express or implied.  See the LICENSE for the
-- specific language governing permissions and limitations
-- under the LICENSE.

-- 
-- 000004_create_module_configs.up
--
-- Three tables implement the configuration engine described in the architecture:
--
--   1. module_schemas  — JSON Schema registry (one row per module, platform-wide)
--      The config-service owns this table.  It stores the JSON Schema document
--      that defines valid configuration parameters for each module, the default
--      values, and the current schema version.  When a schema evolves, a new
--      version is inserted and the old one is retained for audit purposes.
--
--   2. module_configs  — Per-tenant, per-module configuration (the live config)
--      One row per (tenant, module) pair.  The `config` JSONB column holds the
--      developer's current settings, validated by the config-service against the
--      corresponding module_schemas.schema document before insertion or update.
--      The `schema_version` column records which schema version the stored
--      config was validated against — important when the schema is bumped.
--
--   3. config_history  — Immutable audit log of every config change
--      Append-only.  Never updated or deleted.  Enables "who changed what and
--      when" queries in the management UI and satisfies compliance requirements.
--
-- RLS strategy:
--   • module_schemas has NO RLS — it is a platform-wide registry, not tenant-
--     scoped.  The config-service reads it as a superuser / migration role.
--   • module_configs and config_history ARE RLS-protected so that tenant
--     context automatically scopes every query.
-- 

--  1. module_schemas — platform-wide JSON Schema registry ─
CREATE TABLE module_schemas (
    -- Canonical module identifier, lowercase (e.g. "booking", "payment").
    module          VARCHAR(100)  NOT NULL,
    -- Monotonically increasing version counter.  Increment on every breaking
    -- or additive schema change.
    version         INTEGER       NOT NULL DEFAULT 1,
    -- The full JSON Schema (draft-07) document describing valid config keys,
    -- types, min/max values, required fields, and UI rendering hints.
    schema          JSONB         NOT NULL,
    -- Default configuration values applied to new tenant subscriptions.
    -- Must conform to the schema above.
    defaults        JSONB         NOT NULL DEFAULT '{}',
    -- Human-readable description shown in the service catalog.
    description     TEXT,
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    PRIMARY KEY (module, version)
);

CREATE TRIGGER trg_module_schemas_updated_at
    BEFORE UPDATE ON module_schemas
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

--  2. module_configs — per-tenant live configuration ─
CREATE TABLE module_configs (
    id              UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID          NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    -- Must reference a known module in module_schemas.
    module          VARCHAR(100)  NOT NULL,
    -- Tenant's current configuration for this module.  Validated by the
    -- config-service before every write; stored raw here.
    config          JSONB         NOT NULL DEFAULT '{}',
    -- The schema version against which `config` was last validated.
    -- Used by config-service to detect when a tenant's config needs migration
    -- after a schema version bump.
    schema_version  INTEGER       NOT NULL DEFAULT 1,
    -- Whether this module subscription is active.
    status          VARCHAR(20)   NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active', 'suspended')),
    subscribed_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    -- Enforce one config row per (tenant, module) pair.
    CONSTRAINT uq_module_configs_tenant_module UNIQUE (tenant_id, module)
);

CREATE TRIGGER trg_module_configs_updated_at
    BEFORE UPDATE ON module_configs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

--  Indices for module_configs ─
-- config-service fetches a tenant's config for a specific module on every
-- request — this covers the hot path.
CREATE INDEX idx_module_configs_tenant_module
    ON module_configs (tenant_id, module);

--  RLS for module_configs 
ALTER TABLE module_configs ENABLE ROW LEVEL SECURITY;

CREATE POLICY module_configs_tenant_isolation ON module_configs
    USING      (tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID);

--  3. config_history — immutable audit log ─
CREATE TABLE config_history (
    id              UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID          NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    module          VARCHAR(100)  NOT NULL,
    -- NULL on the first subscription (no "before" state exists).
    config_before   JSONB,
    config_after    JSONB         NOT NULL,
    schema_version  INTEGER       NOT NULL DEFAULT 1,
    -- UUID of the management-UI user or service account that made the change.
    -- NULL if the change was made programmatically without an actor context.
    changed_by      UUID,
    changed_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW()
    -- Deliberately no updated_at — this table is append-only.
);

--  Indices for config_history ─
-- Primary access pattern: "show me all config changes for tenant X, module Y,
-- newest first".
CREATE INDEX idx_config_history_tenant_module_time
    ON config_history (tenant_id, module, changed_at DESC);

--  RLS for config_history 
ALTER TABLE config_history ENABLE ROW LEVEL SECURITY;

-- Read-only: a tenant can only see their own history.
-- config-service always inserts with the tenant context set, so the WITH CHECK
-- clause prevents accidentally writing a history row for the wrong tenant.
CREATE POLICY config_history_tenant_isolation ON config_history
    USING      (tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID);

-- 
-- Seed data: initial module schema for the booking module
--
-- This seed is part of the migration so that dev environments start with a
-- usable config-service without requiring a separate seed script.  Production
-- deployments also apply this migration and therefore get the same baseline.
-- 
INSERT INTO module_schemas (module, version, description, schema, defaults)
VALUES (
    'booking',
    1,
    'Slot-based appointment booking with configurable capacity, duration, and cancellation policy.',
    '{
        "$schema": "http://json-schema.org/draft-07/schema#",
        "title": "Booking Module Configuration",
        "type": "object",
        "additionalProperties": false,
        "properties": {
            "slotDurationMinutes": {
                "type": "integer",
                "minimum": 5,
                "maximum": 480,
                "description": "Duration of each bookable slot in minutes.",
                "x-ui": {"component": "slider", "step": 5}
            },
            "maxBookingsPerDay": {
                "type": "integer",
                "minimum": 1,
                "description": "Maximum bookings accepted per calendar day (UTC).",
                "x-ui": {"component": "number_input"}
            },
            "advanceBookingDays": {
                "type": "integer",
                "minimum": 1,
                "maximum": 365,
                "description": "How many days ahead a booking can be created.",
                "x-ui": {"component": "number_input"}
            },
            "autoConfirm": {
                "type": "boolean",
                "description": "If true, bookings move to confirmed automatically. If false, they require manual approval.",
                "x-ui": {"component": "toggle"}
            },
            "bufferMinutes": {
                "type": "integer",
                "minimum": 0,
                "maximum": 120,
                "description": "Gap required between consecutive bookings.",
                "x-ui": {"component": "slider", "step": 5}
            },
            "cancellationPolicy": {
                "type": "object",
                "additionalProperties": false,
                "properties": {
                    "freeCancelHoursBefore": {
                        "type": "integer",
                        "minimum": 0,
                        "description": "Hours before the slot start within which cancellation is free."
                    },
                    "refundPercent": {
                        "type": "integer",
                        "minimum": 0,
                        "maximum": 100,
                        "description": "Percentage of payment refunded on late cancellation."
                    }
                },
                "required": ["freeCancelHoursBefore", "refundPercent"],
                "x-ui": {"component": "nested_form"}
            }
        },
        "required": ["slotDurationMinutes", "maxBookingsPerDay", "autoConfirm"]
    }'::jsonb,
    '{
        "slotDurationMinutes": 30,
        "maxBookingsPerDay": 100,
        "advanceBookingDays": 30,
        "autoConfirm": true,
        "bufferMinutes": 0,
        "cancellationPolicy": {
            "freeCancelHoursBefore": 24,
            "refundPercent": 100
        }
    }'::jsonb
);
