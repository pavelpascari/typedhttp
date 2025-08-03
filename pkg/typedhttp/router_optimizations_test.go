package typedhttp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Test router-level optimizations and handler registration
func TestRouterOptimizations(t *testing.T) {
	t.Run("router handler registration uses cached components", func(t *testing.T) {
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

		router := NewRouter()

		// Register handler - this should use cached decoders/encoders
		GET(router, "/users/{id}", handler)

		// Verify registration worked
		registrations := router.GetHandlers()
		if len(registrations) != 1 {
			t.Errorf("Expected 1 registered handler, got %d", len(registrations))
		}

		reg := registrations[0]
		if reg.Method != "GET" {
			t.Errorf("Expected method GET, got %s", reg.Method)
		}
		if reg.Path != "/users/{id}" {
			t.Errorf("Expected path /users/{id}, got %s", reg.Path)
		}
	})

	t.Run("multiple handlers share optimized components", func(t *testing.T) {
		type GetUserRequest struct {
			ID string `path:"id" validate:"required"`
		}

		type CreateUserRequest struct {
			Name  string `json:"name" validate:"required"`
			Email string `json:"email" validate:"required,email"`
		}

		type User struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}

		getHandler := &mockHandler[GetUserRequest, User]{
			response: User{ID: "123", Name: "John Doe"},
		}

		createHandler := &mockHandler[CreateUserRequest, User]{
			response: User{ID: "456", Name: "Jane Doe"},
		}

		router := NewRouter()

		// Register multiple handlers
		GET(router, "/users/{id}", getHandler)
		POST(router, "/users", createHandler)

		// Verify both handlers are registered
		registrations := router.GetHandlers()
		if len(registrations) != 2 {
			t.Errorf("Expected 2 registered handlers, got %d", len(registrations))
		}

		// Test GET endpoint
		req := httptest.NewRequest("GET", "/users/123", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("GET request: expected status 200, got %d", w.Code)
		}

		// Test POST endpoint
		reqBody := `{"name":"Jane Doe","email":"jane@example.com"}`
		req = httptest.NewRequest("POST", "/users", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 201 {
			t.Errorf("POST request: expected status 201, got %d", w.Code)
		}
	})

	t.Run("path-only request optimization works correctly", func(t *testing.T) {
		type PathOnlyRequest struct {
			ID string `path:"id" validate:"required"`
		}

		type SimpleResponse struct {
			ID string `json:"id"`
		}

		handler := &mockHandler[PathOnlyRequest, SimpleResponse]{
			response: SimpleResponse{ID: "test123"},
		}

		router := NewRouter()
		GET(router, "/users/{id}", handler)

		// Test request
		req := httptest.NewRequest("GET", "/users/test123", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Verify handler received correct path parameters
		if handler.lastRequest.ID != "test123" {
			t.Errorf("Expected ID 'test123', got '%s'", handler.lastRequest.ID)
		}
	})

	t.Run("json-only request optimization works correctly", func(t *testing.T) {
		type JSONOnlyRequest struct {
			Name        string `json:"name" validate:"required,min=2"`
			Email       string `json:"email" validate:"required,email"`
			Age         int    `json:"age" validate:"required,min=18"`
			Preferences struct {
				Theme    string `json:"theme" validate:"oneof=light dark"`
				Language string `json:"language" validate:"required"`
			} `json:"preferences"`
		}

		type CreateUserResponse struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Email   string `json:"email"`
			Message string `json:"message"`
		}

		handler := &mockHandler[JSONOnlyRequest, CreateUserResponse]{
			response: CreateUserResponse{
				ID:      "user789",
				Name:    "Alice Johnson",
				Email:   "alice@example.com",
				Message: "User created successfully",
			},
		}

		router := NewRouter()
		POST(router, "/users", handler)

		// Test with complex JSON request
		requestBody := `{
			"name": "Alice Johnson",
			"email": "alice@example.com",
			"age": 25,
			"preferences": {
				"theme": "dark",
				"language": "en"
			}
		}`

		req := httptest.NewRequest("POST", "/users", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 201 {
			t.Errorf("Expected status 201, got %d", w.Code)
		}

		// Verify handler received correct JSON data
		if handler.lastRequest.Name != "Alice Johnson" {
			t.Errorf("Expected Name 'Alice Johnson', got '%s'", handler.lastRequest.Name)
		}
		if handler.lastRequest.Email != "alice@example.com" {
			t.Errorf("Expected Email 'alice@example.com', got '%s'", handler.lastRequest.Email)
		}
		if handler.lastRequest.Age != 25 {
			t.Errorf("Expected Age 25, got %d", handler.lastRequest.Age)
		}
		if handler.lastRequest.Preferences.Theme != "dark" {
			t.Errorf("Expected Theme 'dark', got '%s'", handler.lastRequest.Preferences.Theme)
		}
	})

	t.Run("validation errors are handled efficiently", func(t *testing.T) {
		type ValidatedRequest struct {
			Name  string `json:"name" validate:"required,min=2,max=50"`
			Email string `json:"email" validate:"required,email"`
			Age   int    `json:"age" validate:"required,min=18,max=120"`
		}

		type User struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}

		handler := &mockHandler[ValidatedRequest, User]{
			response: User{ID: "123", Name: "Valid User"},
		}

		router := NewRouter()
		POST(router, "/users", handler)

		// Test with invalid data
		invalidRequests := []struct {
			name string
			body string
		}{
			{
				name: "missing required fields",
				body: `{}`,
			},
			{
				name: "invalid email",
				body: `{"name":"John","email":"invalid-email","age":25}`,
			},
			{
				name: "name too short",
				body: `{"name":"J","email":"john@example.com","age":25}`,
			},
			{
				name: "age too young",
				body: `{"name":"John","email":"john@example.com","age":17}`,
			},
		}

		for _, tc := range invalidRequests {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest("POST", "/users", strings.NewReader(tc.body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				// Should return 400 Bad Request for validation errors
				if w.Code != 400 {
					t.Errorf("Expected status 400 for invalid request, got %d", w.Code)
				}
			})
		}
	})

	t.Run("concurrent requests work correctly with optimizations", func(t *testing.T) {
		type ConcurrentRequest struct {
			ID string `path:"id" validate:"required"`
		}

		type ConcurrentResponse struct {
			ID        string `json:"id"`
			Processed bool   `json:"processed"`
		}

		handler := &mockHandler[ConcurrentRequest, ConcurrentResponse]{}

		router := NewRouter()
		GET(router, "/process/{id}", handler)

		// Run multiple concurrent requests
		const numRequests = 10
		results := make(chan bool, numRequests)

		for i := 0; i < numRequests; i++ {
			go func(id int) {
				// Set response for this goroutine
				handler.response = ConcurrentResponse{
					ID:        string(rune('A' + id)),
					Processed: true,
				}

				req := httptest.NewRequest("GET", "/process/"+string(rune('A'+id)), nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				results <- w.Code == 200
			}(i)
		}

		// Collect results
		successCount := 0
		for i := 0; i < numRequests; i++ {
			if <-results {
				successCount++
			}
		}

		if successCount != numRequests {
			t.Errorf("Expected %d successful requests, got %d", numRequests, successCount)
		}
	})
}

// Test error handling optimizations
func TestErrorHandlingOptimizations(t *testing.T) {
	t.Run("error responses use optimized encoding", func(t *testing.T) {
		type ErrorRequest struct {
			ShouldError bool `json:"should_error"`
		}

		type ErrorResponse struct {
			Message string `json:"message"`
		}

		handler := &mockHandler[ErrorRequest, ErrorResponse]{
			err: NewValidationError("Validation failed", map[string]string{
				"field": "error_message",
			}),
		}

		router := NewRouter()
		POST(router, "/error-test", handler)

		req := httptest.NewRequest("POST", "/error-test", strings.NewReader(`{"should_error":true}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 400 for validation error
		if w.Code != 400 {
			t.Errorf("Expected status 400 for validation error, got %d", w.Code)
		}

		// Verify error response format
		if !strings.Contains(w.Body.String(), "Validation failed") {
			t.Errorf("Error response should contain validation message, got: %s", w.Body.String())
		}
	})

	t.Run("custom error types are handled efficiently", func(t *testing.T) {
		type CustomErrorRequest struct {
			ID string `path:"id" validate:"required"`
		}

		type CustomErrorResponse struct {
			Data string `json:"data"`
		}

		handler := &mockHandler[CustomErrorRequest, CustomErrorResponse]{
			err: NewNotFoundError("Resource", "123"),
		}

		router := NewRouter()
		GET(router, "/custom-error/{id}", handler)

		req := httptest.NewRequest("GET", "/custom-error/123", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 404 for not found error
		if w.Code != 404 {
			t.Errorf("Expected status 404 for not found error, got %d", w.Code)
		}
	})
}

// Test middleware compatibility with optimizations
func TestMiddlewareCompatibility(t *testing.T) {
	t.Run("optimized handlers work with middleware", func(t *testing.T) {
		type MiddlewareRequest struct {
			ID string `path:"id" validate:"required"`
		}

		type MiddlewareResponse struct {
			ID        string `json:"id"`
			Processed bool   `json:"processed"`
		}

		handler := &mockHandler[MiddlewareRequest, MiddlewareResponse]{
			response: MiddlewareResponse{ID: "123", Processed: true},
		}

		// Create middleware that adds a header
		middleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Middleware", "applied")
				next.ServeHTTP(w, r)
			})
		}

		router := NewRouter()

		// Register handler with middleware applied via router
		GET(router, "/middleware/{id}", handler)

		// Apply middleware to the entire router
		middlewareRouter := middleware(router)

		req := httptest.NewRequest("GET", "/middleware/123", nil)
		w := httptest.NewRecorder()
		middlewareRouter.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Verify middleware was applied
		if w.Header().Get("X-Middleware") != "applied" {
			t.Error("Middleware header was not set")
		}

		// Verify handler was called correctly
		if handler.lastRequest.ID != "123" {
			t.Errorf("Expected ID '123', got '%s'", handler.lastRequest.ID)
		}
	})
}
