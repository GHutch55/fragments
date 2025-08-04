package handlers

import (
	"encoding/json"
	"net/http"
)

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"name":    "Fragments API",
		"version": "0.1.0",
		"status":  "development",
		"routes": map[string]string{
			"health": "/health",
			"api":    "/api",
		},
	}
	json.NewEncoder(w).Encode(response)
}
