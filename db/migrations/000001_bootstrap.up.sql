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
-- 000001_bootstrap.up
--
-- One-time cluster-level setup that every subsequent migration depends on:
--   • PostgreSQL extensions
--   • The update_updated_at() trigger function (shared by all tables with an
--     updated_at column — avoids duplicating it per-table)
--   • The set_tenant_context() helper (called by services at the start of
--     every DB transaction to arm Row-Level Security)
--   • The sf_app role (services connect as this role, never as the superuser)
-- 

--  Extensions 
-- uuid_generate_v4() is used as the default for all primary-key columns.
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- gen_salt / crypt for bcrypt-style hashing (used by auth-service for API
-- key storage as a defence-in-depth measure alongside SHA-256 lookup).
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Trigram index support — used for fast LIKE / similarity searches on
-- tenant slugs and customer references.
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

--  Shared trigger function ─
-- Automatically maintains the updated_at column on any table that attaches
-- this trigger.  Using a single shared function rather than per-table copies
-- keeps the schema DRY and avoids drift.
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

--  Tenant-context helper ─
-- Services call this at the beginning of every transaction (or connection pool
-- checkout) to set the GUC that RLS policies read.  Using a transaction-scoped
-- setting (is_local = TRUE) means the value is automatically cleared when the
-- transaction ends, so a pooled connection can never leak one tenant's context
-- into the next request.
--
-- Usage (from Go with pgx):
--   _, err = tx.Exec(ctx, "SELECT set_tenant_context($1)", tenantID)
CREATE OR REPLACE FUNCTION set_tenant_context(p_tenant_id UUID)
RETURNS VOID
LANGUAGE plpgsql
AS $$
BEGIN
    PERFORM set_config('app.current_tenant_id', p_tenant_id::TEXT, TRUE);
END;
$$;

--  Application role ─
-- All five services connect as sf_app — never as the database owner.  This
-- limits blast radius: a compromised service cannot DROP TABLE or ALTER ROLE.
-- The password is overridden in production via a secrets manager; this default
-- is safe only for local development.
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'sf_app') THEN
        CREATE ROLE sf_app LOGIN PASSWORD 'sf_app_dev_only';
    END IF;
END;
$$;

GRANT CONNECT ON DATABASE serviceforge TO sf_app;
GRANT USAGE   ON SCHEMA public TO sf_app;

-- Grant DML on all *existing* tables now and on all *future* tables created by
-- the migration runner (which runs as the owner, not as sf_app).
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES   IN SCHEMA public TO sf_app;
GRANT USAGE, SELECT                  ON ALL SEQUENCES IN SCHEMA public TO sf_app;

ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES    TO sf_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT USAGE, SELECT                  ON SEQUENCES TO sf_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT EXECUTE                        ON FUNCTIONS TO sf_app;
