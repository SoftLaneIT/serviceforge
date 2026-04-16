-- Copyright (c) 2026, SoftlaneIT (https://softlaneit.com/) All Rights Reserved.
--
-- SoftlaneIT licenses this file to you under the Apache License,
-- Version 2.0 (the "LICENSE"); you may not use this file except
-- in compliance with the LICENSE.

--
-- 000007_seed_advanced_modules.down
--
-- Removes the five advanced module schemas added by 000007_seed_advanced_modules.up.
-- Only deletes rows with version = 1; any subsequently-added versions are left
-- intact (a full rollback would require deleting those too, but this keeps the
-- rollback focused on what this migration specifically inserted).
--

DELETE FROM module_schemas WHERE module = 'queue'          AND version = 1;
DELETE FROM module_schemas WHERE module = 'notifications'  AND version = 1;
DELETE FROM module_schemas WHERE module = 'business-hours' AND version = 1;
DELETE FROM module_schemas WHERE module = 'access'         AND version = 1;
DELETE FROM module_schemas WHERE module = 'branding'       AND version = 1;
