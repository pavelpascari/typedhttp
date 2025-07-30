package typedhttp

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Common error types that can be used across applications

// ValidationError represents a request validation error.
type ValidationError struct {
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func (e *ValidationError) Error() string {
	return e.Message
}

// NewValidationError creates a new validation error.
func NewValidationError(message string, fields map[string]string) *ValidationError {
	return &ValidationError{
		Message: message,
		Fields:  fields,
	}
}

// NotFoundError represents a resource not found error.
type NotFoundError struct {
	Resource string `json:"resource"`
	ID       string `json:"id"`
	Message  string `json:"message"`
}

func (e *NotFoundError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("%s with id '%s' not found", e.Resource, e.ID)
}

// NewNotFoundError creates a new not found error.
func NewNotFoundError(resource, id string) *NotFoundError {
	return &NotFoundError{
		Resource: resource,
		ID:       id,
	}
}

// ConflictError represents a business logic conflict error.
type ConflictError struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

func (e *ConflictError) Error() string {
	return e.Message
}

// NewConflictError creates a new conflict error.
func NewConflictError(message string) *ConflictError {
	return &ConflictError{
		Message: message,
	}
}

// UnauthorizedError represents an authentication error.
type UnauthorizedError struct {
	Message string `json:"message"`
}

func (e *UnauthorizedError) Error() string {
	return e.Message
}

// NewUnauthorizedError creates a new unauthorized error.
func NewUnauthorizedError(message string) *UnauthorizedError {
	return &UnauthorizedError{
		Message: message,
	}
}

// ForbiddenError represents an authorization error.
type ForbiddenError struct {
	Message string `json:"message"`
}

func (e *ForbiddenError) Error() string {
	return e.Message
}

// NewForbiddenError creates a new forbidden error.
func NewForbiddenError(message string) *ForbiddenError {
	return &ForbiddenError{
		Message: message,
	}
}

// ErrorResponse represents a standardized error response.
type ErrorResponse struct {
	Error     string      `json:"error"`
	Code      string      `json:"code,omitempty"`
	Details   interface{} `json:"details,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
}

// DefaultErrorMapper provides a default implementation of ErrorMapper.
type DefaultErrorMapper struct{}

// MapError maps application errors to HTTP status codes and responses.
func (m *DefaultErrorMapper) MapError(err error) (int, interface{}) {
	var valErr *ValidationError
	if errors.As(err, &valErr) {
		return http.StatusBadRequest, ErrorResponse{
			Error:   "Validation failed",
			Code:    "VALIDATION_ERROR",
			Details: valErr.Fields,
		}
	}

	var nfErr *NotFoundError
	if errors.As(err, &nfErr) {
		return http.StatusNotFound, ErrorResponse{
			Error: nfErr.Error(),
			Code:  "NOT_FOUND",
		}
	}

	var conflictErr *ConflictError
	if errors.As(err, &conflictErr) {
		return http.StatusConflict, ErrorResponse{
			Error: conflictErr.Message,
			Code:  "CONFLICT",
		}
	}

	var authErr *UnauthorizedError
	if errors.As(err, &authErr) {
		return http.StatusUnauthorized, ErrorResponse{
			Error: authErr.Message,
			Code:  "UNAUTHORIZED",
		}
	}

	var forbErr *ForbiddenError
	if errors.As(err, &forbErr) {
		return http.StatusForbidden, ErrorResponse{
			Error: forbErr.Message,
			Code:  "FORBIDDEN",
		}
	}

	// Handle JSON parse errors as bad requests
	if strings.Contains(err.Error(), "invalid JSON") || 
	   strings.Contains(err.Error(), "invalid character") ||
	   strings.Contains(err.Error(), "unexpected end of JSON") {
		return http.StatusBadRequest, ErrorResponse{
			Error: "Invalid JSON in request body",
			Code:  "INVALID_JSON",
		}
	}

	// Log internal errors but don't expose details
	return http.StatusInternalServerError, ErrorResponse{
		Error: "Internal server error",
		Code:  "INTERNAL_ERROR",
	}
}