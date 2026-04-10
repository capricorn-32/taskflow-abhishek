package httpapi

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse is the standard error payload returned by API endpoints.
type ErrorResponse struct {
	Error  string            `json:"error"`
	Fields map[string]string `json:"fields,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func WriteNoContent(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}

func WriteValidationError(w http.ResponseWriter, fields map[string]string) {
	WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "validation failed", Fields: fields})
}

func WriteUnauthorized(w http.ResponseWriter) {
	WriteJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
}

func WriteForbidden(w http.ResponseWriter) {
	WriteJSON(w, http.StatusForbidden, ErrorResponse{Error: "forbidden"})
}

func WriteNotFound(w http.ResponseWriter) {
	WriteJSON(w, http.StatusNotFound, ErrorResponse{Error: "not found"})
}

func WriteInternal(w http.ResponseWriter) {
	WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
}
