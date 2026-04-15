package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/SoftLaneIT/serviceforge/packages/go-common/config"
)

type createTenantRequest struct {
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	PlanID string `json:"planId"`
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusOK, map[string]any{"service": "tenant-service", "status": "ok"})
	})
	mux.HandleFunc("/v1/tenants", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var req createTenantRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid payload", http.StatusBadRequest)
				return
			}
			respondJSON(w, http.StatusCreated, map[string]any{
				"id":     "tenant_001",
				"name":   req.Name,
				"slug":   req.Slug,
				"planId": req.PlanID,
			})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	port := config.GetEnv("PORT", "8083")
	log.Printf("tenant-service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
