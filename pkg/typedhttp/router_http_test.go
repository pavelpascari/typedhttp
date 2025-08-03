package typedhttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHTTPHandlerServeHTTP tests the main ServeHTTP method which currently has 0% coverage.
func TestHTTPHandlerServeHTTP(t *testing.T) {
	type TestResponse struct {
		Message string `json:"message"`
		Status  string `json:"status"`
	}

	// Create a typed handler that works with our specific types
	handler := &TypedTestHandler{
		response: TestResponse{Message: "Hello", Status: "success"},
	}

	// Create HTTP handler with default decoders/encoders
	httpHandler := NewHTTPHandler(handler)

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		contentType    string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "successful POST request",
			method:         http.MethodPost,
			path:           "/test",
			body:           `{"name":"John","age":30}`,
			contentType:    "application/json",
			expectedStatus: http.StatusCreated, // POST should return 201
			expectedBody:   `{"message":"Hello","status":"success"}`,
		},
		{
			name:           "successful GET request",
			method:         http.MethodGet,
			path:           "/test",
			body:           `{"name":"John","age":30}`,
			contentType:    "application/json",
			expectedStatus: http.StatusOK, // GET should return 200
			expectedBody:   `{"message":"Hello","status":"success"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			} else {
				req = httptest.NewRequest(tt.method, tt.path, http.NoBody)
			}

			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			w := httptest.NewRecorder()

			// This tests the ServeHTTP method which had 0% coverage
			httpHandler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != "" {
				assert.JSONEq(t, tt.expectedBody, w.Body.String())
			}
		})
	}
}

// TestHTTPHandlerWithCustomDecoderEncoder tests ServeHTTP with custom decoder/encoder.
func TestHTTPHandlerWithCustomDecoderEncoder(t *testing.T) {
	// Create a typed handler for query parameters
	handler := &QueryTestHandler{
		result: "custom",
	}

	// Create with custom decoder and encoder
	httpHandler := NewHTTPHandler(
		handler,
		WithDecoder[struct {
			ID string `query:"id"`
		}](NewQueryDecoder[struct {
			ID string `query:"id"`
		}](nil)),
		WithEncoder[struct {
			Result string `json:"result"`
		}](NewJSONEncoder[struct {
			Result string `json:"result"`
		}]()),
	)

	req := httptest.NewRequest(http.MethodGet, "/test?id=123", http.NoBody)
	w := httptest.NewRecorder()

	httpHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"result":"custom"}`, w.Body.String())
}

// TestHTTPHandlerWithJSONDecoder tests ServeHTTP with JSON-only decoder to test JSON parsing errors.
func TestHTTPHandlerWithJSONDecoder(t *testing.T) {
	// Create a handler that expects JSON input
	handler := &JSONTestHandler{
		response: struct {
			Message string `json:"message"`
			Status  string `json:"status"`
		}{Message: "Hello", Status: "success"},
	}

	// Create with JSON-only decoder to test JSON parsing errors
	httpHandler := NewHTTPHandler(
		handler,
		WithDecoder[struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}](NewJSONDecoder[struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}](nil)),
		WithEncoder[struct {
			Message string `json:"message"`
			Status  string `json:"status"`
		}](NewJSONEncoder[struct {
			Message string `json:"message"`
			Status  string `json:"status"`
		}]()),
	)

	// Test invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{invalid json}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	httpHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	if w.Body.Len() > 0 {
		var errorResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &errorResp)
		require.NoError(t, err)
		assert.Contains(t, errorResp["error"], "Invalid JSON")
	}
}

// TestHTTPHandlerErrorHandling tests the handleError method which had 0% coverage.
func TestHTTPHandlerErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		handlerError   error
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "validation error",
			handlerError:   NewValidationError("validation failed", map[string]string{"name": "required"}),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Validation failed",
		},
		{
			name:           "not found error",
			handlerError:   NewNotFoundError("user", "123"),
			expectedStatus: http.StatusNotFound,
			expectedError:  "user with id '123' not found",
		},
		{
			name:           "conflict error",
			handlerError:   NewConflictError("resource already exists"),
			expectedStatus: http.StatusConflict,
			expectedError:  "resource already exists",
		},
		{
			name:           "unauthorized error",
			handlerError:   NewUnauthorizedError("invalid credentials"),
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid credentials",
		},
		{
			name:           "forbidden error",
			handlerError:   NewForbiddenError("access denied"),
			expectedStatus: http.StatusForbidden,
			expectedError:  "access denied",
		},
		{
			name:           "generic error",
			handlerError:   assert.AnError,
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a typed error handler that returns errors
			errorHandler := &TypedErrorTestHandler{err: tt.handlerError}
			httpHandler := NewHTTPHandler(errorHandler)

			req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
			w := httptest.NewRecorder()

			// This will trigger the handleError method
			httpHandler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var errorResp map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &errorResp)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedError, errorResp["error"])
		})
	}
}

