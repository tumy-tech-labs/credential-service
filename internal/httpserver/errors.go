package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/bradtumy/credential-service/internal/domain"
	"github.com/bradtumy/credential-service/internal/version"
)

// APIError provides a consistent error body for HTTP handlers with enhanced error information.
type APIError struct {
	Error       string       `json:"error"`
	Description string       `json:"description,omitempty"`
	Code        string       `json:"code"`
	APIVersion  string       `json:"api_version"`
	Fields      []FieldError `json:"fields,omitempty"`
	TraceID     string       `json:"trace_id,omitempty"`
}

// FieldError provides detailed field-level validation errors.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// WriteAPIError writes a standardized API error response.
func WriteAPIError(w http.ResponseWriter, status int, code string, description string) {
	WriteAPIErrorWithFields(w, status, code, description, nil)
}

// WriteAPIErrorWithFields writes a standardized API error response with field-level errors.
func WriteAPIErrorWithFields(w http.ResponseWriter, status int, code string, description string, fields []FieldError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIError{
		Error:       code,
		Description: description,
		Code:        code,
		APIVersion:  version.APIVersion,
		Fields:      fields,
	})
}

// WriteValidationError writes a validation error with better developer experience.
func WriteValidationError(w http.ResponseWriter, err error) {
	var validationErr domain.ValidationError
	if errors.As(err, &validationErr) {
		fields := []FieldError{}
		if validationErr.Field != "" {
			fields = append(fields, FieldError{
				Field:   validationErr.Field,
				Message: validationErr.Message,
				Code:    validationErr.Code,
			})
		}
		WriteAPIErrorWithFields(w, http.StatusBadRequest, "validation_error", "Invalid request data", fields)
	} else {
		WriteAPIError(w, http.StatusBadRequest, "validation_error", err.Error())
	}
}

// WriteDIDError writes a DID-specific error with helpful information.
func WriteDIDError(w http.ResponseWriter, err error) {
	WriteAPIError(w, http.StatusBadRequest, "invalid_did", "Invalid DID format: "+err.Error())
}

// WriteInternalError writes an internal server error while hiding implementation details.
func WriteInternalError(w http.ResponseWriter, message string) {
	WriteAPIError(w, http.StatusInternalServerError, "internal_error", message)
}
