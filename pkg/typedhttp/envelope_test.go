package typedhttp

import (
	"context"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types for envelope testing
type EnvelopeTestResponseData struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

type EnvelopeTestRequest struct {
	Name string `json:"name"`
}

// TestAPIResponse tests the basic envelope structure
func TestAPIResponse(t *testing.T) {
	t.Run("successful response with data", func(t *testing.T) {
		data := &EnvelopeTestResponseData{ID: "123", Message: "test"}
		response := &APIResponse[EnvelopeTestResponseData]{
			Data:    data,
			Success: true,
		}

		assert.True(t, response.Success)
		assert.Equal(t, data, response.Data)
		assert.Nil(t, response.Error)
	})

	t.Run("error response", func(t *testing.T) {
		errorMsg := "something went wrong"
		response := &APIResponse[EnvelopeTestResponseData]{
			Error:   &errorMsg,
			Success: false,
		}

		assert.False(t, response.Success)
		assert.Nil(t, response.Data)
		assert.Equal(t, &errorMsg, response.Error)
	})

	t.Run("response with metadata", func(t *testing.T) {
		data := &EnvelopeTestResponseData{ID: "123", Message: "test"}
		meta := &ResponseMeta{
			RequestID: "req-123",
			Timestamp: "2023-01-01T12:00:00Z",
		}
		response := &APIResponse[EnvelopeTestResponseData]{
			Data:    data,
			Success: true,
			Meta:    meta,
		}

		assert.True(t, response.Success)
		assert.Equal(t, data, response.Data)
		assert.Equal(t, meta, response.Meta)
		assert.Equal(t, "req-123", response.Meta.RequestID)
		assert.Equal(t, "2023-01-01T12:00:00Z", response.Meta.Timestamp)
	})
}

// TestResponseEnvelopeMiddleware tests the response envelope middleware
func TestResponseEnvelopeMiddleware(t *testing.T) {
	t.Run("wraps response with default options", func(t *testing.T) {
		middleware := NewResponseEnvelopeMiddleware[EnvelopeTestResponseData]()
		
		// Create context with request ID
		ctx := context.WithValue(context.Background(), "request_id", "test-request-123")
		
		originalResponse := &EnvelopeTestResponseData{ID: "123", Message: "test"}
		
		envelope, err := middleware.After(ctx, originalResponse)
		require.NoError(t, err)
		require.NotNil(t, envelope)
		
		assert.True(t, envelope.Success)
		assert.Equal(t, originalResponse, envelope.Data)
		assert.Nil(t, envelope.Error)
		assert.NotNil(t, envelope.Meta)
		assert.Equal(t, "test-request-123", envelope.Meta.RequestID)
		assert.NotEmpty(t, envelope.Meta.Timestamp)
		
		// Verify timestamp format
		_, err = time.Parse(time.RFC3339, envelope.Meta.Timestamp)
		assert.NoError(t, err)
	})

	t.Run("wraps response without request ID in context", func(t *testing.T) {
		middleware := NewResponseEnvelopeMiddleware[EnvelopeTestResponseData]()
		ctx := context.Background()
		
		originalResponse := &EnvelopeTestResponseData{ID: "123", Message: "test"}
		
		envelope, err := middleware.After(ctx, originalResponse)
		require.NoError(t, err)
		require.NotNil(t, envelope)
		
		assert.True(t, envelope.Success)
		assert.Equal(t, originalResponse, envelope.Data)
		assert.NotNil(t, envelope.Meta)
		assert.Empty(t, envelope.Meta.RequestID) // No request ID in context
		assert.NotEmpty(t, envelope.Meta.Timestamp)
	})

	t.Run("respects WithRequestID(false) option", func(t *testing.T) {
		middleware := NewResponseEnvelopeMiddleware[EnvelopeTestResponseData](
			WithRequestID(false),
		)
		
		ctx := context.WithValue(context.Background(), "request_id", "test-request-123")
		originalResponse := &EnvelopeTestResponseData{ID: "123", Message: "test"}
		
		envelope, err := middleware.After(ctx, originalResponse)
		require.NoError(t, err)
		require.NotNil(t, envelope)
		
		assert.True(t, envelope.Success)
		assert.NotNil(t, envelope.Meta)
		assert.Empty(t, envelope.Meta.RequestID) // Should be empty due to option
		assert.NotEmpty(t, envelope.Meta.Timestamp)
	})

	t.Run("respects WithTimestamp(false) option", func(t *testing.T) {
		middleware := NewResponseEnvelopeMiddleware[EnvelopeTestResponseData](
			WithTimestamp(false),
		)
		
		ctx := context.WithValue(context.Background(), "request_id", "test-request-123")
		originalResponse := &EnvelopeTestResponseData{ID: "123", Message: "test"}
		
		envelope, err := middleware.After(ctx, originalResponse)
		require.NoError(t, err)
		require.NotNil(t, envelope)
		
		assert.True(t, envelope.Success)
		assert.NotNil(t, envelope.Meta)
		assert.Equal(t, "test-request-123", envelope.Meta.RequestID)
		assert.Empty(t, envelope.Meta.Timestamp) // Should be empty due to option
	})

	t.Run("respects WithMeta(false) option", func(t *testing.T) {
		middleware := NewResponseEnvelopeMiddleware[EnvelopeTestResponseData](
			WithMeta(false),
		)
		
		ctx := context.WithValue(context.Background(), "request_id", "test-request-123")
		originalResponse := &EnvelopeTestResponseData{ID: "123", Message: "test"}
		
		envelope, err := middleware.After(ctx, originalResponse)
		require.NoError(t, err)
		require.NotNil(t, envelope)
		
		assert.True(t, envelope.Success)
		assert.Equal(t, originalResponse, envelope.Data)
		assert.Nil(t, envelope.Meta) // Should be nil due to option
	})

	t.Run("WithMeta(false) disables RequestID and Timestamp", func(t *testing.T) {
		middleware := NewResponseEnvelopeMiddleware[EnvelopeTestResponseData](
			WithMeta(false),
			WithRequestID(true), // Should be overridden by WithMeta(false)
			WithTimestamp(true), // Should be overridden by WithMeta(false)
		)
		
		ctx := context.WithValue(context.Background(), "request_id", "test-request-123")
		originalResponse := &EnvelopeTestResponseData{ID: "123", Message: "test"}
		
		envelope, err := middleware.After(ctx, originalResponse)
		require.NoError(t, err)
		require.NotNil(t, envelope)
		
		assert.True(t, envelope.Success)
		assert.Nil(t, envelope.Meta) // Meta should be completely disabled
	})
}

// TestResponseEnvelopeMiddleware_OpenAPISchema tests OpenAPI schema modification
func TestResponseEnvelopeMiddleware_OpenAPISchema(t *testing.T) {
	t.Run("modifies schema with all metadata", func(t *testing.T) {
		middleware := NewResponseEnvelopeMiddleware[EnvelopeTestResponseData]()
		
		// Create original schema
		originalSchema := &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Type: &openapi3.Types{"object"},
				Properties: map[string]*openapi3.SchemaRef{
					"id": {
						Value: &openapi3.Schema{
							Type: &openapi3.Types{"string"},
						},
					},
					"message": {
						Value: &openapi3.Schema{
							Type: &openapi3.Types{"string"},
						},
					},
				},
			},
		}
		
		ctx := context.Background()
		envelopeSchema, err := middleware.ModifyResponseSchema(ctx, originalSchema)
		require.NoError(t, err)
		require.NotNil(t, envelopeSchema)
		require.NotNil(t, envelopeSchema.Value)
		
		// Verify envelope structure
		assert.Equal(t, &openapi3.Types{"object"}, envelopeSchema.Value.Type)
		assert.Contains(t, envelopeSchema.Value.Properties, "data")
		assert.Contains(t, envelopeSchema.Value.Properties, "error")
		assert.Contains(t, envelopeSchema.Value.Properties, "success")
		assert.Contains(t, envelopeSchema.Value.Properties, "meta")
		assert.Equal(t, []string{"success"}, envelopeSchema.Value.Required)
		
		// Verify data field references original schema
		dataSchema := envelopeSchema.Value.Properties["data"]
		assert.True(t, dataSchema.Value.Nullable)
		assert.Len(t, dataSchema.Value.OneOf, 1)
		assert.Equal(t, originalSchema, dataSchema.Value.OneOf[0])
		
		// Verify meta structure
		metaSchema := envelopeSchema.Value.Properties["meta"]
		assert.True(t, metaSchema.Value.Nullable)
		assert.Equal(t, &openapi3.Types{"object"}, metaSchema.Value.Type)
		assert.Contains(t, metaSchema.Value.Properties, "request_id")
		assert.Contains(t, metaSchema.Value.Properties, "timestamp")
		
		// Verify timestamp format
		timestampSchema := metaSchema.Value.Properties["timestamp"]
		assert.Equal(t, "date-time", timestampSchema.Value.Format)
	})

	t.Run("modifies schema without meta", func(t *testing.T) {
		middleware := NewResponseEnvelopeMiddleware[EnvelopeTestResponseData](
			WithMeta(false),
		)
		
		originalSchema := &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Type: &openapi3.Types{"object"},
			},
		}
		
		ctx := context.Background()
		envelopeSchema, err := middleware.ModifyResponseSchema(ctx, originalSchema)
		require.NoError(t, err)
		require.NotNil(t, envelopeSchema)
		
		// Should not include meta
		assert.NotContains(t, envelopeSchema.Value.Properties, "meta")
		assert.Contains(t, envelopeSchema.Value.Properties, "data")
		assert.Contains(t, envelopeSchema.Value.Properties, "error")
		assert.Contains(t, envelopeSchema.Value.Properties, "success")
	})

	t.Run("modifies schema with partial meta", func(t *testing.T) {
		middleware := NewResponseEnvelopeMiddleware[EnvelopeTestResponseData](
			WithRequestID(false),
			WithTimestamp(true),
		)
		
		originalSchema := &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Type: &openapi3.Types{"object"},
			},
		}
		
		ctx := context.Background()
		envelopeSchema, err := middleware.ModifyResponseSchema(ctx, originalSchema)
		require.NoError(t, err)
		require.NotNil(t, envelopeSchema)
		
		// Should include meta but only timestamp
		assert.Contains(t, envelopeSchema.Value.Properties, "meta")
		metaSchema := envelopeSchema.Value.Properties["meta"]
		assert.NotContains(t, metaSchema.Value.Properties, "request_id")
		assert.Contains(t, metaSchema.Value.Properties, "timestamp")
	})
}

