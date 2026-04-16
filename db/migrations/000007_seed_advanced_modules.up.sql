-- Copyright (c) 2026, SoftlaneIT (https://softlaneit.com/) All Rights Reserved.
--
-- SoftlaneIT licenses this file to you under the Apache License,
-- Version 2.0 (the "LICENSE"); you may not use this file except
-- in compliance with the LICENSE.

--
-- 000007_seed_advanced_modules.up
--
-- Adds five advanced configuration modules to the module_schemas registry:
--
--   queue          — Queue management: locking, capacity, overflow, wait-time
--   notifications  — Notification channels: webhooks, email, SMS, retry policy
--   business-hours — Operating schedule: timezone, per-day hours, holidays
--   access         — Access control: session policy, IP allowlist, rate limits
--   branding       — White-label settings: colors, logo, domain, company info
--
-- Each schema uses JSON Schema draft-07 with x-ui hints so the management-UI
-- SchemaForm renders the right control automatically (toggle, slider, number
-- input, nested section, select, etc.).
--

-- ──
-- 1. queue — booking-queue and locking behaviour
-- ──
INSERT INTO module_schemas (module, version, description, schema, defaults)
VALUES (
    'queue',
    1,
    'Controls queue management: concurrent locking timeouts, seat capacity, overflow behaviour, and maximum wait times.',
    '{
        "$schema": "http://json-schema.org/draft-07/schema#",
        "title": "Queue Module Configuration",
        "type": "object",
        "additionalProperties": false,
        "properties": {
            "lockingTimeoutSeconds": {
                "type": "integer",
                "minimum": 10,
                "maximum": 600,
                "description": "How long (seconds) a seat lock is held before it expires and the seat is released back to the queue.",
                "x-ui": {"component": "slider", "step": 10}
            },
            "lockingTimePeriodMinutes": {
                "type": "integer",
                "minimum": 1,
                "maximum": 1440,
                "description": "Sliding window (minutes) within which the locking timeout is enforced.",
                "x-ui": {"component": "number_input"}
            },
            "maxSeats": {
                "type": "integer",
                "minimum": 1,
                "maximum": 10000,
                "description": "Total seats available in the queue at any one time.",
                "x-ui": {"component": "number_input"}
            },
            "maxConcurrentUsers": {
                "type": "integer",
                "minimum": 1,
                "maximum": 5000,
                "description": "Maximum number of users allowed inside the queue simultaneously.",
                "x-ui": {"component": "number_input"}
            },
            "queueType": {
                "type": "string",
                "enum": ["fifo", "priority", "fair-share"],
                "description": "Dispatching algorithm: fifo (first-in-first-out), priority (weighted), or fair-share (round-robin by tenant group)."
            },
            "overflowBehaviour": {
                "type": "string",
                "enum": ["reject", "wait", "redirect"],
                "description": "What to do when the queue is full: reject the request, place it in a waiting list, or redirect to a fallback URL."
            },
            "maxWaitTimeSeconds": {
                "type": "integer",
                "minimum": 0,
                "maximum": 3600,
                "description": "Maximum time (seconds) a user waits before being automatically ejected. 0 = unlimited.",
                "x-ui": {"component": "slider", "step": 30}
            },
            "enablePositionNotifications": {
                "type": "boolean",
                "description": "Send real-time queue-position updates to waiting users.",
                "x-ui": {"component": "toggle"}
            },
            "positionUpdateIntervalSeconds": {
                "type": "integer",
                "minimum": 5,
                "maximum": 120,
                "description": "How often (seconds) queue-position notifications are pushed to waiting users.",
                "x-ui": {"component": "slider", "step": 5}
            },
            "fairnessPolicy": {
                "type": "object",
                "additionalProperties": false,
                "description": "Controls how the fair-share algorithm distributes seats across tenant groups.",
                "properties": {
                    "maxConsecutiveGrantsPerGroup": {
                        "type": "integer",
                        "minimum": 1,
                        "maximum": 100,
                        "description": "Maximum consecutive seat grants before the scheduler moves to the next group."
                    },
                    "priorityBoostAfterWaitMinutes": {
                        "type": "integer",
                        "minimum": 0,
                        "maximum": 60,
                        "description": "Minutes a low-priority user must wait before receiving a temporary priority boost."
                    }
                },
                "required": ["maxConsecutiveGrantsPerGroup", "priorityBoostAfterWaitMinutes"],
                "x-ui": {"component": "nested_form"}
            }
        },
        "required": ["lockingTimeoutSeconds", "maxSeats", "queueType", "overflowBehaviour"]
    }'::jsonb,
    '{
        "lockingTimeoutSeconds": 60,
        "lockingTimePeriodMinutes": 30,
        "maxSeats": 100,
        "maxConcurrentUsers": 200,
        "queueType": "fifo",
        "overflowBehaviour": "wait",
        "maxWaitTimeSeconds": 900,
        "enablePositionNotifications": true,
        "positionUpdateIntervalSeconds": 30,
        "fairnessPolicy": {
            "maxConsecutiveGrantsPerGroup": 5,
            "priorityBoostAfterWaitMinutes": 10
        }
    }'::jsonb
);

