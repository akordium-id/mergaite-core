package response

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
)

type Envelope struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Meta    any    `json:"meta,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// JSON writes a successful JSON response.
func JSON(w http.ResponseWriter, status int, data any, meta ...any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	env := Envelope{
		Success: true,
		Data:    data,
	}
	if len(meta) > 0 {
		env.Meta = meta[0]
	}

	_ = json.NewEncoder(w).Encode(env)
}

// Err writes an error JSON response.
func Err(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	env := Envelope{
		Success: false,
		Error: &Error{
			Code:    code,
			Message: message,
		},
	}

	_ = json.NewEncoder(w).Encode(env)
}

// FromDomainError automatically translates domain errors into appropriate HTTP status codes and payloads.
func FromDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, shared.ErrNotFound):
		Err(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, shared.ErrAlreadyExists):
		Err(w, http.StatusConflict, "ALREADY_EXISTS", err.Error())
	case errors.Is(err, shared.ErrInvalidInput),
		errors.Is(err, shared.ErrInvalidCurrency),
		errors.Is(err, shared.ErrInvalidAmount),
		errors.Is(err, shared.ErrUnitMismatch),
		errors.Is(err, shared.ErrCurrencyMismatch):
		Err(w, http.StatusBadRequest, "INVALID_INPUT", err.Error())
	case errors.Is(err, shared.ErrTenantRequired):
		Err(w, http.StatusBadRequest, "TENANT_REQUIRED", "X-Tenant-ID header is missing or invalid")
	case errors.Is(err, shared.ErrTenantSuspended):
		Err(w, http.StatusForbidden, "TENANT_SUSPENDED", "Tenant account is suspended")
	case errors.Is(err, shared.ErrUnauthorized):
		Err(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
	case errors.Is(err, shared.ErrForbidden):
		Err(w, http.StatusForbidden, "FORBIDDEN", err.Error())
	default:
		Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
	}
}
