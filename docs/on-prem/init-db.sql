-- ============================================================================
-- ServiceForge Platform — Database Initialization
-- ============================================================================
-- This script runs once when PostgreSQL container starts for the first time.
-- It creates extensions, the base schema, and enables Row-Level Security.
-- ============================================================================

-- Required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";     -- Trigram index for full-text search

-- ── Tenants table (core of multi-tenancy) ──
CREATE TABLE IF NOT EXISTS tenants (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            VARCHAR(255) NOT NULL,
    slug            VARCHAR(100) UNIQUE NOT NULL,
    plan            VARCHAR(50) NOT NULL DEFAULT 'free',
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    settings        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── API Keys ──
CREATE TABLE IF NOT EXISTS api_keys (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    key_hash        VARCHAR(255) NOT NULL,       -- bcrypt hash of the key
    key_prefix      VARCHAR(12) NOT NULL,        -- First 8 chars for identification
    environment     VARCHAR(20) NOT NULL DEFAULT 'sandbox',  -- sandbox | production
    module_scope    TEXT[] DEFAULT '{}',          -- Empty = all modules
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    last_used_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_api_keys_tenant ON api_keys(tenant_id);
CREATE UNIQUE INDEX idx_api_keys_prefix ON api_keys(key_prefix);

-- ── Module Subscriptions ──
CREATE TABLE IF NOT EXISTS module_subscriptions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    module_name     VARCHAR(100) NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    config          JSONB NOT NULL DEFAULT '{}',
    subscribed_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, module_name)
);
CREATE INDEX idx_module_subs_tenant ON module_subscriptions(tenant_id);

-- ── Configuration History ──
CREATE TABLE IF NOT EXISTS config_history (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    module_name     VARCHAR(100) NOT NULL,
    config_before   JSONB,
    config_after    JSONB NOT NULL,
    changed_by      UUID,
    changed_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_config_history_tenant_module ON config_history(tenant_id, module_name, changed_at DESC);

-- ── Webhook Endpoints ──
CREATE TABLE IF NOT EXISTS webhooks (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    url             TEXT NOT NULL,
    events          TEXT[] NOT NULL DEFAULT '{}',
    secret          VARCHAR(255) NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    retry_policy    JSONB NOT NULL DEFAULT '{"max_retries": 5, "backoff": "exponential"}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_webhooks_tenant ON webhooks(tenant_id);

-- ── Enable Row-Level Security on all tenant-scoped tables ──
ALTER TABLE api_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE module_subscriptions ENABLE ROW LEVEL SECURITY;
ALTER TABLE config_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE webhooks ENABLE ROW LEVEL SECURITY;

-- RLS policies: each query must SET app.current_tenant_id before accessing data
CREATE POLICY tenant_isolation_api_keys ON api_keys
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

CREATE POLICY tenant_isolation_module_subs ON module_subscriptions
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

CREATE POLICY tenant_isolation_config_history ON config_history
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

CREATE POLICY tenant_isolation_webhooks ON webhooks
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

-- ── Application role (services connect as this, not as superuser) ──
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'sf_app') THEN
        CREATE ROLE sf_app LOGIN PASSWORD 'sf_app_password';
    END IF;
END $$;

GRANT USAGE ON SCHEMA public TO sf_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO sf_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO sf_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO sf_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO sf_app;

-- ── Utility function: set tenant context (called at start of each request) ──
CREATE OR REPLACE FUNCTION set_tenant_context(tid UUID) RETURNS VOID AS $$
BEGIN
    PERFORM set_config('app.current_tenant_id', tid::TEXT, TRUE);
END;
$$ LANGUAGE plpgsql;

-- ── Updated_at trigger function ──
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_tenants_updated
    BEFORE UPDATE ON tenants
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Log initialization
DO $$ BEGIN RAISE NOTICE 'ServiceForge database initialized successfully'; END $$;