// TestErrorEnvelopeMiddleware tests the error envelope middleware
func TestErrorEnvelopeMiddleware(t *testing.T) {
	t.Run("passes through successful responses", func(t *testing.T) {
		middleware := NewErrorEnvelopeMiddleware[EnvelopeTestRequest, EnvelopeTestResponseData]()
		
		ctx := context.Background()
		req := &EnvelopeTestRequest{Name: "test"}
		resp := &EnvelopeTestResponseData{ID: "123", Message: "success"}
		
		resultResp, resultErr := middleware.After(ctx, req, resp, nil)
		
		assert.Equal(t, resp, resultResp)
		assert.NoError(t, resultErr)
	})

	t.Run("wraps errors in envelope", func(t *testing.T) {
		middleware := NewErrorEnvelopeMiddleware[EnvelopeTestRequest, EnvelopeTestResponseData]()
		
		ctx := context.WithValue(context.Background(), "request_id", "error-request-123")
		req := &EnvelopeTestRequest{Name: "test"}
		resp := &EnvelopeTestResponseData{}
		originalError := assert.AnError
		
		resultResp, resultErr := middleware.After(ctx, req, resp, originalError)
		
		assert.Equal(t, resp, resultResp) // Original response passed through
		require.Error(t, resultErr)
		
		// Verify it's an EnvelopeError
		envelopeErr, ok := resultErr.(*EnvelopeError)
		require.True(t, ok)
		
		// Verify envelope structure
		envelope, ok := envelopeErr.Envelope.(*APIResponse[EnvelopeTestResponseData])
		require.True(t, ok)
		
		assert.False(t, envelope.Success)
		assert.Nil(t, envelope.Data)
		require.NotNil(t, envelope.Error)
		assert.Equal(t, originalError.Error(), *envelope.Error)
		assert.NotNil(t, envelope.Meta)
		assert.Equal(t, "error-request-123", envelope.Meta.RequestID)
		assert.NotEmpty(t, envelope.Meta.Timestamp)
	})

	t.Run("Before method is no-op", func(t *testing.T) {
		middleware := NewErrorEnvelopeMiddleware[EnvelopeTestRequest, EnvelopeTestResponseData]()
		
		ctx := context.Background()
		req := &EnvelopeTestRequest{Name: "test"}
		
		resultCtx, err := middleware.Before(ctx, req)
		
		assert.Equal(t, ctx, resultCtx)
		assert.NoError(t, err)
	})

	t.Run("ModifyResponseSchema returns original schema", func(t *testing.T) {
		middleware := NewErrorEnvelopeMiddleware[EnvelopeTestRequest, EnvelopeTestResponseData]()
		
		originalSchema := &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Type: &openapi3.Types{"object"},
			},
		}
		
		ctx := context.Background()
		resultSchema, err := middleware.ModifyResponseSchema(ctx, originalSchema)
		
		assert.Equal(t, originalSchema, resultSchema)
		assert.NoError(t, err)
	})
}