-- ──
-- 2. notifications — outbound notification channels and retry policy
-- ──
INSERT INTO module_schemas (module, version, description, schema, defaults)
VALUES (
    'notifications',
    1,
    'Configures outbound notification channels (webhook, email, SMS) and the retry/backoff policy for failed deliveries.',
    '{
        "$schema": "http://json-schema.org/draft-07/schema#",
        "title": "Notifications Module Configuration",
        "type": "object",
        "additionalProperties": false,
        "properties": {
            "webhookEnabled": {
                "type": "boolean",
                "description": "Enable outbound webhook delivery for booking and queue events.",
                "x-ui": {"component": "toggle"}
            },
            "webhookUrl": {
                "type": "string",
                "description": "HTTPS endpoint that receives event payloads. Must be publicly reachable."
            },
            "webhookSecret": {
                "type": "string",
                "description": "HMAC-SHA256 signing secret. Included as X-Signature-256 header on every delivery."
            },
            "webhookEvents": {
                "type": "string",
                "enum": ["all", "booking_only", "queue_only", "custom"],
                "description": "Which event categories are forwarded to the webhook URL."
            },
            "emailEnabled": {
                "type": "boolean",
                "description": "Enable transactional email notifications (confirmation, reminder, cancellation).",
                "x-ui": {"component": "toggle"}
            },
            "emailFromAddress": {
                "type": "string",
                "description": "Sender address used for outbound emails (e.g. noreply@yourcompany.com)."
            },
            "emailTriggers": {
                "type": "object",
                "additionalProperties": false,
                "description": "Fine-grained control over which email notifications are sent.",
                "properties": {
                    "onBookingConfirmed": {
                        "type": "boolean",
                        "description": "Send email when a booking is confirmed."
                    },
                    "onBookingCancelled": {
                        "type": "boolean",
                        "description": "Send email when a booking is cancelled."
                    },
                    "onBookingReminder": {
                        "type": "boolean",
                        "description": "Send reminder email before the slot start time."
                    },
                    "reminderHoursBefore": {
                        "type": "integer",
                        "minimum": 1,
                        "maximum": 72,
                        "description": "How many hours before the slot to send the reminder."
                    }
                },
                "required": ["onBookingConfirmed", "onBookingCancelled", "onBookingReminder", "reminderHoursBefore"],
                "x-ui": {"component": "nested_form"}
            },
            "smsEnabled": {
                "type": "boolean",
                "description": "Enable SMS notifications via the configured SMS gateway.",
                "x-ui": {"component": "toggle"}
            },
            "smsProviderApiKey": {
                "type": "string",
                "description": "API key for your SMS gateway provider (Twilio, Vonage, etc.)."
            },
            "smsTriggers": {
                "type": "object",
                "additionalProperties": false,
                "description": "Controls which events trigger an SMS.",
                "properties": {
                    "onBookingConfirmed": {
                        "type": "boolean",
                        "description": "Send SMS when a booking is confirmed."
                    },
                    "onBookingReminder": {
                        "type": "boolean",
                        "description": "Send SMS reminder before the slot."
                    },
                    "reminderHoursBefore": {
                        "type": "integer",
                        "minimum": 1,
                        "maximum": 48,
                        "description": "Hours before the slot to send the SMS reminder."
                    }
                },
                "required": ["onBookingConfirmed", "onBookingReminder", "reminderHoursBefore"],
                "x-ui": {"component": "nested_form"}
            },
            "retryPolicy": {
                "type": "object",
                "additionalProperties": false,
                "description": "Governs how failed webhook/email/SMS deliveries are retried.",
                "properties": {
                    "maxAttempts": {
                        "type": "integer",
                        "minimum": 1,
                        "maximum": 10,
                        "description": "Maximum delivery attempts before the event is marked as permanently failed."
                    },
                    "initialBackoffSeconds": {
                        "type": "integer",
                        "minimum": 1,
                        "maximum": 300,
                        "description": "Delay before the first retry, in seconds."
                    },
                    "backoffMultiplier": {
                        "type": "integer",
                        "minimum": 1,
                        "maximum": 10,
                        "description": "Exponential multiplier applied to the delay between successive retries."
                    },
                    "alertOnPermanentFailure": {
                        "type": "boolean",
                        "description": "Emit an internal alert when an event reaches the maximum retry limit."
                    }
                },
                "required": ["maxAttempts", "initialBackoffSeconds", "backoffMultiplier"],
                "x-ui": {"component": "nested_form"}
            }
        },
        "required": ["webhookEnabled", "emailEnabled", "smsEnabled"]
    }'::jsonb,
    '{
        "webhookEnabled": false,
        "webhookUrl": "",
        "webhookSecret": "",
        "webhookEvents": "all",
        "emailEnabled": true,
        "emailFromAddress": "noreply@example.com",
        "emailTriggers": {
            "onBookingConfirmed": true,
            "onBookingCancelled": true,
            "onBookingReminder": true,
            "reminderHoursBefore": 24
        },
        "smsEnabled": false,
        "smsProviderApiKey": "",
        "smsTriggers": {
            "onBookingConfirmed": false,
            "onBookingReminder": false,
            "reminderHoursBefore": 2
        },
        "retryPolicy": {
            "maxAttempts": 3,
            "initialBackoffSeconds": 30,
            "backoffMultiplier": 2,
            "alertOnPermanentFailure": true
        }
    }'::jsonb
);

