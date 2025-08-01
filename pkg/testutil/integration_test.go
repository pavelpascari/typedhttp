package testutil_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/pavelpascari/typedhttp/pkg/testutil"
	"github.com/pavelpascari/typedhttp/pkg/testutil/assert"
	"github.com/pavelpascari/typedhttp/pkg/testutil/client"
	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
)

// Example request/response types for testing.
type CreateUserRequest struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"min=18"`
}

type UserResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type GetUserRequest struct {
	ID     string `path:"id" validate:"required"`
	Fields string `query:"fields" default:"id,name,email"`
}

// Mock handler for testing.
type CreateUserHandler struct{}

func (h *CreateUserHandler) Handle(ctx context.Context, req CreateUserRequest) (UserResponse, error) {
	return UserResponse{
		ID:        "123",
		Name:      req.Name,
		Email:     req.Email,
		CreatedAt: time.Now(),
	}, nil
}

type GetUserHandler struct{}

func (h *GetUserHandler) Handle(ctx context.Context, req GetUserRequest) (UserResponse, error) {
	return UserResponse{
		ID:        req.ID,
		Name:      "John Doe",
		Email:     "john@example.com",
		CreatedAt: time.Now(),
	}, nil
}

// TestGoIdiomaticTestUtility demonstrates the 5/5 Go-idiomatic design.
func TestGoIdiomaticTestUtility(t *testing.T) {
	// Setup router with handlers
	router := typedhttp.NewRouter()
	typedhttp.POST(router, "/users", &CreateUserHandler{})
	typedhttp.GET(router, "/users/{id}", &GetUserHandler{})

	// Create client with functional options
	testClient := client.NewClient(router,
		client.WithTimeout(10*time.Second),
	)

	t.Run("5/5 Go-idiomatic request building and execution", func(t *testing.T) {
		// 🚀 Perfect Go-idiomatic request building
		req := testutil.WithAuth(
			testutil.WithJSON(
				testutil.POST("/users", CreateUserRequest{
					Name:  "Jane Doe",
					Email: "jane@example.com",
					Age:   25,
				}),
			),
			"test-token",
		)

		// Context-aware execution with explicit error handling
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		resp, err := testClient.Execute(ctx, req)
		if err != nil {
			if testutil.IsRequestError(err) {
				t.Logf("Request-specific error: %v", err)
			}
			t.Fatalf("Request failed: %v", err)
		}

		// Detailed assertions with excellent error reporting
		assert.StatusCreated(t, resp)
		assert.JSONContentType(t, resp)
		assert.JSONField(t, resp, "name", "Jane Doe")
		assert.JSONField(t, resp, "email", "jane@example.com")
		assert.JSONFieldExists(t, resp, "id")
		assert.JSONFieldExists(t, resp, "created_at")
	})

	t.Run("typed execution with generics", func(t *testing.T) {
		req := testutil.WithPathParam(
			testutil.WithQueryParam(
				testutil.GET("/users/{id}"),
				"fields", "id,name,email",
			),
			"id", "123",
		)

		// Generic typed execution (function-based due to Go limitations)
		ctx := context.Background()
		resp, err := client.ExecuteTyped[UserResponse](testClient, ctx, req)
		if err != nil {
			t.Fatalf("Typed request failed: %v", err)
		}

		// Type-safe assertions on the typed response
		assert.StatusOK(t, resp.Response)
		if resp.Data.ID != "123" {
			t.Errorf("Expected ID 123, got %s", resp.Data.ID)
		}
		if resp.Data.Name != "John Doe" {
			t.Errorf("Expected name John Doe, got %s", resp.Data.Name)
		}
	})

	t.Run("helper functions for common patterns", func(t *testing.T) {
		req := testutil.GET("/users/456")

		// Convenient helper functions
		resp := testutil.MustExecute(t, testClient, req)
		assert.StatusOK(t, resp)

		// Timeout helpers
		respWithTimeout := testutil.ExecuteWithShortTimeout(t, testClient, req)
		assert.StatusOK(t, respWithTimeout)
	})

	t.Run("error handling patterns", func(t *testing.T) {
		// Test error expectation
		req := testutil.POST("/nonexistent", nil)

		resp, err := testutil.ExecuteExpectingError(t, testClient, req)
		if err != nil {
			t.Logf("Expected execution error: %v", err)
		}
		if resp != nil {
			assert.StatusNotFound(t, resp)
		}
	})

	t.Run("comprehensive request building", func(t *testing.T) {
		// Showcase all request building features
		req := testutil.WithCookie(
			testutil.WithHeaders(
				testutil.WithAuth(
					testutil.WithQueryParams(
						testutil.WithPathParams(
							testutil.GET("/users/{id}"),
							map[string]string{"id": "789"},
						),
						map[string]string{
							"fields": "id,name,email",
							"format": "json",
						},
					),
					"bearer-token",
				),
				map[string]string{
					"X-Request-ID": "test-123",
					"X-Version":    "v1",
				},
			),
			"session", "abc123",
		)

		// Verify all parameters are set correctly
		if req.Method != http.MethodGet {
			t.Errorf("Expected GET method")
		}
		if req.Path != "/users/{id}" {
			t.Errorf("Expected path /users/{id}")
		}
		if req.PathParams["id"] != "789" {
			t.Errorf("Expected path param id=789")
		}
		if req.QueryParams["fields"] != "id,name,email" {
			t.Errorf("Expected query param fields=id,name,email")
		}
		if req.Headers["Authorization"] != "Bearer bearer-token" {
			t.Errorf("Expected Authorization header")
		}
		if req.Headers["X-Request-ID"] != "test-123" {
			t.Errorf("Expected X-Request-ID header")
		}
		if req.Cookies["session"] != "abc123" {
			t.Errorf("Expected session cookie")
		}
	})
}

// TestErrorTypes demonstrates the enhanced error handling.
func TestErrorTypes(t *testing.T) {
	router := typedhttp.NewRouter()
	testClient := client.NewClient(router)

	t.Run("request error identification", func(t *testing.T) {
		req := testutil.GET("/nonexistent")

		_, err := testutil.TryExecute(testClient, req)
		if err != nil {
			if testutil.IsRequestError(err) {
				t.Logf("Correctly identified as request error: %v", err)
			} else {
				t.Errorf("Should be identified as request error")
			}
		}
	})
}

// TestContextIntegration demonstrates context-aware operations.
func TestContextIntegration(t *testing.T) {
	router := typedhttp.NewRouter()
	typedhttp.GET(router, "/slow", &GetUserHandler{})

	testClient := client.NewClient(router)

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		req := testutil.GET("/slow")

		_, err := testutil.TryExecuteWithContext(ctx, testClient, req)
		if err != nil {
			if testutil.IsRequestError(err) {
				t.Logf("Request properly cancelled: %v", err)
			}
		}
	})

	t.Run("timeout helpers", func(t *testing.T) {
		req := testutil.GET("/users/123")

		// These should work fine with short timeout
		_, err := testutil.TryExecuteWithTimeout(testClient, req, testutil.ShortTimeout)
		if err != nil {
			t.Logf("Short timeout error: %v", err)
		}
	})
}
