// Package testutil provides comprehensive utilities for testing TypedHTTP handlers
// with Go-idiomatic patterns including context support, explicit error handling,
// and focused interfaces.
package testutil

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// Request represents an HTTP request with all necessary data.
// This struct follows Go conventions by being explicit and self-contained.
type Request struct {
	Method      string
	Path        string
	PathParams  map[string]string
	QueryParams map[string]string
	Headers     map[string]string
	Cookies     map[string]string
	Body        interface{}
	Files       map[string][]byte
}

// Response represents an HTTP response (generic-free base type).
// Generics are only used where they add real type safety value.
type Response struct {
	StatusCode int
	Headers    http.Header
	Raw        []byte
}

// TypedResponse wraps Response with typed data for when type safety is needed.
type TypedResponse[T any] struct {
	*Response
	Data T
}

// HTTPClient defines the main client interface for HTTP testing.
// Go interfaces cannot have type parameters, so ExecuteTyped is a concrete method.
type HTTPClient interface {
	Execute(ctx context.Context, req Request) (*Response, error)
	// ExecuteTyped is implemented as a concrete method with generics
}

// RequestError provides context-aware error handling for HTTP requests.
type RequestError struct {
	Method string
	Path   string
	Err    error
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("request %s %s failed: %v", e.Method, e.Path, e.Err)
}

func (e *RequestError) Unwrap() error {
	return e.Err
}

// ValidationError represents validation failures with field-specific context.
type ValidationError struct {
	Field   string
	Message string
	Err     error
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field %q: %s", e.Field, e.Message)
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}

// IsRequestError checks if an error is a RequestError using Go 1.13+ error patterns.
func IsRequestError(err error) bool {
	var reqErr *RequestError

	return errors.As(err, &reqErr)
}

// IsValidationError checks if an error is a ValidationError.
func IsValidationError(err error) bool {
	var valErr *ValidationError

	return errors.As(err, &valErr)
}
