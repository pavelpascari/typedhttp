package openapi

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types
type TestUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type TestResponse struct {
	User    TestUser `json:"user"`
	Message string   `json:"message"`
}

func TestMiddlewareSchemaTransformation(t *testing.T) {
	tests := []struct {
		name                   string
		middleware             []typedhttp.MiddlewareEntry
		expectedResponseSchema map[string]interface{}
		expectedRequired       []string
	}{
		{
			name:       "no_middleware",
			middleware: []typedhttp.MiddlewareEntry{},
			expectedResponseSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"id":   map[string]interface{}{"type": "string"},
							"name": map[string]interface{}{"type": "string"},
						},
						"required": []interface{}{"id", "name"},
					},
					"message": map[string]interface{}{"type": "string"},
				},
				"required": []interface{}{"user", "message"},
			},
		},
		{
			name: "envelope_middleware",
			middleware: []typedhttp.MiddlewareEntry{
				{
					Middleware: typedhttp.NewResponseEnvelopeMiddleware[TestResponse](
						typedhttp.WithRequestID(true),
						typedhttp.WithTimestamp(true),
					),
					Config: typedhttp.MiddlewareConfig{
						Name: "envelope",
					},
				},
			},
			expectedResponseSchema: map[string]interface{}{
				"type":        "object",
				"description": "Standard response envelope wrapping the actual response data",
				"properties": map[string]interface{}{
					"data": map[string]interface{}{
						"description": "The actual response data",
						"nullable":    true,
					},
					"error": map[string]interface{}{
						"type":        "string",
						"description": "Error message when success is false",
						"nullable":    true,
					},
					"success": map[string]interface{}{
						"type":        "boolean",
						"description": "Indicates whether the request was successful",
					},
					"meta": map[string]interface{}{
						"type":        "object",
						"description": "Additional metadata about the response",
						"nullable":    true,
						"properties": map[string]interface{}{
							"request_id": map[string]interface{}{
								"type":        "string",
								"description": "Unique identifier for the request",
							},
							"timestamp": map[string]interface{}{
								"type":        "string",
								"format":      "date-time",
								"description": "Timestamp when the response was generated",
							},
						},
					},
				},
				"required": []interface{}{"success"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create generator
			generator := NewGenerator(&Config{
				Info: Info{
					Title:   "Test API",
					Version: "1.0.0",
				},
			})

			// Create base schema for TestResponse
			baseSchema, err := generator.createSchemaFromType(reflect.TypeOf(TestResponse{}))
			require.NoError(t, err)

			// Apply middleware transformations
			finalSchema, err := generator.applyMiddlewareSchemaTransformations(
				context.Background(),
				tt.middleware,
				baseSchema,
			)
			require.NoError(t, err)

			// Convert to JSON for easier comparison
			finalSchemaJSON, err := json.Marshal(finalSchema.Value)
			require.NoError(t, err)

			var actualSchema map[string]interface{}
			err = json.Unmarshal(finalSchemaJSON, &actualSchema)
			require.NoError(t, err)

			// For envelope middleware, we need to check the nested structure
			if len(tt.middleware) > 0 {
				// Verify envelope structure
				assert.Equal(t, "object", actualSchema["type"])
				assert.Equal(t, "Standard response envelope wrapping the actual response data", actualSchema["description"])

				properties, ok := actualSchema["properties"].(map[string]interface{})
				require.True(t, ok)

				// Check required fields
				required, ok := actualSchema["required"].([]interface{})
				require.True(t, ok)
				assert.Contains(t, required, "success")

				// Check data field contains original schema
				dataField, ok := properties["data"].(map[string]interface{})
				require.True(t, ok)
				nullable, ok := dataField["nullable"].(bool)
				require.True(t, ok)
				assert.True(t, nullable)
				assert.Equal(t, "The actual response data", dataField["description"])

				// Check error field
				errorField, ok := properties["error"].(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, "string", errorField["type"])
				errorNullable, ok := errorField["nullable"].(bool)
				require.True(t, ok)
				assert.True(t, errorNullable)

				// Check success field
				successField, ok := properties["success"].(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, "boolean", successField["type"])

				// Check meta field
				metaField, ok := properties["meta"].(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, "object", metaField["type"])
				metaNullable, ok := metaField["nullable"].(bool)
				require.True(t, ok)
				assert.True(t, metaNullable)

				metaProperties, ok := metaField["properties"].(map[string]interface{})
				require.True(t, ok)

				// Check request_id field
				requestIDField, ok := metaProperties["request_id"].(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, "string", requestIDField["type"])

				// Check timestamp field
				timestampField, ok := metaProperties["timestamp"].(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, "string", timestampField["type"])
				assert.Equal(t, "date-time", timestampField["format"])
			} else {
				// No middleware - should match original schema structure
				assert.Equal(t, "object", actualSchema["type"])

				properties, ok := actualSchema["properties"].(map[string]interface{})
				require.True(t, ok)

				// Check user field
				userField, ok := properties["user"].(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, "object", userField["type"])

				// Check message field
				messageField, ok := properties["message"].(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, "string", messageField["type"])
			}
		})
	}
}

