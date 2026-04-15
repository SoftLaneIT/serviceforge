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
		respondJSON(w, http.StatusOK, map[string]any{"service": "api-gateway", "status": "ok"})
	})
	mux.HandleFunc("/v1/tenant/context", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusOK, map[string]any{"tenantId": tenant.FromContext(r.Context())})
	})

	// Phase 1 placeholder routes.
	mux.HandleFunc("/v1/bookings", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusOK, map[string]any{
			"message": "Route is wired at the gateway.",
			"next": "Forward to booking-service in the next increment.",
			"tenantId": tenant.FromContext(r.Context()),
		})
	})
	mux.HandleFunc("/v1/config/booking", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusOK, map[string]any{
			"message": "Route is wired at the gateway.",
			"next": "Forward to config-service in the next increment.",
			"tenantId": tenant.FromContext(r.Context()),
		})
	})

	port := config.GetEnv("PORT", "8081")
	log.Printf("api-gateway listening on :%s", port)
	if err := http.ListenAndServe(":"+port, tenant.Middleware(mux)); err != nil {
		log.Fatal(err)
	}
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
