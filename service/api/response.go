package api

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse matches the Error schema in api.yaml
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// sendJSON writes a JSON response with the given status code
func sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

// sendError writes a JSON error response matching the Error schema
func sendError(w http.ResponseWriter, status int, code string, message string) {
	sendJSON(w, status, ErrorResponse{
		Code:    code,
		Message: message,
	})
}

// Common error helpers
func sendBadRequest(w http.ResponseWriter, message string) {
	sendError(w, http.StatusBadRequest, "bad-request", message)
}

func sendUnauthorized(w http.ResponseWriter, message string) {
	sendError(w, http.StatusUnauthorized, "unauthorized", message)
}

func sendForbidden(w http.ResponseWriter, message string) {
	sendError(w, http.StatusForbidden, "forbidden", message)
}

func sendNotFound(w http.ResponseWriter, message string) {
	sendError(w, http.StatusNotFound, "not-found", message)
}

func sendConflict(w http.ResponseWriter, message string) {
	sendError(w, http.StatusConflict, "conflict", message)
}

func sendInternalError(w http.ResponseWriter, message string) {
	sendError(w, http.StatusInternalServerError, "internal-error", message)
}
