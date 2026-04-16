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

package keygen_test

import (
	"strings"
	"testing"

	"github.com/SoftLaneIT/serviceforge/services/auth-service/internal/domain"
	"github.com/SoftLaneIT/serviceforge/services/auth-service/internal/keygen"
)

func TestGenerate_Production(t *testing.T) {
	r, err := keygen.Generate(domain.EnvProduction)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if !strings.HasPrefix(r.RawKey, "sf_live_") {
		t.Errorf("production key should start with sf_live_, got %q", r.RawKey[:8])
	}
	// sf_live_ (8) + 64 hex = 72 chars total
	if len(r.RawKey) != 72 {
		t.Errorf("raw key length = %d, want 72", len(r.RawKey))
	}
}

func TestGenerate_Sandbox(t *testing.T) {
	r, err := keygen.Generate(domain.EnvSandbox)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if !strings.HasPrefix(r.RawKey, "sf_test_") {
		t.Errorf("sandbox key should start with sf_test_, got %q", r.RawKey[:8])
	}
	if len(r.RawKey) != 72 {
		t.Errorf("raw key length = %d, want 72", len(r.RawKey))
	}
}

func TestGenerate_HashLength(t *testing.T) {
	r, err := keygen.Generate(domain.EnvSandbox)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	// SHA-256 → 32 bytes → 64 hex chars
	if len(r.Hash) != 64 {
		t.Errorf("hash length = %d, want 64", len(r.Hash))
	}
}

func TestGenerate_PrefixIs12Chars(t *testing.T) {
	r, err := keygen.Generate(domain.EnvProduction)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(r.Prefix) != 12 {
		t.Errorf("prefix length = %d, want 12", len(r.Prefix))
	}
	if r.Prefix != r.RawKey[:12] {
		t.Errorf("prefix %q is not first 12 chars of raw key", r.Prefix)
	}
}

func TestGenerate_Uniqueness(t *testing.T) {
	a, _ := keygen.Generate(domain.EnvSandbox)
	b, _ := keygen.Generate(domain.EnvSandbox)
	if a.RawKey == b.RawKey {
		t.Error("two Generate calls returned the same raw key")
	}
	if a.Hash == b.Hash {
		t.Error("two Generate calls returned the same hash")
	}
}

func TestHashRaw_Deterministic(t *testing.T) {
	r, err := keygen.Generate(domain.EnvSandbox)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	// HashRaw of the same key must equal the hash embedded in the result.
	if got := keygen.HashRaw(r.RawKey); got != r.Hash {
		t.Errorf("HashRaw(%q) = %q, want %q", r.RawKey, got, r.Hash)
	}
}

func TestHashRaw_Length(t *testing.T) {
	h := keygen.HashRaw("sf_live_somekey")
	if len(h) != 64 {
		t.Errorf("HashRaw length = %d, want 64", len(h))
	}
}
