package typedhttp

import (
	"context"
	"io"
	"net/http"
)

// Handler represents the core business logic interface (transport-agnostic).
// This is the main interface that business logic handlers should implement.
type Handler[TRequest, TResponse any] interface {
	Handle(ctx context.Context, req TRequest) (TResponse, error)
}

// Service is an alias for Handler following Go's idiomatic naming conventions.
// Use this when you want to emphasize that this is a service layer component.
type Service[TRequest, TResponse any] interface {
	Execute(ctx context.Context, req TRequest) (TResponse, error)
}

// RequestDecoder handles decoding HTTP requests into typed request objects.
type RequestDecoder[T any] interface {
	Decode(r *http.Request) (T, error)
	ContentTypes() []string
}

// ResponseEncoder handles encoding typed response objects into HTTP responses.
type ResponseEncoder[T any] interface {
	Encode(w http.ResponseWriter, data T, statusCode int) error
	ContentType() string
}

// ErrorMapper maps application errors to HTTP status codes and response bodies.
type ErrorMapper interface {
	MapError(err error) (statusCode int, response interface{})
}

// StreamingResponse represents a response that should be streamed to the client.
type StreamingResponse struct {
	ContentType string
	Filename    string
	Stream      io.Reader
	StatusCode  int
}

// Middleware represents HTTP middleware following the standard Go pattern.
type Middleware func(http.Handler) http.Handler

// HandlerOption allows configuration of HTTP handlers during registration.
type HandlerOption func(*HandlerConfig)

// HandlerConfig contains all configuration options for a typed handler.
type HandlerConfig struct {
	Decoder       interface{} // RequestDecoder[T]
	Encoder       interface{} // ResponseEncoder[T]
	ErrorMapper   ErrorMapper
	Middleware    []Middleware
	Metadata      OpenAPIMetadata
	Observability ObservabilityConfig
}

// OpenAPIMetadata contains metadata for OpenAPI specification generation.
type OpenAPIMetadata struct {
	Summary     string                  `json:"summary,omitempty"`
	Description string                  `json:"description,omitempty"`
	Tags        []string                `json:"tags,omitempty"`
	Parameters  []ParameterSpec         `json:"parameters,omitempty"`
	RequestBody *RequestBodySpec        `json:"requestBody,omitempty"`
	Responses   map[string]ResponseSpec `json:"responses,omitempty"`
}

// ParameterSpec defines an OpenAPI parameter specification.
type ParameterSpec struct {
	Name        string      `json:"name"`
	In          string      `json:"in"` // query, path, header, cookie
	Description string      `json:"description,omitempty"`
	Required    bool        `json:"required,omitempty"`
	Schema      interface{} `json:"schema,omitempty"`
}

// RequestBodySpec defines an OpenAPI request body specification.
type RequestBodySpec struct {
	Description string                 `json:"description,omitempty"`
	Required    bool                   `json:"required,omitempty"`
	Content     map[string]MediaType   `json:"content"`
}

// ResponseSpec defines an OpenAPI response specification.
type ResponseSpec struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
	Headers     map[string]Header    `json:"headers,omitempty"`
}

// MediaType defines an OpenAPI media type specification.
type MediaType struct {
	Schema  interface{} `json:"schema,omitempty"`
	Example interface{} `json:"example,omitempty"`
}

// Header defines an OpenAPI header specification.
type Header struct {
	Description string      `json:"description,omitempty"`
	Schema      interface{} `json:"schema,omitempty"`
}

// ObservabilityConfig contains configuration for observability features.
type ObservabilityConfig struct {
	Tracing         bool                   `json:"tracing,omitempty"`
	Metrics         bool                   `json:"metrics,omitempty"`
	Logging         bool                   `json:"logging,omitempty"`
	TraceAttributes map[string]interface{} `json:"trace_attributes,omitempty"`
	MetricLabels    map[string]string      `json:"metric_labels,omitempty"`
}