-- ──
-- 3. business-hours — operating schedule and holiday calendar
-- ──
INSERT INTO module_schemas (module, version, description, schema, defaults)
VALUES (
    'business-hours',
    1,
    'Defines the tenant''s operating schedule (timezone, per-day open/close times) and how out-of-hours bookings are handled.',
    '{
        "$schema": "http://json-schema.org/draft-07/schema#",
        "title": "Business Hours Module Configuration",
        "type": "object",
        "additionalProperties": false,
        "properties": {
            "timezone": {
                "type": "string",
                "enum": [
                    "UTC",
                    "America/New_York",
                    "America/Chicago",
                    "America/Denver",
                    "America/Los_Angeles",
                    "Europe/London",
                    "Europe/Paris",
                    "Europe/Berlin",
                    "Asia/Dubai",
                    "Asia/Kolkata",
                    "Asia/Singapore",
                    "Asia/Tokyo",
                    "Australia/Sydney",
                    "Pacific/Auckland"
                ],
                "description": "IANA timezone identifier used to interpret all opening/closing times."
            },
            "allowBookingsOutsideHours": {
                "type": "boolean",
                "description": "If false, the booking service rejects new bookings for slots that fall outside the defined business hours.",
                "x-ui": {"component": "toggle"}
            },
            "monday": {
                "type": "object",
                "additionalProperties": false,
                "description": "Monday operating hours.",
                "properties": {
                    "open": {"type": "boolean", "description": "Whether the business is open on this day."},
                    "openTime": {"type": "string", "description": "Opening time (HH:MM, 24-hour)."},
                    "closeTime": {"type": "string", "description": "Closing time (HH:MM, 24-hour)."}
                },
                "required": ["open", "openTime", "closeTime"],
                "x-ui": {"component": "nested_form"}
            },
            "tuesday": {
                "type": "object",
                "additionalProperties": false,
                "description": "Tuesday operating hours.",
                "properties": {
                    "open": {"type": "boolean", "description": "Whether the business is open on this day."},
                    "openTime": {"type": "string", "description": "Opening time (HH:MM, 24-hour)."},
                    "closeTime": {"type": "string", "description": "Closing time (HH:MM, 24-hour)."}
                },
                "required": ["open", "openTime", "closeTime"],
                "x-ui": {"component": "nested_form"}
            },
            "wednesday": {
                "type": "object",
                "additionalProperties": false,
                "description": "Wednesday operating hours.",
                "properties": {
                    "open": {"type": "boolean", "description": "Whether the business is open on this day."},
                    "openTime": {"type": "string", "description": "Opening time (HH:MM, 24-hour)."},
                    "closeTime": {"type": "string", "description": "Closing time (HH:MM, 24-hour)."}
                },
                "required": ["open", "openTime", "closeTime"],
                "x-ui": {"component": "nested_form"}
            },
            "thursday": {
                "type": "object",
                "additionalProperties": false,
                "description": "Thursday operating hours.",
                "properties": {
                    "open": {"type": "boolean", "description": "Whether the business is open on this day."},
                    "openTime": {"type": "string", "description": "Opening time (HH:MM, 24-hour)."},
                    "closeTime": {"type": "string", "description": "Closing time (HH:MM, 24-hour)."}
                },
                "required": ["open", "openTime", "closeTime"],
                "x-ui": {"component": "nested_form"}
            },
            "friday": {
                "type": "object",
                "additionalProperties": false,
                "description": "Friday operating hours.",
                "properties": {
                    "open": {"type": "boolean", "description": "Whether the business is open on this day."},
                    "openTime": {"type": "string", "description": "Opening time (HH:MM, 24-hour)."},
                    "closeTime": {"type": "string", "description": "Closing time (HH:MM, 24-hour)."}
                },
                "required": ["open", "openTime", "closeTime"],
                "x-ui": {"component": "nested_form"}
            },
            "saturday": {
                "type": "object",
                "additionalProperties": false,
                "description": "Saturday operating hours.",
                "properties": {
                    "open": {"type": "boolean", "description": "Whether the business is open on this day."},
                    "openTime": {"type": "string", "description": "Opening time (HH:MM, 24-hour)."},
                    "closeTime": {"type": "string", "description": "Closing time (HH:MM, 24-hour)."}
                },
                "required": ["open", "openTime", "closeTime"],
                "x-ui": {"component": "nested_form"}
            },
            "sunday": {
                "type": "object",
                "additionalProperties": false,
                "description": "Sunday operating hours.",
                "properties": {
                    "open": {"type": "boolean", "description": "Whether the business is open on this day."},
                    "openTime": {"type": "string", "description": "Opening time (HH:MM, 24-hour)."},
                    "closeTime": {"type": "string", "description": "Closing time (HH:MM, 24-hour)."}
                },
                "required": ["open", "openTime", "closeTime"],
                "x-ui": {"component": "nested_form"}
            },
            "breakDurationMinutes": {
                "type": "integer",
                "minimum": 0,
                "maximum": 240,
                "description": "Length of the mid-day break (minutes). 0 disables the break.",
                "x-ui": {"component": "slider", "step": 15}
            },
            "breakStartTime": {
                "type": "string",
                "description": "Time when the daily break begins (HH:MM, 24-hour). Ignored if breakDurationMinutes is 0."
            }
        },
        "required": ["timezone", "allowBookingsOutsideHours"]
    }'::jsonb,
    '{
        "timezone": "UTC",
        "allowBookingsOutsideHours": false,
        "monday":    {"open": true,  "openTime": "09:00", "closeTime": "17:00"},
        "tuesday":   {"open": true,  "openTime": "09:00", "closeTime": "17:00"},
        "wednesday": {"open": true,  "openTime": "09:00", "closeTime": "17:00"},
        "thursday":  {"open": true,  "openTime": "09:00", "closeTime": "17:00"},
        "friday":    {"open": true,  "openTime": "09:00", "closeTime": "17:00"},
        "saturday":  {"open": false, "openTime": "10:00", "closeTime": "14:00"},
        "sunday":    {"open": false, "openTime": "10:00", "closeTime": "14:00"},
        "breakDurationMinutes": 60,
        "breakStartTime": "12:00"
    }'::jsonb
);