// TestHTTPHandlerWithCustomErrorMapper tests error handling with custom error mapper.
func TestHTTPHandlerWithCustomErrorMapper(t *testing.T) {
	// Custom error mapper that returns 418 for all errors
	customMapper := &CustomErrorMapper{}

	errorHandler := &TypedErrorTestHandler{err: assert.AnError}
	httpHandler := NewHTTPHandler(
		errorHandler,
		WithErrorMapper(customMapper),
	)

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	w := httptest.NewRecorder()

	httpHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTeapot, w.Code) // 418
	assert.JSONEq(t, `{"custom":"error"}`, w.Body.String())
}

// TestTypedRouterServeHTTP tests the TypedRouter ServeHTTP method which had 0% coverage.
func TestTypedRouterServeHTTP(t *testing.T) {
	router := NewRouter()
	handler := &TestBusinessHandlerWithGreeting{}

	// Register a handler
	POST(router, "/greet", handler)

	req := httptest.NewRequest(http.MethodPost, "/greet", strings.NewReader(`{"name":"Alice"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// This tests TypedRouter.ServeHTTP which had 0% coverage
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.JSONEq(t, `{"greeting":"Hello, Alice!"}`, w.Body.String())
}

// Test helper handlers

type TestBusinessHandler struct {
	response interface{}
	err      error
}

func (h *TestBusinessHandler) Handle(ctx context.Context, req interface{}) (interface{}, error) {
	if h.err != nil {
		return nil, h.err
	}

	return h.response, nil
}

type ErrorTestHandler struct {
	err error
}

func (h *ErrorTestHandler) Handle(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, h.err
}

type CustomErrorMapper struct{}

func (m *CustomErrorMapper) MapError(err error) (statusCode int, response interface{}) {
	return http.StatusTeapot, map[string]string{"custom": "error"}
}

type TestBusinessHandlerWithGreeting struct{}

func (h *TestBusinessHandlerWithGreeting) Handle(ctx context.Context, req struct {
	Name string `json:"name"`
}) (struct {
	Greeting string `json:"greeting"`
},
	error,
) {
	return struct {
		Greeting string `json:"greeting"`
	}{
		Greeting: "Hello, " + req.Name + "!",
	}, nil
}

// TypedTestHandler is a handler that works with specific request/response types.
type TypedTestHandler struct {
	response interface{}
	err      error
}

func (h *TypedTestHandler) Handle(ctx context.Context, req struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}) (struct {
	Message string `json:"message"`
	Status  string `json:"status"`
},
	error,
) {
	if h.err != nil {
		return struct {
			Message string `json:"message"`
			Status  string `json:"status"`
		}{}, h.err
	}

	if resp, ok := h.response.(struct {
		Message string `json:"message"`
		Status  string `json:"status"`
	}); ok {
		return resp, nil
	}

	// Fallback response
	return struct {
		Message string `json:"message"`
		Status  string `json:"status"`
	}{Message: "Hello", Status: "success"}, nil
}

// TypedErrorTestHandler is a handler that returns errors without requiring complex decoding.
type TypedErrorTestHandler struct {
	err error
}

func (h *TypedErrorTestHandler) Handle(ctx context.Context, req struct{}) (struct{}, error) {
	return struct{}{}, h.err
}

// QueryTestHandler handles query parameter requests.
type QueryTestHandler struct {
	result string
}

func (h *QueryTestHandler) Handle(ctx context.Context, req struct {
	ID string `query:"id"`
}) (struct {
	Result string `json:"result"`
},
	error,
) {
	return struct {
		Result string `json:"result"`
	}{Result: h.result}, nil
}

// JSONTestHandler handles JSON requests.
type JSONTestHandler struct {
	response interface{}
}

func (h *JSONTestHandler) Handle(ctx context.Context, req struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}) (struct {
	Message string `json:"message"`
	Status  string `json:"status"`
},
	error,
) {
	if resp, ok := h.response.(struct {
		Message string `json:"message"`
		Status  string `json:"status"`
	}); ok {
		return resp, nil
	}

	// Fallback response
	return struct {
		Message string `json:"message"`
		Status  string `json:"status"`
	}{Message: "Hello", Status: "success"}, nil
}
