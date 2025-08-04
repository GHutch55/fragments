package handlers

import (
	"encoding/json"
	"net/http"
)

func ApiInfoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"message": "Fragments API v0.1.0",
		"endpoints": map[string]string{
			"snippets": "Coming soon",
			"users":    "Coming soon",
			"tags":     "Coming soon",
		},
	}
	json.NewEncoder(w).Encode(response)
}
