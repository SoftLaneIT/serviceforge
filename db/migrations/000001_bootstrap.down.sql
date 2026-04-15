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
-- 000001_bootstrap.down
--
-- Reverses the bootstrap migration.  This is intentionally conservative:
-- we drop functions and revoke grants but do NOT drop the extensions because
-- other databases on the same cluster may depend on them, and extension drops
-- require superuser rights that sf_app does not have.
-- 

REVOKE EXECUTE ON ALL FUNCTIONS  IN SCHEMA public FROM sf_app;
REVOKE SELECT, INSERT, UPDATE, DELETE ON ALL TABLES    IN SCHEMA public FROM sf_app;
REVOKE USAGE, SELECT              ON ALL SEQUENCES IN SCHEMA public FROM sf_app;
REVOKE USAGE  ON SCHEMA public FROM sf_app;
REVOKE CONNECT ON DATABASE serviceforge FROM sf_app;

DROP FUNCTION IF EXISTS set_tenant_context(UUID);
DROP FUNCTION IF EXISTS update_updated_at();

-- NOTE: We intentionally leave the sf_app role intact because dropping a role
-- requires that it owns no objects, which we cannot guarantee in a down migration.
-- Run `DROP ROLE sf_app;` manually after confirming no objects are owned.