// TestEnvelopeError tests the envelope error type
func TestEnvelopeError(t *testing.T) {
	t.Run("returns error message from envelope", func(t *testing.T) {
		errorMsg := "test error message"
		envelope := &APIResponse[any]{
			Error:   &errorMsg,
			Success: false,
		}
		
		envelopeErr := &EnvelopeError{Envelope: envelope}
		
		assert.Equal(t, errorMsg, envelopeErr.Error())
	})

	t.Run("returns default message for invalid envelope", func(t *testing.T) {
		envelopeErr := &EnvelopeError{Envelope: "invalid"}
		
		assert.Equal(t, "envelope error", envelopeErr.Error())
	})

	t.Run("returns default message for nil error in envelope", func(t *testing.T) {
		envelope := &APIResponse[any]{
			Error:   nil,
			Success: false,
		}
		
		envelopeErr := &EnvelopeError{Envelope: envelope}
		
		assert.Equal(t, "envelope error", envelopeErr.Error())
	})
}

// TestEnvelopeOptions tests the configuration options
func TestEnvelopeOptions(t *testing.T) {
	t.Run("WithRequestID option", func(t *testing.T) {
		config := &EnvelopeConfig{}
		
		// Test enabling
		WithRequestID(true)(config)
		assert.True(t, config.IncludeRequestID)
		
		// Test disabling
		WithRequestID(false)(config)
		assert.False(t, config.IncludeRequestID)
	})

	t.Run("WithTimestamp option", func(t *testing.T) {
		config := &EnvelopeConfig{}
		
		// Test enabling
		WithTimestamp(true)(config)
		assert.True(t, config.IncludeTimestamp)
		
		// Test disabling
		WithTimestamp(false)(config)
		assert.False(t, config.IncludeTimestamp)
	})

	t.Run("WithMeta option enables/disables meta", func(t *testing.T) {
		config := &EnvelopeConfig{
			IncludeRequestID: true,
			IncludeTimestamp: true,
		}
		
		// Test enabling (should not affect other settings)
		WithMeta(true)(config)
		assert.True(t, config.IncludeMeta)
		assert.True(t, config.IncludeRequestID)
		assert.True(t, config.IncludeTimestamp)
		
		// Test disabling (should disable other meta-related settings)
		WithMeta(false)(config)
		assert.False(t, config.IncludeMeta)
		assert.False(t, config.IncludeRequestID)
		assert.False(t, config.IncludeTimestamp)
	})
}

