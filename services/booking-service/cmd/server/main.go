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

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/SoftLaneIT/serviceforge/packages/go-common/config"
	"github.com/SoftLaneIT/serviceforge/packages/go-common/tenant"
)

type booking struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenantId"`
	Customer  string    `json:"customer"`
	Service   string    `json:"service"`
	StartAt   time.Time `json:"startAt"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type createBookingRequest struct {
	Customer string `json:"customer"`
	Service  string `json:"service"`
	StartAt  string `json:"startAt"`
}

var store = struct {
	sync.RWMutex
	bookings map[string]booking
}{bookings: map[string]booking{}}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/v1/bookings", bookingsHandler)
	mux.HandleFunc("/v1/bookings/", bookingByIDHandler)

	port := config.GetEnv("PORT", "8085")
	log.Printf("booking-service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, tenant.Middleware(mux)); err != nil {
		log.Fatal(err)
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]any{"service": "booking-service", "status": "ok"})
}

func bookingsHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := tenant.FromContext(r.Context())
	switch r.Method {
	case http.MethodGet:
		store.RLock()
		defer store.RUnlock()
		results := make([]booking, 0)
		for _, item := range store.bookings {
			if item.TenantID == tenantID {
				results = append(results, item)
			}
		}
		respondJSON(w, http.StatusOK, results)
	case http.MethodPost:
		var req createBookingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		startAt, err := time.Parse(time.RFC3339, req.StartAt)
		if err != nil {
			http.Error(w, "startAt must be RFC3339", http.StatusBadRequest)
			return
		}
		id := "bk_" + time.Now().UTC().Format("20060102150405")
		record := booking{
			ID:        id,
			TenantID:  tenantID,
			Customer:  req.Customer,
			Service:   req.Service,
			StartAt:   startAt,
			Status:    "created",
			CreatedAt: time.Now().UTC(),
		}
		store.Lock()
		store.bookings[id] = record
		store.Unlock()
		respondJSON(w, http.StatusCreated, record)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func bookingByIDHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := tenant.FromContext(r.Context())
	id := strings.TrimPrefix(r.URL.Path, "/v1/bookings/")
	if id == "" {
		http.Error(w, "missing booking id", http.StatusBadRequest)
		return
	}

	store.Lock()
	record, ok := store.bookings[id]
	if !ok || record.TenantID != tenantID {
		store.Unlock()
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		store.Unlock()
		respondJSON(w, http.StatusOK, record)
	case http.MethodPut:
		record.Status = "updated"
		store.bookings[id] = record
		store.Unlock()
		respondJSON(w, http.StatusOK, record)
	case http.MethodDelete:
		delete(store.bookings, id)
		store.Unlock()
		respondJSON(w, http.StatusOK, map[string]any{"id": id, "deleted": true})
	default:
		store.Unlock()
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
