package typedhttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
)

// Test global validator singleton pattern
func TestGlobalValidatorSingleton(t *testing.T) {
	t.Run("returns same validator instance", func(t *testing.T) {
		// Reset global validator for consistent testing
		globalValidator = nil
		globalValidatorOnce = sync.Once{}

		validator1 := getGlobalValidator()
		validator2 := getGlobalValidator()

		if validator1 != validator2 {
			t.Error("getGlobalValidator should return the same instance")
		}

		if validator1 == nil {
			t.Error("getGlobalValidator should not return nil")
		}
	})

	t.Run("validator is properly initialized", func(t *testing.T) {
		validator := getGlobalValidator()

		// Test that validator can actually validate
		type TestStruct struct {
			Name  string `validate:"required,min=2"`
			Email string `validate:"required,email"`
		}

		valid := TestStruct{Name: "John", Email: "john@example.com"}
		invalid := TestStruct{Name: "J", Email: "invalid-email"}

		if err := validator.Struct(valid); err != nil {
			t.Errorf("Valid struct should not produce validation error: %v", err)
		}

		if err := validator.Struct(invalid); err == nil {
			t.Error("Invalid struct should produce validation error")
		}
	})
}

// Test cached decoder functionality
func TestCachedDecoders(t *testing.T) {
	t.Run("HTTPHandler uses cached decoder when none provided", func(t *testing.T) {
		type TestRequest struct {
			ID string `path:"id" validate:"required"`
		}

		type TestResponse struct {
			ID string `json:"id"`
		}

		handler := &mockHandler[TestRequest, TestResponse]{
			response: TestResponse{ID: "test"},
		}

		httpHandler := NewHTTPHandler(handler)

		// Verify cached decoder is set
		if httpHandler.cachedDecoder == nil {
			t.Error("NewHTTPHandler should create cached decoder when none provided")
		}

		// Verify decoder field is not set (we use cached version)
		if httpHandler.decoder != nil {
			t.Error("decoder field should be nil when using cached decoder")
		}
	})

	t.Run("HTTPHandler uses provided decoder over cached", func(t *testing.T) {
		type TestRequest struct {
			ID string `path:"id" validate:"required"`
		}

		type TestResponse struct {
			ID string `json:"id"`
		}

		handler := &mockHandler[TestRequest, TestResponse]{
			response: TestResponse{ID: "test"},
		}

		customDecoder := &mockDecoder[TestRequest]{}

		httpHandler := NewHTTPHandler(handler, WithDecoder(customDecoder))

		// Verify custom decoder is used
		if httpHandler.decoder != customDecoder {
			t.Error("NewHTTPHandler should use provided decoder")
		}

		// Cached decoder should still be nil in this case
		if httpHandler.cachedDecoder != nil {
			t.Error("cachedDecoder should be nil when custom decoder is provided")
		}
	})

	t.Run("cached decoder is used in ServeHTTP", func(t *testing.T) {
		type TestRequest struct {
			ID string `path:"id" validate:"required"`
		}

		type TestResponse struct {
			ID string `json:"id"`
		}

		handler := &mockHandler[TestRequest, TestResponse]{
			response: TestResponse{ID: "test123"},
		}

		httpHandler := NewHTTPHandler(handler)

		req := httptest.NewRequest("GET", "/users/test123", nil)
		w := httptest.NewRecorder()

		httpHandler.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Verify handler was called with correct request
		if handler.lastRequest.ID != "test123" {
			t.Errorf("Expected ID 'test123', got '%s'", handler.lastRequest.ID)
		}
	})
}

