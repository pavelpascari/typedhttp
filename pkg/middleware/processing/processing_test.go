package processing

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types for processing middleware testing
type ValidationRequest struct {
	Name  string `json:"name" validate:"required,min=2,max=50"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"min=18,max=120"`
}

type ValidationResponse struct {
	Message string `json:"message"`
	Valid   bool   `json:"valid"`
}

// TestValidationMiddleware_Configuration tests validation middleware configuration
func TestValidationMiddleware_Configuration(t *testing.T) {
	t.Run("default_configuration", func(t *testing.T) {
		middleware := NewValidationMiddleware()
		assert.NotNil(t, middleware)

		config := middleware.GetConfig()
		assert.True(t, config.ValidateJSON)
		assert.True(t, config.ValidateStruct)
		assert.False(t, config.StrictValidation)
	})

	t.Run("custom_configuration", func(t *testing.T) {
		middleware := NewValidationMiddleware(
			WithStrictValidation(true),
			WithCustomValidators(map[string]ValidationFunc{
				"custom": func(value interface{}) error {
					return nil
				},
			}),
			WithValidationErrorHandler(func(err error) error {
				return fmt.Errorf("custom: %w", err)
			}),
		)

		config := middleware.GetConfig()
		assert.True(t, config.StrictValidation)
		assert.Len(t, config.CustomValidators, 1)
		assert.NotNil(t, config.ErrorHandler)
	})
}

// TestValidationMiddleware_HTTPMiddleware tests validation as HTTP middleware
func TestValidationMiddleware_HTTPMiddleware(t *testing.T) {
	middleware := NewValidationMiddleware()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "validation passed"}`))
	})

	handler := middleware.HTTPMiddleware()(testHandler)

	t.Run("valid_json_request", func(t *testing.T) {
		validJSON := `{"name": "John Doe", "email": "john@example.com", "age": 25}`
		req := httptest.NewRequest(http.MethodPost, "/validate", strings.NewReader(validJSON))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "validation passed")
	})

	t.Run("invalid_json_request", func(t *testing.T) {
		invalidJSON := `{"name": "", "email": "invalid", "age": 10}`
		req := httptest.NewRequest(http.MethodPost, "/validate", strings.NewReader(invalidJSON))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "validation error")
	})

	t.Run("malformed_json", func(t *testing.T) {
		malformedJSON := `{"name": "John", "email": "john@example.com", "age":}`
		req := httptest.NewRequest(http.MethodPost, "/validate", strings.NewReader(malformedJSON))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "invalid JSON")
	})
}

// TestValidationMiddleware_TypedMiddleware tests validation as typed middleware
func TestValidationMiddleware_TypedMiddleware(t *testing.T) {
	middleware := NewValidationMiddleware()

	t.Run("valid_typed_request", func(t *testing.T) {
		req := &ValidationRequest{
			Name:  "Alice Johnson",
			Email: "alice@example.com",
			Age:   30,
		}

		ctx := context.Background()
		newCtx, err := middleware.Before(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, newCtx)
	})

	t.Run("invalid_typed_request", func(t *testing.T) {
		req := &ValidationRequest{
			Name:  "A", // Too short
			Email: "invalid-email",
			Age:   15, // Too young
		}

		ctx := context.Background()
		_, err := middleware.Before(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})

	t.Run("nil_request", func(t *testing.T) {
		ctx := context.Background()
		_, err := middleware.Before(ctx, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil request")
	})
}

// TestValidationMiddleware_CustomValidators tests custom validation functions
func TestValidationMiddleware_CustomValidators(t *testing.T) {
	customValidators := map[string]ValidationFunc{
		"strong_password": func(value interface{}) error {
			if str, ok := value.(string); ok {
				if len(str) < 8 {
					return fmt.Errorf("password must be at least 8 characters")
				}
			}
			return nil
		},
	}

	middleware := NewValidationMiddleware(
		WithCustomValidators(customValidators),
	)

	t.Run("custom_validation_pass", func(t *testing.T) {
		err := middleware.ValidateValue("strongPassword123", "strong_password")
		assert.NoError(t, err)
	})

	t.Run("custom_validation_fail", func(t *testing.T) {
		err := middleware.ValidateValue("weak", "strong_password")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least 8 characters")
	})
}

// TestCompressionMiddleware_Configuration tests compression middleware configuration
func TestCompressionMiddleware_Configuration(t *testing.T) {
	t.Run("default_configuration", func(t *testing.T) {
		middleware := NewCompressionMiddleware()
		assert.NotNil(t, middleware)

		config := middleware.GetConfig()
		assert.Equal(t, DefaultCompressionLevel, config.Level)
		assert.Contains(t, config.Types, "application/json")
		assert.Contains(t, config.Types, "text/html")
		assert.Equal(t, DefaultMinSize, config.MinSize)
	})

	t.Run("custom_configuration", func(t *testing.T) {
		middleware := NewCompressionMiddleware(
			WithCompressionLevel(BestCompression),
			WithCompressionTypes([]string{"application/json", "text/plain"}),
			WithMinCompressionSize(2048),
			WithCompressionHeaders(map[string]string{"X-Compression": "gzip"}),
		)

		config := middleware.GetConfig()
		assert.Equal(t, BestCompression, config.Level)
		assert.Equal(t, []string{"application/json", "text/plain"}, config.Types)
		assert.Equal(t, 2048, config.MinSize)
		assert.Equal(t, "gzip", config.Headers["X-Compression"])
	})
}

// TestCompressionMiddleware_HTTPMiddleware tests compression as HTTP middleware
func TestCompressionMiddleware_HTTPMiddleware(t *testing.T) {
	middleware := NewCompressionMiddleware()

	// Create test data that's large enough to trigger compression
	largeData := strings.Repeat("This is test data for compression. ", 100)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(`{"data": "%s"}`, largeData)))
	})

	handler := middleware.HTTPMiddleware()(testHandler)

	t.Run("gzip_compression_supported", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/data", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "gzip", rr.Header().Get("Content-Encoding"))

		// Verify content can be decompressed
		gzReader, err := gzip.NewReader(rr.Body)
		require.NoError(t, err)
		defer gzReader.Close()

		decompressed, err := io.ReadAll(gzReader)
		require.NoError(t, err)
		assert.Contains(t, string(decompressed), largeData)
	})

	t.Run("no_compression_without_accept_encoding", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/data", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Empty(t, rr.Header().Get("Content-Encoding"))
		assert.Contains(t, rr.Body.String(), largeData)
	})

	t.Run("small_content_not_compressed", func(t *testing.T) {
		smallHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"small": "data"}`))
		})

		handler := middleware.HTTPMiddleware()(smallHandler)

		req := httptest.NewRequest(http.MethodGet, "/small", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Empty(t, rr.Header().Get("Content-Encoding"))
	})
}

