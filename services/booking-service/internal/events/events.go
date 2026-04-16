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

// Package events defines the domain events published by the booking-service
// and the Publisher interface used to deliver them.
package events

import (
	"context"
	"time"
)

// Topic names for booking events.
const (
	TopicBookingCreated   = "booking.created"
	TopicBookingCancelled = "booking.cancelled"
	TopicBookingUpdated   = "booking.updated"
)

// BookingCreatedEvent is published when a new booking is successfully persisted.
type BookingCreatedEvent struct {
	EventID     string    `json:"eventId"`
	EventType   string    `json:"eventType"`
	OccurredAt  time.Time `json:"occurredAt"`
	TenantID    string    `json:"tenantId"`
	BookingID   string    `json:"bookingId"`
	CustomerRef string    `json:"customerRef"`
	ServiceRef  string    `json:"serviceRef"`
	SlotStart   time.Time `json:"slotStart"`
	SlotEnd     time.Time `json:"slotEnd"`
}

// BookingCancelledEvent is published when a booking is cancelled.
type BookingCancelledEvent struct {
	EventID    string    `json:"eventId"`
	EventType  string    `json:"eventType"`
	OccurredAt time.Time `json:"occurredAt"`
	TenantID   string    `json:"tenantId"`
	BookingID  string    `json:"bookingId"`
}

// BookingUpdatedEvent is published when a booking's status changes.
type BookingUpdatedEvent struct {
	EventID    string    `json:"eventId"`
	EventType  string    `json:"eventType"`
	OccurredAt time.Time `json:"occurredAt"`
	TenantID   string    `json:"tenantId"`
	BookingID  string    `json:"bookingId"`
	NewStatus  string    `json:"newStatus"`
}

// Publisher delivers domain events to the message broker.
type Publisher interface {
	// Publish sends msg to the broker.  The key is used for partition
	// assignment (tenant-scoped routing).
	Publish(ctx context.Context, topic string, key []byte, value []byte) error
	// Close flushes pending messages and releases resources.
	Close() error
}
