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
		respondJSON(w, http.StatusOK, map[string]any{"service": "auth-service", "status": "ok"})
	})
	mux.HandleFunc("/v1/tokens", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		respondJSON(w, http.StatusCreated, map[string]any{
			"tenantId":    tenant.FromContext(r.Context()),
			"accessToken": "stub-access-token",
			"tokenType":   "Bearer",
		})
	})

	port := config.GetEnv("PORT", "8082")
	log.Printf("auth-service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, tenant.Middleware(mux)); err != nil {
		log.Fatal(err)
	}
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
