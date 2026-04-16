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

package domain_test

import (
	"strings"
	"testing"

	"github.com/SoftLaneIT/serviceforge/services/tenant-service/internal/domain"
)

// ptr is a helper to take the address of a string literal in table tests.
func ptr[T any](v T) *T { return &v }

//  CreateParams.Validate ─

func TestCreateParams_Validate(t *testing.T) {
	tests := []struct {
		name    string
		params  domain.CreateParams
		wantErr bool
		errFrag string // substring expected in the error message
	}{
		{
			name:   "valid minimal",
			params: domain.CreateParams{Name: "Acme Corp", Slug: "acme-corp", Plan: domain.PlanStarter},
		},
		{
			name:   "valid pro plan",
			params: domain.CreateParams{Name: "Big Co", Slug: "big-co", Plan: domain.PlanPro},
		},
		{
			name:   "valid enterprise plan",
			params: domain.CreateParams{Name: "Mega", Slug: "mega-org", Plan: domain.PlanEnterprise},
		},
		{
			name:    "missing name",
			params:  domain.CreateParams{Slug: "acme-corp", Plan: domain.PlanStarter},
			wantErr: true,
			errFrag: "name is required",
		},
		{
			name:    "name only whitespace",
			params:  domain.CreateParams{Name: "   ", Slug: "acme-corp", Plan: domain.PlanStarter},
			wantErr: true,
			errFrag: "name is required",
		},
		{
			name:    "name too long",
			params:  domain.CreateParams{Name: strings.Repeat("x", 256), Slug: "acme", Plan: domain.PlanStarter},
			wantErr: true,
			errFrag: "255 characters",
		},
		{
			name:    "missing slug",
			params:  domain.CreateParams{Name: "Acme", Plan: domain.PlanStarter},
			wantErr: true,
			errFrag: "slug is required",
		},
		{
			name:    "slug starts with hyphen",
			params:  domain.CreateParams{Name: "Acme", Slug: "-acme", Plan: domain.PlanStarter},
			wantErr: true,
			errFrag: "slug must be",
		},
		{
			name:    "slug ends with hyphen",
			params:  domain.CreateParams{Name: "Acme", Slug: "acme-", Plan: domain.PlanStarter},
			wantErr: true,
			errFrag: "slug must be",
		},
		{
			name:    "slug has uppercase",
			params:  domain.CreateParams{Name: "Acme", Slug: "ACME", Plan: domain.PlanStarter},
			wantErr: true,
			errFrag: "slug must be",
		},
		{
			name:    "slug too short (2 chars)",
			params:  domain.CreateParams{Name: "Acme", Slug: "ab", Plan: domain.PlanStarter},
			wantErr: true,
			errFrag: "slug must be",
		},
		{
			name:    "slug too long (101 chars)",
			params:  domain.CreateParams{Name: "Acme", Slug: "a" + strings.Repeat("b", 98) + "c1", Plan: domain.PlanStarter},
			wantErr: true,
			errFrag: "slug must be",
		},
		{
			name:    "slug has spaces",
			params:  domain.CreateParams{Name: "Acme", Slug: "acme corp", Plan: domain.PlanStarter},
			wantErr: true,
			errFrag: "slug must be",
		},
		{
			name:    "missing plan",
			params:  domain.CreateParams{Name: "Acme", Slug: "acme-corp"},
			wantErr: true,
			errFrag: "plan is required",
		},
		{
			name:    "invalid plan",
			params:  domain.CreateParams{Name: "Acme", Slug: "acme-corp", Plan: "golden"},
			wantErr: true,
			errFrag: "plan must be one of",
		},
		{
			name:    "multiple errors reported together",
			params:  domain.CreateParams{},
			wantErr: true,
			errFrag: "name is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.params.Validate()
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.errFrag != "" && !strings.Contains(err.Error(), tc.errFrag) {
					t.Fatalf("error %q does not contain %q", err.Error(), tc.errFrag)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

//  UpdateParams.Validate ─

func TestUpdateParams_Validate(t *testing.T) {
	tests := []struct {
		name    string
		params  domain.UpdateParams
		wantErr bool
		errFrag string
	}{
		{
			name:   "name only",
			params: domain.UpdateParams{Name: ptr("New Name")},
		},
		{
			name:   "plan only",
			params: domain.UpdateParams{Plan: ptr(domain.PlanPro)},
		},
		{
			name:   "settings only",
			params: domain.UpdateParams{Settings: map[string]any{"key": "val"}},
		},
		{
			name:   "all fields",
			params: domain.UpdateParams{Name: ptr("X"), Plan: ptr(domain.PlanEnterprise), Settings: map[string]any{}},
		},
		{
			name:    "empty (no fields)",
			params:  domain.UpdateParams{},
			wantErr: true,
			errFrag: "at least one field",
		},
		{
			name:    "name blank",
			params:  domain.UpdateParams{Name: ptr("   ")},
			wantErr: true,
			errFrag: "name must not be empty",
		},
		{
			name:    "name too long",
			params:  domain.UpdateParams{Name: ptr(strings.Repeat("x", 256))},
			wantErr: true,
			errFrag: "255 characters",
		},
		{
			name:    "invalid plan",
			params:  domain.UpdateParams{Plan: ptr(domain.Plan("diamond"))},
			wantErr: true,
			errFrag: "plan must be one of",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.params.Validate()
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.errFrag != "" && !strings.Contains(err.Error(), tc.errFrag) {
					t.Fatalf("error %q does not contain %q", err.Error(), tc.errFrag)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

//  Plan / Status validity

func TestPlan_Valid(t *testing.T) {
	for _, p := range []domain.Plan{domain.PlanStarter, domain.PlanPro, domain.PlanEnterprise} {
		if !p.Valid() {
			t.Errorf("expected %q to be valid", p)
		}
	}
	for _, p := range []domain.Plan{"", "free", "gold", "STARTER"} {
		if p.Valid() {
			t.Errorf("expected %q to be invalid", p)
		}
	}
}

func TestStatus_Valid(t *testing.T) {
	for _, s := range []domain.Status{domain.StatusActive, domain.StatusSuspended, domain.StatusDeleted} {
		if !s.Valid() {
			t.Errorf("expected %q to be valid", s)
		}
	}
	for _, s := range []domain.Status{"", "paused", "ACTIVE"} {
		if s.Valid() {
			t.Errorf("expected %q to be invalid", s)
		}
	}
}
