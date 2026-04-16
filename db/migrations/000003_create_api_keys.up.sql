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
-- 000003_create_api_keys.up
--
-- API keys are the primary authentication credential for tenant applications.
-- A tenant may hold multiple keys (e.g. one per environment, one per service
-- integration).
--
-- Security model:
--   • The raw key is generated once by auth-service and shown to the developer
--     exactly once; it is never stored.
--   • `key_hash` stores a SHA-256 hex digest of the raw key.  Fast enough for
--     O(1) lookup by hash; no need for bcrypt-level cost here because keys are
--     high-entropy random strings (not passwords).
--   • `key_prefix` stores the first 8 characters of the raw key (e.g. "sf_live_")
--     for display in the management UI so developers can identify which key
--     they are looking at without exposing the full secret.
--
-- Rotation:
--   Keys are never mutated after creation.  To rotate, the developer creates a
--   new key, updates their application, then revokes the old key by setting
--   revoked_at.  This gives a zero-downtime rotation window.
--
-- Row-Level Security:
--   api_keys are scoped to a tenant.  The RLS policy enforces that a service
--   connected as sf_app can only see rows belonging to the tenant whose ID was
--   set via set_tenant_context().  The gateway's auth middleware bypasses RLS
--   during the key-lookup step by calling the function as a superuser; once the
--   tenant is identified it sets the context and all subsequent queries are
--   automatically scoped.
-- 

CREATE TABLE api_keys (
    id           UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id    UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    -- Human-readable label set by the developer (e.g. "mobile-app-prod").
    name         VARCHAR(255) NOT NULL,
    -- SHA-256 hex digest of the raw key.  Used for fast O(1) lookup.
    key_hash     CHAR(64)     NOT NULL,
    -- First 8 characters of the raw key for UI display (e.g. "sf_live_a").
    key_prefix   VARCHAR(12)  NOT NULL,
    -- Environment this key is valid in.
    environment  VARCHAR(20)  NOT NULL DEFAULT 'sandbox'
                    CHECK (environment IN ('sandbox', 'production')),
    -- Optional scope restriction: empty array = access to all subscribed modules.
    -- Non-empty = only the listed module names (e.g. '{"booking","payment"}').
    module_scope TEXT[]       NOT NULL DEFAULT '{}',
    -- Lifecycle state.  Revoked keys must be retained for audit purposes.
    status       VARCHAR(20)  NOT NULL DEFAULT 'active'
                    CHECK (status IN ('active', 'revoked')),
    -- Populated on successful authentication for anomaly detection.
    last_used_at TIMESTAMPTZ,
    -- Optional expiry.  NULL = no expiry.
    expires_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    -- Soft-revoke timestamp; NULL = key is not revoked.
    revoked_at   TIMESTAMPTZ,
    -- Ensure key_hash is unique across all tenants (prevents hash collisions
    -- from different tenants accidentally sharing a credential).
    CONSTRAINT uq_api_keys_hash UNIQUE (key_hash)
);

--  Indices 
-- The gateway performs key lookup by hash on every authenticated request;
-- this must be a fast index scan.
CREATE UNIQUE INDEX idx_api_keys_hash   ON api_keys (key_hash);

-- Lists all keys for a tenant in the management UI.
CREATE        INDEX idx_api_keys_tenant ON api_keys (tenant_id);

-- Prefix lookup for display deduplication.
CREATE UNIQUE INDEX idx_api_keys_prefix ON api_keys (key_prefix);

--  Row-Level Security ─
ALTER TABLE api_keys ENABLE ROW LEVEL SECURITY;

-- The SELECT / INSERT / UPDATE / DELETE policy:
--   • SELECT: a connection can only read keys that belong to its current tenant.
--   • INSERT: tenant_id must equal the current tenant (the CHECK option enforces
--     this on write so a service cannot accidentally insert a key for another
--     tenant).
--   • The gateway's superuser connection that performs the initial key-lookup
--     (before tenant context is known) must use SET LOCAL to temporarily bypass
--     RLS — it should do so only for the hash lookup, then immediately set the
--     tenant context for the rest of the transaction.
CREATE POLICY api_keys_tenant_isolation ON api_keys
    USING       (tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID)
    WITH CHECK  (tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID);

--  Trigger 
-- api_keys are immutable after creation; there is no updated_at column and
-- therefore no trigger needed here.