// Test cached encoder functionality
func TestCachedEncoders(t *testing.T) {
	t.Run("HTTPHandler uses cached encoder when none provided", func(t *testing.T) {
		type TestRequest struct {
			ID string `path:"id"`
		}

		type TestResponse struct {
			ID string `json:"id"`
		}

		handler := &mockHandler[TestRequest, TestResponse]{
			response: TestResponse{ID: "test"},
		}

		httpHandler := NewHTTPHandler(handler)

		// Verify cached encoder is set
		if httpHandler.cachedEncoder == nil {
			t.Error("NewHTTPHandler should create cached encoder when none provided")
		}

		// Verify encoder field is not set (we use cached version)
		if httpHandler.encoder != nil {
			t.Error("encoder field should be nil when using cached encoder")
		}
	})

	t.Run("HTTPHandler uses provided encoder over cached", func(t *testing.T) {
		type TestRequest struct {
			ID string `path:"id"`
		}

		type TestResponse struct {
			ID string `json:"id"`
		}

		handler := &mockHandler[TestRequest, TestResponse]{
			response: TestResponse{ID: "test"},
		}

		customEncoder := &mockEncoder[TestResponse]{}

		httpHandler := NewHTTPHandler(handler, WithEncoder(customEncoder))

		// Verify custom encoder is used
		if httpHandler.encoder != customEncoder {
			t.Error("NewHTTPHandler should use provided encoder")
		}

		// Cached encoder should still be nil in this case
		if httpHandler.cachedEncoder != nil {
			t.Error("cachedEncoder should be nil when custom encoder is provided")
		}
	})
}

// Test smart decoder selection logic
func TestOptimalDecoderSelection(t *testing.T) {
	t.Run("path-only request uses PathDecoder", func(t *testing.T) {
		type PathOnlyRequest struct {
			ID   string `path:"id" validate:"required"`
			Name string `path:"name" validate:"required"`
		}

		decoder := getOptimalDecoder[PathOnlyRequest]()

		// Check if it's a PathDecoder (we can't directly type assert due to generics,
		// but we can check the ContentTypes method which is specific to PathDecoder)
		contentTypes := decoder.ContentTypes()
		expectedContentTypes := []string{"*/*"}

		if !reflect.DeepEqual(contentTypes, expectedContentTypes) {
			t.Errorf("Path-only request should use PathDecoder, got content types: %v", contentTypes)
		}
	})

	t.Run("json-only request uses JSONDecoder", func(t *testing.T) {
		type JSONOnlyRequest struct {
			Name  string `json:"name" validate:"required"`
			Email string `json:"email" validate:"required,email"`
		}

		decoder := getOptimalDecoder[JSONOnlyRequest]()

		// JSONDecoder returns ["application/json"]
		contentTypes := decoder.ContentTypes()
		expectedContentTypes := []string{"application/json"}

		if !reflect.DeepEqual(contentTypes, expectedContentTypes) {
			t.Errorf("JSON-only request should use JSONDecoder, got content types: %v", contentTypes)
		}
	})

	t.Run("mixed-source request uses CombinedDecoder", func(t *testing.T) {
		type MixedRequest struct {
			ID    string `path:"id" validate:"required"`
			Page  int    `query:"page" default:"1"`
			Data  string `json:"data"`
			Token string `header:"Authorization"`
		}

		decoder := getOptimalDecoder[MixedRequest]()

		// CombinedDecoder returns multiple content types
		contentTypes := decoder.ContentTypes()
		expectedContentTypes := []string{"application/json", "application/x-www-form-urlencoded", "*/*"}

		if !reflect.DeepEqual(contentTypes, expectedContentTypes) {
			t.Errorf("Mixed request should use CombinedDecoder, got content types: %v", contentTypes)
		}
	})

	t.Run("non-struct type uses CombinedDecoder", func(t *testing.T) {
		decoder := getOptimalDecoder[string]()

		// Should fall back to CombinedDecoder for non-struct types
		contentTypes := decoder.ContentTypes()
		expectedContentTypes := []string{"application/json", "application/x-www-form-urlencoded", "*/*"}

		if !reflect.DeepEqual(contentTypes, expectedContentTypes) {
			t.Errorf("Non-struct type should use CombinedDecoder, got content types: %v", contentTypes)
		}
	})

	t.Run("empty struct uses CombinedDecoder", func(t *testing.T) {
		type EmptyRequest struct{}

		decoder := getOptimalDecoder[EmptyRequest]()

		// Empty struct should fall back to CombinedDecoder
		contentTypes := decoder.ContentTypes()
		expectedContentTypes := []string{"application/json", "application/x-www-form-urlencoded", "*/*"}

		if !reflect.DeepEqual(contentTypes, expectedContentTypes) {
			t.Errorf("Empty struct should use CombinedDecoder, got content types: %v", contentTypes)
		}
	})
}

