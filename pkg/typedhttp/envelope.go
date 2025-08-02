package typedhttp

import (
	"context"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

// APIResponse represents a standard response envelope that wraps handler responses
// with additional metadata and error handling capabilities.
type APIResponse[T any] struct {
	Data    *T            `json:"data,omitempty"`
	Error   *string       `json:"error,omitempty"`
	Success bool          `json:"success"`
	Meta    *ResponseMeta `json:"meta,omitempty"`
}

// ResponseMeta contains additional metadata about the response.
type ResponseMeta struct {
	RequestID string `json:"request_id,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

// ResponseEnvelopeMiddleware wraps responses in a standard envelope structure.
// It implements both TypedPostMiddleware for runtime behavior and ResponseSchemaModifier
// for OpenAPI schema generation.
type ResponseEnvelopeMiddleware[TResponse any] struct {
	includeRequestID bool
	includeTimestamp bool
	includeMeta      bool
}

// NewResponseEnvelopeMiddleware creates a new response envelope middleware with the specified options.
func NewResponseEnvelopeMiddleware[TResponse any](opts ...EnvelopeOption) *ResponseEnvelopeMiddleware[TResponse] {
	config := &EnvelopeConfig{
		IncludeRequestID: true,
		IncludeTimestamp: true,
		IncludeMeta:      true,
	}

	for _, opt := range opts {
		opt(config)
	}

	return &ResponseEnvelopeMiddleware[TResponse]{
		includeRequestID: config.IncludeRequestID,
		includeTimestamp: config.IncludeTimestamp,
		includeMeta:      config.IncludeMeta,
	}
}

// EnvelopeConfig configures the envelope middleware behavior.
type EnvelopeConfig struct {
	IncludeRequestID bool
	IncludeTimestamp bool
	IncludeMeta      bool
}

// EnvelopeOption configures envelope middleware.
type EnvelopeOption func(*EnvelopeConfig)

// WithRequestID enables or disables request ID inclusion in the envelope meta.
func WithRequestID(include bool) EnvelopeOption {
	return func(config *EnvelopeConfig) {
		config.IncludeRequestID = include
	}
}

// WithTimestamp enables or disables timestamp inclusion in the envelope meta.
func WithTimestamp(include bool) EnvelopeOption {
	return func(config *EnvelopeConfig) {
		config.IncludeTimestamp = include
	}
}

// WithMeta enables or disables the entire meta section in the envelope.
func WithMeta(include bool) EnvelopeOption {
	return func(config *EnvelopeConfig) {
		config.IncludeMeta = include
		if !include {
			config.IncludeRequestID = false
			config.IncludeTimestamp = false
		}
	}
}

// After implements TypedPostMiddleware, wrapping the response in an envelope structure.
func (m *ResponseEnvelopeMiddleware[TResponse]) After(ctx context.Context, resp *TResponse) (*APIResponse[TResponse], error) {
	envelope := &APIResponse[TResponse]{
		Data:    resp,
		Success: true,
	}

	if m.includeMeta {
		meta := &ResponseMeta{}
		hasMetaData := false

		if m.includeRequestID {
			if requestID := ctx.Value("request_id"); requestID != nil {
				if reqIDStr, ok := requestID.(string); ok && reqIDStr != "" {
					meta.RequestID = reqIDStr
					hasMetaData = true
				}
			}
		}

		if m.includeTimestamp {
			meta.Timestamp = time.Now().Format(time.RFC3339)
			hasMetaData = true
		}

		if hasMetaData {
			envelope.Meta = meta
		}
	}

	return envelope, nil
}

// ModifyResponseSchema implements ResponseSchemaModifier, transforming the OpenAPI schema
// to reflect the envelope structure that will be returned to clients.
func (m *ResponseEnvelopeMiddleware[TResponse]) ModifyResponseSchema(ctx context.Context, originalSchema *openapi3.SchemaRef) (*openapi3.SchemaRef, error) {
	envelopeSchema := &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			Type:        &openapi3.Types{"object"},
			Description: "Standard response envelope wrapping the actual response data",
			Properties: map[string]*openapi3.SchemaRef{
				"data": {
					Value: &openapi3.Schema{
						Description: "The actual response data",
						OneOf:       []*openapi3.SchemaRef{originalSchema},
						Nullable:    true,
					},
				},
				"error": {
					Value: &openapi3.Schema{
						Type:        &openapi3.Types{"string"},
						Description: "Error message when success is false",
						Nullable:    true,
					},
				},
				"success": {
					Value: &openapi3.Schema{
						Type:        &openapi3.Types{"boolean"},
						Description: "Indicates whether the request was successful",
					},
				},
			},
			Required: []string{"success"},
		},
	}

	if m.includeMeta {
		metaProperties := make(map[string]*openapi3.SchemaRef)

		if m.includeRequestID {
			metaProperties["request_id"] = &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type:        &openapi3.Types{"string"},
					Description: "Unique identifier for the request",
				},
			}
		}

		if m.includeTimestamp {
			metaProperties["timestamp"] = &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type:        &openapi3.Types{"string"},
					Format:      "date-time",
					Description: "Timestamp when the response was generated",
				},
			}
		}

		if len(metaProperties) > 0 {
			envelopeSchema.Value.Properties["meta"] = &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type:        &openapi3.Types{"object"},
					Description: "Additional metadata about the response",
					Properties:  metaProperties,
					Nullable:    true,
				},
			}
		}
	}

	return envelopeSchema, nil
}

// ErrorEnvelopeMiddleware wraps errors in the same envelope structure for consistency.
// This middleware should be placed early in the chain to catch errors from other middleware.
type ErrorEnvelopeMiddleware[TRequest, TResponse any] struct {
	includeRequestID bool
	includeTimestamp bool
	includeMeta      bool
}

// NewErrorEnvelopeMiddleware creates a new error envelope middleware.
func NewErrorEnvelopeMiddleware[TRequest, TResponse any](opts ...EnvelopeOption) *ErrorEnvelopeMiddleware[TRequest, TResponse] {
	config := &EnvelopeConfig{
		IncludeRequestID: true,
		IncludeTimestamp: true,
		IncludeMeta:      true,
	}

	for _, opt := range opts {
		opt(config)
	}

	return &ErrorEnvelopeMiddleware[TRequest, TResponse]{
		includeRequestID: config.IncludeRequestID,
		includeTimestamp: config.IncludeTimestamp,
		includeMeta:      config.IncludeMeta,
	}
}

// Before implements TypedMiddleware.Before (no-op for error handling).
func (m *ErrorEnvelopeMiddleware[TRequest, TResponse]) Before(ctx context.Context, req *TRequest) (context.Context, error) {
	return ctx, nil
}

// After implements TypedMiddleware.After, wrapping errors in envelope structure.
func (m *ErrorEnvelopeMiddleware[TRequest, TResponse]) After(ctx context.Context, req *TRequest, resp *TResponse, err error) (*TResponse, error) {
	if err != nil {
		// Convert error to envelope format
		errorMessage := err.Error()
		envelope := &APIResponse[TResponse]{
			Data:    nil,
			Error:   &errorMessage,
			Success: false,
		}

		if m.includeMeta {
			meta := &ResponseMeta{}
			hasMetaData := false

			if m.includeRequestID {
				if requestID := ctx.Value("request_id"); requestID != nil {
					if reqIDStr, ok := requestID.(string); ok && reqIDStr != "" {
						meta.RequestID = reqIDStr
						hasMetaData = true
					}
				}
			}

			if m.includeTimestamp {
				meta.Timestamp = time.Now().Format(time.RFC3339)
				hasMetaData = true
			}

			if hasMetaData {
				envelope.Meta = meta
			}
		}

		// Note: This is a type assertion that will need to be handled carefully
		// in the actual implementation, possibly with interface{} and type conversion
		return resp, &EnvelopeError{Envelope: envelope}
	}

	return resp, nil
}

// ModifyResponseSchema implements ResponseSchemaModifier for error responses.
func (m *ErrorEnvelopeMiddleware[TRequest, TResponse]) ModifyResponseSchema(ctx context.Context, originalSchema *openapi3.SchemaRef) (*openapi3.SchemaRef, error) {
	// This middleware doesn't modify successful response schemas,
	// but it does ensure error responses follow the envelope pattern
	return originalSchema, nil
}

// EnvelopeError represents an error that has been wrapped in an envelope structure.
type EnvelopeError struct {
	Envelope interface{}
}

func (e *EnvelopeError) Error() string {
	if env, ok := e.Envelope.(*APIResponse[any]); ok && env.Error != nil {
		return *env.Error
	}
	return "envelope error"
}