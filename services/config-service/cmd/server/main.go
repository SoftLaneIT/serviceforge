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

	"github.com/SoftLaneIT/serviceforge/packages/go-common/config"
	"github.com/SoftLaneIT/serviceforge/packages/go-common/tenant"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusOK, map[string]any{"service": "config-service", "status": "ok"})
	})
	mux.HandleFunc("/v1/config/booking", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			respondJSON(w, http.StatusOK, map[string]any{
				"tenantId":            tenant.FromContext(r.Context()),
				"slotDurationMinutes": 30,
				"maxBookingsPerDay":   100,
				"autoConfirm":         true,
			})
		case http.MethodPut:
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, "invalid payload", http.StatusBadRequest)
				return
			}
			respondJSON(w, http.StatusOK, map[string]any{
				"tenantId": tenant.FromContext(r.Context()),
				"updated":  payload,
			})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	port := config.GetEnv("PORT", "8084")
	log.Printf("config-service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, tenant.Middleware(mux)); err != nil {
		log.Fatal(err)
	}
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
