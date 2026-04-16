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

// Package keygen handles the cryptographic generation and hashing of API keys.
//
// Key format:
//
//	sf_live_<64 hex chars>   (production)
//	sf_test_<64 hex chars>   (sandbox)
//
// The 64 hex chars are the hex-encoding of 32 cryptographically random bytes.
// The SHA-256 hash of the full key string is stored in the database; the raw
// key is returned to the caller exactly once and is never persisted.
package keygen

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/SoftLaneIT/serviceforge/services/auth-service/internal/domain"
)

// Result holds the three values derived from a single key-generation call.
type Result struct {
	// RawKey is the full opaque secret, e.g. "sf_live_abc123…".  Return it to
	// the caller once and discard; it is never stored.
	RawKey string
	// Hash is the hex-encoded SHA-256 digest of RawKey.  This is what gets
	// stored in api_keys.key_hash.
	Hash string
	// Prefix is the first 12 characters of RawKey, used for display in UIs.
	// Stored in api_keys.key_prefix.
	Prefix string
}

// Generate creates a new API key for the given environment.
// It reads 32 bytes from crypto/rand and panics if the OS CSPRNG is broken.
func Generate(env domain.Environment) (Result, error) {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return Result{}, fmt.Errorf("keygen: read random bytes: %w", err)
	}

	entropy := hex.EncodeToString(b) // 64 lowercase hex chars

	var prefix string
	if env == domain.EnvProduction {
		prefix = "sf_live_"
	} else {
		prefix = "sf_test_"
	}

	rawKey := prefix + entropy

	sum := sha256.Sum256([]byte(rawKey))
	hash := hex.EncodeToString(sum[:])

	return Result{
		RawKey: rawKey,
		Hash:   hash,
		Prefix: rawKey[:12], // e.g. "sf_live_ab12"
	}, nil
}

// HashRaw returns the hex-encoded SHA-256 digest of rawKey.  Used on the
// validation hot-path to hash the caller-supplied key before the cache/DB
// lookup — avoids storing the raw key anywhere in memory longer than needed.
func HashRaw(rawKey string) string {
	sum := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(sum[:])
}
