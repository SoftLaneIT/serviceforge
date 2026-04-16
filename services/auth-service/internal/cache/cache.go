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

// Package cache provides the caching layer for API key validation.
//
// On the validate hot-path the service checks Redis first; only on a cache
// miss does it fall back to Postgres.  Cached entries use a 5-minute TTL so
// that a revoked key stops working within that window even without an explicit
// cache invalidation.  Revoke calls actively delete the cache entry so the
// window is usually much shorter in practice.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/SoftLaneIT/serviceforge/services/auth-service/internal/domain"
)

const (
	// DefaultTTL is the cache entry lifetime for validated API keys.
	DefaultTTL = 5 * time.Minute
	// keyPrefix is prepended to every Redis key.
	keyPrefix = "apikey:"
)

// Cache defines the contract for the API-key validation cache layer.
type Cache interface {
	// Get returns the cached APIKey for the given SHA-256 hash.
	// found=false (and err=nil) when the entry is absent or expired.
	Get(ctx context.Context, hash string) (key *domain.APIKey, found bool, err error)

	// Set stores key in the cache under hash with the given TTL.
	Set(ctx context.Context, hash string, key *domain.APIKey, ttl time.Duration) error

	// Delete removes the cache entry for hash.  It is safe to call when the
	// entry does not exist.
	Delete(ctx context.Context, hash string) error

	// Ping verifies the cache connection is alive.
	Ping(ctx context.Context) error
}

// RedisCache is the Redis-backed implementation of Cache.
type RedisCache struct {
	client *redis.Client
}

// NewRedis creates a RedisCache connected to addr (e.g. "localhost:6379").
func NewRedis(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

func cacheKey(hash string) string { return keyPrefix + hash }

// Get retrieves and deserialises an APIKey from Redis.
func (c *RedisCache) Get(ctx context.Context, hash string) (*domain.APIKey, bool, error) {
	b, err := c.client.Get(ctx, cacheKey(hash)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("cache get: %w", err)
	}

	var k domain.APIKey
	if err := json.Unmarshal(b, &k); err != nil {
		// Treat corrupt entries as a cache miss — Postgres will repopulate.
		return nil, false, nil
	}
	return &k, true, nil
}

// Set serialises key to JSON and stores it in Redis with ttl.
func (c *RedisCache) Set(ctx context.Context, hash string, key *domain.APIKey, ttl time.Duration) error {
	b, err := json.Marshal(key)
	if err != nil {
		return fmt.Errorf("cache marshal: %w", err)
	}
	if err := c.client.Set(ctx, cacheKey(hash), b, ttl).Err(); err != nil {
		return fmt.Errorf("cache set: %w", err)
	}
	return nil
}

// Delete removes the entry for hash from Redis.
func (c *RedisCache) Delete(ctx context.Context, hash string) error {
	if err := c.client.Del(ctx, cacheKey(hash)).Err(); err != nil {
		return fmt.Errorf("cache delete: %w", err)
	}
	return nil
}

// Ping checks the Redis connection.
func (c *RedisCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// NoopCache is a Cache implementation that always reports a miss.
// Used in tests and when Redis is intentionally disabled.
type NoopCache struct{}

func (NoopCache) Get(_ context.Context, _ string) (*domain.APIKey, bool, error) {
	return nil, false, nil
}
func (NoopCache) Set(_ context.Context, _ string, _ *domain.APIKey, _ time.Duration) error {
	return nil
}
func (NoopCache) Delete(_ context.Context, _ string) error { return nil }
func (NoopCache) Ping(_ context.Context) error             { return nil }
