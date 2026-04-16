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
-- 000005_create_bookings.up
--
-- The bookings table is the primary data store for the booking-service.
--
-- Column mapping vs. the existing in-memory booking struct:
--   booking.Customer  → customer_ref  (external caller-supplied customer ID)
--   booking.Service   → service_ref   (external caller-supplied service ID)
--   booking.StartAt   → slot_start    (explicit name; slot_end derived from config)
--
-- Lifecycle states:
--   pending    → booking created, waiting for confirmation
--   confirmed  → accepted (auto or manual depending on autoConfirm config)
--   completed  → service was delivered
--   cancelled  → cancelled by either party
--   no_show    → customer did not appear; slot elapsed without cancellation
--
-- Concurrency / double-booking:
--   A partial unique index on (tenant_id, service_ref, slot_start) where
--   status NOT IN ('cancelled', 'no_show') prevents double-booking at the DB
--   layer regardless of application-level race conditions.  This is the safest
--   backstop; the booking-service should also use SELECT FOR UPDATE when
--   checking availability.
--
-- Row-Level Security:
--   Identical to api_keys — scoped to current_setting('app.current_tenant_id').
-- 

CREATE TABLE bookings (
    id              UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID          NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    -- Opaque reference to the customer in the tenant's own system.  We do not
    -- own customer data; this is a correlation handle only.
    customer_ref    VARCHAR(255)  NOT NULL,
    -- Opaque reference to the service/resource being booked (e.g. "doctor-123",
    -- "room-A", "stylist-7").
    service_ref     VARCHAR(255)  NOT NULL,
    -- Slot boundaries.  slot_end is stored explicitly so bookings can have
    -- variable durations without reading the module config on every query.
    slot_start      TIMESTAMPTZ   NOT NULL,
    slot_end        TIMESTAMPTZ   NOT NULL,
    -- Booking lifecycle state.
    status          VARCHAR(20)   NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending', 'confirmed', 'completed', 'cancelled', 'no_show')),
    -- Free-form JSONB for tenant-defined extra fields (e.g. notes, location,
    -- custom attributes).  Not validated by the platform; opaque pass-through.
    metadata        JSONB         NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    -- Set when the booking transitions to 'cancelled'.
    cancelled_at    TIMESTAMPTZ,
    -- Enforce slot validity at the DB layer.
    CONSTRAINT chk_bookings_slot_order CHECK (slot_end > slot_start)
);

--  Constraints ─
-- Prevent double-booking: only one active (non-cancelled, non-no_show) booking
-- is allowed per (tenant, service, start slot).
CREATE UNIQUE INDEX uq_bookings_active_slot
    ON bookings (tenant_id, service_ref, slot_start)
    WHERE status NOT IN ('cancelled', 'no_show');

--  Indices 
-- List all bookings for a tenant, newest first (management UI).
CREATE INDEX idx_bookings_tenant_created
    ON bookings (tenant_id, created_at DESC);

-- Availability check: "is service X booked between time A and time B?"
-- Covers the slot range overlap query pattern used in availability checks.
CREATE INDEX idx_bookings_tenant_service_slot
    ON bookings (tenant_id, service_ref, slot_start, slot_end);

-- Customer view: "show all bookings for customer Y".
CREATE INDEX idx_bookings_tenant_customer
    ON bookings (tenant_id, customer_ref);

-- Status filter: "show all pending bookings for tenant X".
CREATE INDEX idx_bookings_tenant_status
    ON bookings (tenant_id, status);

--  Trigger 
CREATE TRIGGER trg_bookings_updated_at
    BEFORE UPDATE ON bookings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

--  Row-Level Security ─
ALTER TABLE bookings ENABLE ROW LEVEL SECURITY;

CREATE POLICY bookings_tenant_isolation ON bookings
    USING      (tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID);