-- ──
-- 4. access — session management, IP controls, and rate limiting
-- ──
INSERT INTO module_schemas (module, version, description, schema, defaults)
VALUES (
    'access',
    1,
    'Controls session lifecycle, concurrent-session limits, IP allowlisting, and API rate-limiting for the tenant.',
    '{
        "$schema": "http://json-schema.org/draft-07/schema#",
        "title": "Access Control Module Configuration",
        "type": "object",
        "additionalProperties": false,
        "properties": {
            "sessionTimeoutMinutes": {
                "type": "integer",
                "minimum": 5,
                "maximum": 1440,
                "description": "Inactive session TTL in minutes. After this period without activity, the session is invalidated.",
                "x-ui": {"component": "slider", "step": 5}
            },
            "absoluteSessionTimeoutHours": {
                "type": "integer",
                "minimum": 1,
                "maximum": 720,
                "description": "Hard maximum for any session regardless of activity. Forces re-authentication after this many hours.",
                "x-ui": {"component": "number_input"}
            },
            "maxConcurrentSessions": {
                "type": "integer",
                "minimum": 1,
                "maximum": 50,
                "description": "How many simultaneous active sessions one user is allowed. Oldest session is invalidated when the limit is reached.",
                "x-ui": {"component": "number_input"}
            },
            "enforceIpAllowlist": {
                "type": "boolean",
                "description": "When enabled, only requests from IPs in the allowlist are accepted. All others receive 403.",
                "x-ui": {"component": "toggle"}
            },
            "ipAllowlist": {
                "type": "string",
                "description": "Comma-separated list of allowed CIDRs or individual IPs (e.g. 192.168.1.0/24, 10.0.0.1). Ignored when enforceIpAllowlist is false."
            },
            "requireMfa": {
                "type": "boolean",
                "description": "Mandate multi-factor authentication for all users in this tenant.",
                "x-ui": {"component": "toggle"}
            },
            "mfaGracePeriodHours": {
                "type": "integer",
                "minimum": 0,
                "maximum": 168,
                "description": "Hours a new user has to enrol in MFA before access is blocked. 0 = enforce immediately.",
                "x-ui": {"component": "number_input"}
            },
            "rateLimiting": {
                "type": "object",
                "additionalProperties": false,
                "description": "Token-bucket rate limits applied per authenticated user.",
                "properties": {
                    "requestsPerMinute": {
                        "type": "integer",
                        "minimum": 1,
                        "maximum": 10000,
                        "description": "Maximum API requests per user per minute."
                    },
                    "requestsPerHour": {
                        "type": "integer",
                        "minimum": 1,
                        "maximum": 100000,
                        "description": "Maximum API requests per user per hour."
                    },
                    "burstMultiplier": {
                        "type": "integer",
                        "minimum": 1,
                        "maximum": 10,
                        "description": "Short-burst allowance as a multiple of the per-minute limit."
                    },
                    "throttleOnExceed": {
                        "type": "boolean",
                        "description": "If true, slow requests instead of rejecting them when the limit is exceeded."
                    }
                },
                "required": ["requestsPerMinute", "requestsPerHour", "burstMultiplier"],
                "x-ui": {"component": "nested_form"}
            },
            "auditLogRetentionDays": {
                "type": "integer",
                "minimum": 7,
                "maximum": 3650,
                "description": "How many days access-audit log entries are retained before automatic purging.",
                "x-ui": {"component": "number_input"}
            },
            "loginFailurePolicy": {
                "type": "object",
                "additionalProperties": false,
                "description": "Governs lockout behaviour after consecutive failed login attempts.",
                "properties": {
                    "maxFailedAttempts": {
                        "type": "integer",
                        "minimum": 1,
                        "maximum": 20,
                        "description": "Consecutive failures before the account is locked."
                    },
                    "lockoutDurationMinutes": {
                        "type": "integer",
                        "minimum": 1,
                        "maximum": 1440,
                        "description": "How long the account stays locked (minutes). Use 0 for indefinite lockout."
                    },
                    "notifyOnLockout": {
                        "type": "boolean",
                        "description": "Send an email to the user when their account is locked."
                    }
                },
                "required": ["maxFailedAttempts", "lockoutDurationMinutes"],
                "x-ui": {"component": "nested_form"}
            }
        },
        "required": ["sessionTimeoutMinutes", "maxConcurrentSessions", "enforceIpAllowlist"]
    }'::jsonb,
    '{
        "sessionTimeoutMinutes": 60,
        "absoluteSessionTimeoutHours": 24,
        "maxConcurrentSessions": 5,
        "enforceIpAllowlist": false,
        "ipAllowlist": "",
        "requireMfa": false,
        "mfaGracePeriodHours": 48,
        "rateLimiting": {
            "requestsPerMinute": 300,
            "requestsPerHour": 10000,
            "burstMultiplier": 3,
            "throttleOnExceed": false
        },
        "auditLogRetentionDays": 90,
        "loginFailurePolicy": {
            "maxFailedAttempts": 5,
            "lockoutDurationMinutes": 30,
            "notifyOnLockout": true
        }
    }'::jsonb
);

