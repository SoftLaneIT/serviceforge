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

// Package registry provides a static service registry loaded from environment
// variables.  In a later phase this would be replaced by a service-discovery
// mechanism (Consul, Kubernetes EndpointSlices, etc.).
package registry

import (
	"fmt"
	"net/url"

	"github.com/SoftLaneIT/serviceforge/packages/go-common/config"
)

// Service names used as keys in the registry.
const (
	Auth    = "auth-service"
	Tenant  = "tenant-service"
	Booking = "booking-service"
	Config  = "config-service"
)

// Registry maps service names to their base URLs.
type Registry struct {
	urls map[string]*url.URL
}

// Load reads service URLs from environment variables and returns a Registry.
//
// Environment variables (with defaults suitable for docker-compose):
//
//	AUTH_SERVICE_URL     (default: http://auth-service:8082)
//	TENANT_SERVICE_URL   (default: http://tenant-service:8083)
//	BOOKING_SERVICE_URL  (default: http://booking-service:8084)
//	CONFIG_SERVICE_URL   (default: http://config-service:8085)
func Load() (*Registry, error) {
	raw := map[string]string{
		Auth:    config.GetEnv("AUTH_SERVICE_URL", "http://localhost:8082"),
		Tenant:  config.GetEnv("TENANT_SERVICE_URL", "http://localhost:8083"),
		Booking: config.GetEnv("BOOKING_SERVICE_URL", "http://localhost:8084"),
		Config:  config.GetEnv("CONFIG_SERVICE_URL", "http://localhost:8085"),
	}

	r := &Registry{urls: make(map[string]*url.URL, len(raw))}
	for name, rawURL := range raw {
		u, err := url.Parse(rawURL)
		if err != nil {
			return nil, fmt.Errorf("registry: invalid URL for %s (%q): %w", name, rawURL, err)
		}
		r.urls[name] = u
	}
	return r, nil
}

// URL returns the base URL for the named service, or nil if unknown.
func (r *Registry) URL(name string) *url.URL {
	return r.urls[name]
}