// TestCompressionMiddleware_TypedMiddleware tests compression as typed middleware
func TestCompressionMiddleware_TypedMiddleware(t *testing.T) {
	middleware := NewCompressionMiddleware()

	t.Run("typed_compression", func(t *testing.T) {
		req := &ValidationRequest{
			Name:  "Compression Test",
			Email: "test@compression.com",
			Age:   25,
		}

		// Large response to trigger compression
		resp := &ValidationResponse{
			Message: strings.Repeat("Large response data for compression testing. ", 100),
			Valid:   true,
		}

		ctx := context.WithValue(context.Background(), "accept_encoding", "gzip")

		newCtx, err := middleware.Before(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, newCtx)

		finalResp, err := middleware.After(newCtx, req, resp, nil)
		require.NoError(t, err)
		assert.NotNil(t, finalResp)
	})
}

// TestCORSMiddleware_Configuration tests CORS middleware configuration
func TestCORSMiddleware_Configuration(t *testing.T) {
	t.Run("default_configuration", func(t *testing.T) {
		middleware := NewCORSMiddleware()
		assert.NotNil(t, middleware)

		config := middleware.GetConfig()
		assert.Contains(t, config.AllowedOrigins, "*")
		assert.Contains(t, config.AllowedMethods, "GET")
		assert.Contains(t, config.AllowedMethods, "POST")
		assert.Contains(t, config.AllowedHeaders, "Content-Type")
		assert.Equal(t, 86400, config.MaxAge) // 24 hours
	})

	t.Run("custom_configuration", func(t *testing.T) {
		middleware := NewCORSMiddleware(
			WithAllowedOrigins([]string{"https://example.com", "https://api.example.com"}),
			WithAllowedMethods([]string{"GET", "POST", "PUT", "DELETE"}),
			WithAllowedHeaders([]string{"Content-Type", "Authorization", "X-API-Key"}),
			WithExposedHeaders([]string{"X-Total-Count", "X-Page-Count"}),
			WithAllowCredentials(true),
			WithMaxAge(3600),
		)

		config := middleware.GetConfig()
		assert.Equal(t, []string{"https://example.com", "https://api.example.com"}, config.AllowedOrigins)
		assert.Contains(t, config.AllowedMethods, "DELETE")
		assert.Contains(t, config.AllowedHeaders, "X-API-Key")
		assert.Contains(t, config.ExposedHeaders, "X-Total-Count")
		assert.True(t, config.AllowCredentials)
		assert.Equal(t, 3600, config.MaxAge)
	})
}

