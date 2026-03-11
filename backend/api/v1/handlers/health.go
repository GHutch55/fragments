package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthyHandler struct {
	DB *pgxpool.Pool
}

type healthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (h *HealthyHandler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.DB == nil {
		http.Error(w, "database pool not initialized", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	err := h.DB.Ping(ctx)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)

		response := healthResponse{
			Status:  "error",
			Message: "Database unreachable: " + err.Error(),
		}

		json.NewEncoder(w).Encode(response)
		return
	}

	response := healthResponse{
		Status:  "ok",
		Message: "Fragments API is running",
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