-- ──
-- 5. branding — white-label appearance settings
-- ──
INSERT INTO module_schemas (module, version, description, schema, defaults)
VALUES (
    'branding',
    1,
    'White-label customisation: primary colour palette, logos, custom domain, company identity, and UI copy overrides.',
    '{
        "$schema": "http://json-schema.org/draft-07/schema#",
        "title": "Branding Module Configuration",
        "type": "object",
        "additionalProperties": false,
        "properties": {
            "companyName": {
                "type": "string",
                "description": "Displayed company name used in emails, receipts, and the booking portal header."
            },
            "supportEmail": {
                "type": "string",
                "description": "Support contact email shown in customer-facing correspondence."
            },
            "supportPhone": {
                "type": "string",
                "description": "Support phone number shown on the booking portal and confirmation emails."
            },
            "websiteUrl": {
                "type": "string",
                "description": "Company website URL linked from the booking portal footer."
            },
            "customDomain": {
                "type": "string",
                "description": "Custom hostname for the white-labelled booking portal (e.g. book.yourcompany.com). Requires DNS CNAME setup."
            },
            "colors": {
                "type": "object",
                "additionalProperties": false,
                "description": "Brand colour palette applied to the booking portal and email templates.",
                "properties": {
                    "primary": {
                        "type": "string",
                        "description": "Primary brand colour as a hex code (e.g. #4F46E5)."
                    },
                    "primaryForeground": {
                        "type": "string",
                        "description": "Text colour rendered on top of the primary colour (e.g. #FFFFFF)."
                    },
                    "accent": {
                        "type": "string",
                        "description": "Accent / secondary colour used for highlights and CTAs."
                    },
                    "background": {
                        "type": "string",
                        "description": "Page background colour for the booking portal."
                    }
                },
                "required": ["primary", "primaryForeground"],
                "x-ui": {"component": "nested_form"}
            },
            "logoUrl": {
                "type": "string",
                "description": "HTTPS URL of the logo image (PNG/SVG, min 200 × 60 px) shown in the portal header."
            },
            "faviconUrl": {
                "type": "string",
                "description": "HTTPS URL of the 32×32 favicon for the booking portal."
            },
            "portalTitle": {
                "type": "string",
                "description": "Browser tab / <title> text for the booking portal."
            },
            "portalWelcomeMessage": {
                "type": "string",
                "description": "Short welcome text displayed on the booking portal homepage."
            },
            "emailFooterText": {
                "type": "string",
                "description": "Custom footer text appended to all outbound transactional emails."
            },
            "hidePoweredBy": {
                "type": "boolean",
                "description": "Hide the \"Powered by ServiceForge\" attribution in the portal footer. Available on Pro and Enterprise plans only.",
                "x-ui": {"component": "toggle"}
            },
            "locale": {
                "type": "string",
                "enum": ["en-US", "en-GB", "fr-FR", "de-DE", "es-ES", "pt-BR", "ja-JP", "zh-CN"],
                "description": "Default locale for date/time formatting and translated portal copy."
            },
            "dateFormat": {
                "type": "string",
                "enum": ["MM/DD/YYYY", "DD/MM/YYYY", "YYYY-MM-DD"],
                "description": "Date display format used throughout the portal."
            },
            "timeFormat": {
                "type": "string",
                "enum": ["12h", "24h"],
                "description": "Clock format shown to users in the portal."
            }
        },
        "required": ["companyName", "supportEmail", "locale"]
    }'::jsonb,
    '{
        "companyName": "My Company",
        "supportEmail": "support@example.com",
        "supportPhone": "",
        "websiteUrl": "",
        "customDomain": "",
        "colors": {
            "primary": "#4F46E5",
            "primaryForeground": "#FFFFFF",
            "accent": "#7C3AED",
            "background": "#F8FAFC"
        },
        "logoUrl": "",
        "faviconUrl": "",
        "portalTitle": "Book an Appointment",
        "portalWelcomeMessage": "Schedule your appointment quickly and easily.",
        "emailFooterText": "",
        "hidePoweredBy": false,
        "locale": "en-US",
        "dateFormat": "MM/DD/YYYY",
        "timeFormat": "12h"
    }'::jsonb
);