// TestCORSMiddleware_HTTPMiddleware tests CORS as HTTP middleware
func TestCORSMiddleware_HTTPMiddleware(t *testing.T) {
	middleware := NewCORSMiddleware(
		WithAllowedOrigins([]string{"https://example.com"}),
		WithAllowedMethods([]string{"GET", "POST", "OPTIONS"}),
		WithAllowedHeaders([]string{"Content-Type", "Authorization"}),
		WithAllowCredentials(true),
	)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "CORS test"}`))
	})

	handler := middleware.HTTPMiddleware()(testHandler)

	t.Run("simple_cors_request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "https://example.com", rr.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "true", rr.Header().Get("Access-Control-Allow-Credentials"))
	})

	t.Run("preflight_cors_request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "Content-Type,Authorization")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNoContent, rr.Code)
		assert.Equal(t, "https://example.com", rr.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, rr.Header().Get("Access-Control-Allow-Methods"), "POST")
		assert.Contains(t, rr.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
		assert.Equal(t, "true", rr.Header().Get("Access-Control-Allow-Credentials"))
	})

	t.Run("cors_request_disallowed_origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "https://malicious.com")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusForbidden, rr.Code)
		assert.Empty(t, rr.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("cors_request_disallowed_method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", "DELETE")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
	})
}

// TestCORSMiddleware_TypedMiddleware tests CORS as typed middleware
func TestCORSMiddleware_TypedMiddleware(t *testing.T) {
	middleware := NewCORSMiddleware(
		WithAllowedOrigins([]string{"https://api.example.com"}),
	)

	t.Run("typed_cors_validation", func(t *testing.T) {
		req := &ValidationRequest{
			Name:  "CORS Test",
			Email: "cors@example.com",
			Age:   28,
		}

		// Simulate CORS context with origin
		ctx := context.WithValue(context.Background(), "origin", "https://api.example.com")

		newCtx, err := middleware.Before(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, newCtx)

		resp := &ValidationResponse{Message: "CORS passed", Valid: true}
		finalResp, err := middleware.After(newCtx, req, resp, nil)
		require.NoError(t, err)
		assert.Equal(t, resp, finalResp)
	})

	t.Run("typed_cors_origin_not_allowed", func(t *testing.T) {
		req := &ValidationRequest{
			Name:  "CORS Test",
			Email: "cors@example.com",
			Age:   28,
		}

		// Simulate CORS context with disallowed origin
		ctx := context.WithValue(context.Background(), "origin", "https://malicious.com")

		_, err := middleware.Before(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "origin not allowed")
	})
}

// TestProcessingMiddleware_CombinedUsage tests using all processing middleware together
func TestProcessingMiddleware_CombinedUsage(t *testing.T) {
	// Create all processing middleware
	validationMW := NewValidationMiddleware()
	compressionMW := NewCompressionMiddleware()
	corsMW := NewCORSMiddleware(
		WithAllowedOrigins([]string{"https://example.com"}),
	)

	// Combine middleware
	combined := NewProcessingMiddleware(validationMW, compressionMW, corsMW)

	// Large valid JSON for compression
	largeData := strings.Repeat("test data ", 200)
	validJSON := fmt.Sprintf(`{"name": "Combined Test", "email": "test@example.com", "age": 25, "data": "%s"}`, largeData)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(`{"message": "all processing passed", "data": "%s"}`, largeData)))
	})

	handler := combined.HTTPMiddleware()(testHandler)

	t.Run("combined_processing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/combined", strings.NewReader(validJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Accept-Encoding", "gzip")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		// Verify CORS headers
		assert.Equal(t, "https://example.com", rr.Header().Get("Access-Control-Allow-Origin"))

		// Verify compression
		assert.Equal(t, "gzip", rr.Header().Get("Content-Encoding"))

		// Verify content can be decompressed
		gzReader, err := gzip.NewReader(rr.Body)
		require.NoError(t, err)
		defer gzReader.Close()

		decompressed, err := io.ReadAll(gzReader)
		require.NoError(t, err)
		assert.Contains(t, string(decompressed), "all processing passed")
	})
}

// TestProcessingMiddleware_Performance tests performance impact of processing middleware
func TestProcessingMiddleware_Performance(t *testing.T) {
	validationMW := NewValidationMiddleware()
	compressionMW := NewCompressionMiddleware()
	corsMW := NewCORSMiddleware()

	combined := NewProcessingMiddleware(validationMW, compressionMW, corsMW)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"performance": "test"}`))
	})

	handler := combined.HTTPMiddleware()(testHandler)

	// Benchmark the middleware
	iterations := 100
	validJSON := `{"name": "Performance Test", "email": "perf@example.com", "age": 30}`

	for i := 0; i < iterations; i++ {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/perf/%d", i), strings.NewReader(validJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "https://example.com")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}

	t.Logf("Performance test: %d requests completed successfully", iterations)
}

// TestProcessingMiddleware_ErrorHandling tests error handling across processing middleware
func TestProcessingMiddleware_ErrorHandling(t *testing.T) {
	validationMW := NewValidationMiddleware(WithStrictValidation(true))
	compressionMW := NewCompressionMiddleware()
	corsMW := NewCORSMiddleware(
		WithAllowedOrigins([]string{"https://allowed.com"}),
	)

	combined := NewProcessingMiddleware(validationMW, compressionMW, corsMW)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	})

	handler := combined.HTTPMiddleware()(testHandler)

	t.Run("validation_error_blocks_processing", func(t *testing.T) {
		invalidJSON := `{"name": "", "email": "invalid", "age": 10}`
		req := httptest.NewRequest(http.MethodPost, "/error", strings.NewReader(invalidJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "https://allowed.com")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "validation")
	})

	t.Run("cors_error_blocks_processing", func(t *testing.T) {
		validJSON := `{"name": "Valid User", "email": "valid@example.com", "age": 25}`
		req := httptest.NewRequest(http.MethodPost, "/error", strings.NewReader(validJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "https://malicious.com")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusForbidden, rr.Code)
	})
}