func TestEnvelopeMiddlewareDetection(t *testing.T) {
	generator := NewGenerator(&Config{
		Info: Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
	})

	tests := []struct {
		name                string
		middleware          []typedhttp.MiddlewareEntry
		expectedHasEnvelope bool
	}{
		{
			name:                "no_middleware",
			middleware:          []typedhttp.MiddlewareEntry{},
			expectedHasEnvelope: false,
		},
		{
			name: "envelope_middleware",
			middleware: []typedhttp.MiddlewareEntry{
				{
					Middleware: typedhttp.NewResponseEnvelopeMiddleware[TestResponse](),
					Config:     typedhttp.MiddlewareConfig{Name: "envelope"},
				},
			},
			expectedHasEnvelope: true,
		},
		{
			name: "other_middleware",
			middleware: []typedhttp.MiddlewareEntry{
				{
					Middleware: &mockMiddleware{},
					Config:     typedhttp.MiddlewareConfig{Name: "mock"},
				},
			},
			expectedHasEnvelope: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasEnvelope := generator.hasEnvelopeMiddleware(tt.middleware)
			assert.Equal(t, tt.expectedHasEnvelope, hasEnvelope)
		})
	}
}

// Mock middleware for testing
type mockMiddleware struct{}

func (m *mockMiddleware) After(ctx context.Context, resp *TestResponse) (*TestResponse, error) {
	return resp, nil
}

func TestErrorResponseGeneration(t *testing.T) {
	generator := NewGenerator(&Config{
		Info: Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
	})

	operation := &openapi3.Operation{
		Responses: &openapi3.Responses{},
	}

	// Add envelope error responses
	generator.addEnvelopeErrorResponses(operation)

	// Check that error responses were added
	errorCodes := []string{"400", "401", "404", "500"}
	for _, code := range errorCodes {
		response := operation.Responses.Value(code)
		require.NotNil(t, response, "Expected response for status code %s", code)

		// Check response structure
		content := response.Value.Content["application/json"]
		require.NotNil(t, content, "Expected JSON content for status code %s", code)

		schema := content.Schema.Value
		require.NotNil(t, schema, "Expected schema for status code %s", code)

		// Verify envelope structure
		assert.Equal(t, &openapi3.Types{"object"}, schema.Type)
		assert.Contains(t, schema.Required, "success")
		assert.Contains(t, schema.Required, "error")

		// Check properties
		successProp := schema.Properties["success"]
		require.NotNil(t, successProp)
		assert.Equal(t, &openapi3.Types{"boolean"}, successProp.Value.Type)
		assert.Contains(t, successProp.Value.Enum, false)

		errorProp := schema.Properties["error"]
		require.NotNil(t, errorProp)
		assert.Equal(t, &openapi3.Types{"string"}, errorProp.Value.Type)

		dataProp := schema.Properties["data"]
		require.NotNil(t, dataProp)
		assert.Equal(t, &openapi3.Types{"null"}, dataProp.Value.Type)
		assert.True(t, dataProp.Value.Nullable)
	}
}