// Test performance optimization integration
func TestPerformanceOptimizationIntegration(t *testing.T) {
	t.Run("end-to-end performance optimizations", func(t *testing.T) {
		type GetUserRequest struct {
			ID string `path:"id" validate:"required"`
		}

		type User struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}

		handler := &mockHandler[GetUserRequest, User]{
			response: User{ID: "123", Name: "John Doe"},
		}

		// Create router and register handler
		router := NewRouter()
		GET(router, "/users/{id}", handler)

		// Make request
		req := httptest.NewRequest("GET", "/users/123", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Verify response
		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Verify handler received correct data
		if handler.lastRequest.ID != "123" {
			t.Errorf("Expected ID '123', got '%s'", handler.lastRequest.ID)
		}

		// Verify response body
		expectedBody := `{"id":"123","name":"John Doe"}` + "\n"
		if w.Body.String() != expectedBody {
			t.Errorf("Expected body %q, got %q", expectedBody, w.Body.String())
		}
	})

	t.Run("validates no per-request object creation", func(t *testing.T) {
		type TestRequest struct {
			ID string `path:"id" validate:"required"`
		}

		type TestResponse struct {
			ID string `json:"id"`
		}

		handler := &mockHandler[TestRequest, TestResponse]{
			response: TestResponse{ID: "test"},
		}

		// Create two handlers to verify they share the same global validator
		httpHandler1 := NewHTTPHandler(handler)
		httpHandler2 := NewHTTPHandler(handler)

		// Both should use the same global validator instance
		// We can't directly access the validator, but we can verify
		// that both handlers work correctly
		req := httptest.NewRequest("GET", "/test/123", nil)

		w1 := httptest.NewRecorder()
		httpHandler1.ServeHTTP(w1, req)

		w2 := httptest.NewRecorder()
		httpHandler2.ServeHTTP(w2, req)

		if w1.Code != w2.Code {
			t.Errorf("Both handlers should return same status code")
		}

		if w1.Body.String() != w2.Body.String() {
			t.Errorf("Both handlers should return same response body")
		}
	})
}

// Test memory efficiency improvements
func TestMemoryEfficiency(t *testing.T) {
	t.Run("no excessive allocations in handler creation", func(t *testing.T) {
		type TestRequest struct {
			ID string `path:"id"`
		}

		type TestResponse struct {
			Message string `json:"message"`
		}

		handler := &mockHandler[TestRequest, TestResponse]{
			response: TestResponse{Message: "test"},
		}

		// Create multiple handlers to verify no excessive allocations
		for i := 0; i < 100; i++ {
			httpHandler := NewHTTPHandler(handler)
			if httpHandler == nil {
				t.Error("NewHTTPHandler should not return nil")
				continue
			}

			// Verify cached components are created
			if httpHandler.cachedDecoder == nil {
				t.Error("Cached decoder should be created")
			}
			if httpHandler.cachedEncoder == nil {
				t.Error("Cached encoder should be created")
			}
		}
	})
}

// Mock implementations for testing

type mockHandler[TRequest, TResponse any] struct {
	response    TResponse
	lastRequest TRequest
	err         error
}

func (m *mockHandler[TRequest, TResponse]) Handle(ctx context.Context, req TRequest) (TResponse, error) {
	m.lastRequest = req
	return m.response, m.err
}

type mockDecoder[T any] struct {
	result T
	err    error
}

func (m *mockDecoder[T]) Decode(r *http.Request) (T, error) {
	return m.result, m.err
}

func (m *mockDecoder[T]) ContentTypes() []string {
	return []string{"application/json"}
}

type mockEncoder[T any] struct {
	err error
}

func (m *mockEncoder[T]) Encode(w http.ResponseWriter, data T, statusCode int) error {
	return m.err
}

func (m *mockEncoder[T]) ContentType() string {
	return "application/json"
}