// TestEnvelopeIntegration tests envelope middleware integration scenarios
func TestEnvelopeIntegration(t *testing.T) {
	t.Run("multiple envelope options work together", func(t *testing.T) {
		middleware := NewResponseEnvelopeMiddleware[EnvelopeTestResponseData](
			WithRequestID(true),
			WithTimestamp(false),
			WithMeta(true),
		)
		
		ctx := context.WithValue(context.Background(), "request_id", "integration-test-123")
		originalResponse := &EnvelopeTestResponseData{ID: "123", Message: "integration test"}
		
		envelope, err := middleware.After(ctx, originalResponse)
		require.NoError(t, err)
		require.NotNil(t, envelope)
		require.NotNil(t, envelope.Meta)
		
		assert.True(t, envelope.Success)
		assert.Equal(t, originalResponse, envelope.Data)
		assert.Equal(t, "integration-test-123", envelope.Meta.RequestID)
		assert.Empty(t, envelope.Meta.Timestamp) // Disabled by option
	})

	t.Run("context without request_id string type", func(t *testing.T) {
		middleware := NewResponseEnvelopeMiddleware[EnvelopeTestResponseData]()
		
		// Test with non-string request ID
		ctx := context.WithValue(context.Background(), "request_id", 12345)
		originalResponse := &EnvelopeTestResponseData{ID: "123", Message: "test"}
		
		envelope, err := middleware.After(ctx, originalResponse)
		require.NoError(t, err)
		require.NotNil(t, envelope)
		require.NotNil(t, envelope.Meta)
		
		assert.Empty(t, envelope.Meta.RequestID) // Should be empty for non-string type
		assert.NotEmpty(t, envelope.Meta.Timestamp)
	})

	t.Run("empty request_id string", func(t *testing.T) {
		middleware := NewResponseEnvelopeMiddleware[EnvelopeTestResponseData]()
		
		ctx := context.WithValue(context.Background(), "request_id", "")
		originalResponse := &EnvelopeTestResponseData{ID: "123", Message: "test"}
		
		envelope, err := middleware.After(ctx, originalResponse)
		require.NoError(t, err)
		require.NotNil(t, envelope)
		require.NotNil(t, envelope.Meta)
		
		assert.Empty(t, envelope.Meta.RequestID) // Should be empty for empty string
		assert.NotEmpty(t, envelope.Meta.Timestamp)
	})
}