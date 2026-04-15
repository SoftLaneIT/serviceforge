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
