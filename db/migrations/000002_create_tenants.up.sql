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
-- 000002_create_tenants.up
--
-- The tenants table is the root of the entire multi-tenancy hierarchy.
-- Every other tenant-scoped table has a tenant_id FK pointing here.
--
-- Design notes:
--   • `slug` is the human-readable, URL-safe tenant identifier used in API
--     paths and the management UI.  It is immutable after creation.
--   • `plan` controls which modules a tenant may subscribe to and what rate
--     limits and quotas apply.  Values are enforced by a CHECK constraint so
--     any attempt to insert an invalid plan fails at the DB layer regardless
--     of application logic.
--   • `status` follows a simple lifecycle: active → suspended → deleted.
--     Rows are never hard-deleted; deleted status satisfies audit requirements
--     while allowing foreign-key integrity to hold.
--   • `settings` is a free-form JSONB bag for platform-level settings that do
--     not warrant dedicated columns (e.g. custom branding colours, support
--     email).  Module-specific configuration lives in module_configs, not here.
--   • RLS is NOT enabled on tenants itself — tenant_service reads and writes
--     this table using the superuser / migration role and is the only service
--     authorised to do so.  All other services work through tenant_id FKs.
-- 

CREATE TABLE tenants (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        VARCHAR(255) NOT NULL,
    -- Immutable, URL-safe identifier: lowercase letters, digits, hyphens.
    -- Max 100 chars keeps it usable in DNS labels and S3 prefixes.
    slug        VARCHAR(100) NOT NULL,
    -- Billing plan determines quotas and available modules.
    plan        VARCHAR(20)  NOT NULL DEFAULT 'starter'
                    CHECK (plan IN ('starter', 'pro', 'enterprise')),
    -- Lifecycle state.
    status      VARCHAR(20)  NOT NULL DEFAULT 'active'
                    CHECK (status IN ('active', 'suspended', 'deleted')),
    -- Platform-level settings (branding, contact, misc flags).
    settings    JSONB        NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    -- Soft-delete timestamp; NULL means the tenant is not deleted.
    deleted_at  TIMESTAMPTZ
);

-- ── Constraints ───
-- Slugs must be unique among non-deleted tenants only.  A deleted tenant's
-- slug should be reusable so that an organisation can re-register after
-- closure.  A partial unique index achieves this cleanly.
CREATE UNIQUE INDEX uq_tenants_slug_active
    ON tenants (slug)
    WHERE deleted_at IS NULL;

-- ── Indices 
-- Management UI lists tenants filtered by status; this supports O(1) lookup.
CREATE INDEX idx_tenants_status ON tenants (status);

-- Trigram index on slug enables fast ILIKE searches in the management UI
-- ("find tenant whose slug contains 'acme'").
CREATE INDEX idx_tenants_slug_trgm ON tenants USING GIN (slug gin_trgm_ops);

-- ── Trigger 
CREATE TRIGGER trg_tenants_updated_at
    BEFORE UPDATE ON tenants
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();
