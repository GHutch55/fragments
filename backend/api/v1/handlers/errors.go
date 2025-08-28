package handlers

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse represents a JSON error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// APIResponse represents a standardized API response
type APIResponse struct {
	Data       interface{}            `json:"data"`
	Pagination *PaginationInfo        `json:"pagination,omitempty"`
	Meta       map[string]interface{} `json:"meta,omitempty"`
}

// PaginationInfo represents pagination metadata
type PaginationInfo struct {
	Page       int  `json:"page"`
	Limit      int  `json:"limit"`
	Total      int  `json:"total"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

// SendError sends a standardized JSON error response
func SendError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	}
	json.NewEncoder(w).Encode(response)
}

// SendErrorWithCode sends a JSON error response with a custom error code
func SendErrorWithCode(w http.ResponseWriter, message, code string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
		Code:    code,
	}
	json.NewEncoder(w).Encode(response)
}

// SendData sends a successful response with data
func SendData(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(APIResponse{Data: data})
}

// SendPaginatedData sends a successful response with pagination
func SendPaginatedData(w http.ResponseWriter, data interface{}, pagination *PaginationInfo, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(APIResponse{
		Data:       data,
		Pagination: pagination,
	})
}
