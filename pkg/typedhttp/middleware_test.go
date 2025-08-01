package typedhttp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types for middleware testing
type TestRequest struct {
	UserID string `json:"user_id"`
	Action string `json:"action"`
}

type TestResponse struct {
	Result string `json:"result"`
	Status string `json:"status"`
}

// Test middleware implementations
type TestPreMiddleware struct {
	called bool
	err    error
}

func (m *TestPreMiddleware) Before(ctx context.Context, req *TestRequest) (context.Context, error) {
	m.called = true
	if m.err != nil {
		return ctx, m.err
	}
	return context.WithValue(ctx, "pre_middleware", "executed"), nil
}

type TestPostMiddleware struct {
	called bool
	err    error
}

func (m *TestPostMiddleware) After(ctx context.Context, resp *TestResponse) (*TestResponse, error) {
	m.called = true
	if m.err != nil {
		return resp, m.err
	}
	resp.Status = "processed"

	return resp, nil
}

type TestFullMiddleware struct {
	beforeCalled bool
	afterCalled  bool
	beforeErr    error
	afterErr     error
}

func (m *TestFullMiddleware) Before(ctx context.Context, req *TestRequest) (context.Context, error) {
	m.beforeCalled = true
	if m.beforeErr != nil {
		return ctx, m.beforeErr
	}

	return context.WithValue(ctx, "full_middleware", "before"), nil
}

func (m *TestFullMiddleware) After(
	ctx context.Context,
	req *TestRequest, resp *TestResponse,
	err error,
) (*TestResponse, error) {
	m.afterCalled = true
	if m.afterErr != nil {
		return resp, m.afterErr
	}
	if resp != nil {
		resp.Result = "enhanced"
	}
	return resp, err
}

// TestMiddlewareConfig tests the middleware configuration struct
func TestMiddlewareConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   MiddlewareConfig
		expected MiddlewareConfig
	}{
		{
			name: "default_config",
			config: MiddlewareConfig{
				Name:     "test",
				Priority: 0,
				Scope:    ScopeHandler,
			},
			expected: MiddlewareConfig{
				Name:        "test",
				Priority:    0,
				Scope:       ScopeHandler,
				Conditional: nil,
				Metadata:    nil,
			},
		},
		{
			name: "full_config",
			config: MiddlewareConfig{
				Name:     "auth",
				Priority: 100,
				Scope:    ScopeGlobal,
				Conditional: func(r *http.Request) bool {
					return r.Header.Get("Authorization") != ""
				},
				Metadata: map[string]any{"role": "admin"},
			},
			expected: MiddlewareConfig{
				Name:     "auth",
				Priority: 100,
				Scope:    ScopeGlobal,
				Metadata: map[string]any{"role": "admin"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected.Name, tt.config.Name)
			assert.Equal(t, tt.expected.Priority, tt.config.Priority)
			assert.Equal(t, tt.expected.Scope, tt.config.Scope)
			assert.Equal(t, tt.expected.Metadata, tt.config.Metadata)

			// For the full_config test, we expect a conditional function to be present
			if tt.name == "full_config" {
				assert.NotNil(t, tt.config.Conditional)
			} else {
				assert.Nil(t, tt.config.Conditional)
			}
		})
	}
}

// TestMiddlewareBuilder tests the middleware builder pattern
func TestMiddlewareBuilder(t *testing.T) {
	t.Run("empty_builder", func(t *testing.T) {
		builder := NewMiddlewareBuilder()
		entries := builder.Build()
		assert.Empty(t, entries)
	})

	t.Run("single_middleware", func(t *testing.T) {
		middleware := func(next http.Handler) http.Handler {
			return next
		}

		builder := NewMiddlewareBuilder()
		entries := builder.Add(middleware, WithName("test")).Build()

		require.Len(t, entries, 1)
		assert.Equal(t, "test", entries[0].Config.Name)
		assert.Equal(t, 0, entries[0].Config.Priority)
		assert.Equal(t, ScopeHandler, entries[0].Config.Scope)
	})

	t.Run("multiple_middleware_with_priority", func(t *testing.T) {
		mw1 := func(next http.Handler) http.Handler { return next }
		mw2 := func(next http.Handler) http.Handler { return next }
		mw3 := func(next http.Handler) http.Handler { return next }

		builder := NewMiddlewareBuilder()
		entries := builder.
			Add(mw1, WithName("low"), WithPriority(-10)).
			Add(mw2, WithName("high"), WithPriority(10)).
			Add(mw3, WithName("medium"), WithPriority(0)).
			Build()

		require.Len(t, entries, 3)
		// Should be sorted by priority (highest first)
		assert.Equal(t, "high", entries[0].Config.Name)
		assert.Equal(t, 10, entries[0].Config.Priority)
		assert.Equal(t, "medium", entries[1].Config.Name)
		assert.Equal(t, 0, entries[1].Config.Priority)
		assert.Equal(t, "low", entries[2].Config.Name)
		assert.Equal(t, -10, entries[2].Config.Priority)
	})

	t.Run("conditional_middleware", func(t *testing.T) {
		middleware := func(next http.Handler) http.Handler {
			return next
		}

		condition := func(r *http.Request) bool {
			return r.Header.Get("X-Test") == "true"
		}

		builder := NewMiddlewareBuilder()
		entries := builder.
			Add(middleware, WithName("conditional")).
			OnlyFor(condition).
			Build()

		require.Len(t, entries, 1)
		assert.NotNil(t, entries[0].Config.Conditional)

		// Test condition
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		assert.False(t, entries[0].Config.Conditional(req))

		req.Header.Set("X-Test", "true")
		assert.True(t, entries[0].Config.Conditional(req))
	})

	t.Run("fluent_chaining", func(t *testing.T) {
		mw1 := func(next http.Handler) http.Handler { return next }
		mw2 := func(next http.Handler) http.Handler { return next }

		condition := func(r *http.Request) bool { return true }

		builder := NewMiddlewareBuilder()
		entries := builder.
			Add(mw1, WithName("first"), WithPriority(5)).
			WithPriority(10). // Should update the last added middleware
			Add(mw2, WithName("second")).
			OnlyFor(condition). // Should update the last added middleware
			Build()

		require.Len(t, entries, 2)

		// First middleware should have updated priority
		assert.Equal(t, "first", entries[0].Config.Name)
		assert.Equal(t, 10, entries[0].Config.Priority)
		assert.Nil(t, entries[0].Config.Conditional)

		// Second middleware should have condition
		assert.Equal(t, "second", entries[1].Config.Name)
		assert.Equal(t, 0, entries[1].Config.Priority)
		assert.NotNil(t, entries[1].Config.Conditional)
	})
}

// TestMiddlewareRegistry tests the middleware registry
func TestMiddlewareRegistry(t *testing.T) {
	t.Run("empty_registry", func(t *testing.T) {
		registry := NewMiddlewareRegistry()

		global := registry.GetGlobal()
		assert.Empty(t, global)

		groups := registry.GetGroups()
		assert.Empty(t, groups)

		handlers := registry.GetHandlers()
		assert.Empty(t, handlers)
	})

	t.Run("register_global_middleware", func(t *testing.T) {
		registry := NewMiddlewareRegistry()

		middleware := func(next http.Handler) http.Handler { return next }
		entry := MiddlewareEntry{
			Middleware: middleware,
			Config: MiddlewareConfig{
				Name:  "global_test",
				Scope: ScopeGlobal,
			},
		}

		registry.RegisterGlobal(entry)

		global := registry.GetGlobal()
		require.Len(t, global, 1)
		assert.Equal(t, "global_test", global[0].Config.Name)
	})

	t.Run("register_group_middleware", func(t *testing.T) {
		registry := NewMiddlewareRegistry()

		middleware := func(next http.Handler) http.Handler { return next }
		entry := MiddlewareEntry{
			Middleware: middleware,
			Config: MiddlewareConfig{
				Name:  "group_test",
				Scope: ScopeGroup,
			},
		}

		registry.RegisterGroup("api/v1", entry)

		groups := registry.GetGroups()
		require.Contains(t, groups, "api/v1")
		require.Len(t, groups["api/v1"], 1)
		assert.Equal(t, "group_test", groups["api/v1"][0].Config.Name)
	})

	t.Run("register_handler_middleware", func(t *testing.T) {
		registry := NewMiddlewareRegistry()

		middleware := func(next http.Handler) http.Handler { return next }
		entry := MiddlewareEntry{
			Middleware: middleware,
			Config: MiddlewareConfig{
				Name:  "handler_test",
				Scope: ScopeHandler,
			},
		}

		registry.RegisterHandler("GET /users/{id}", entry)

		handlers := registry.GetHandlers()
		require.Contains(t, handlers, "GET /users/{id}")
		require.Len(t, handlers["GET /users/{id}"], 1)
		assert.Equal(t, "handler_test", handlers["GET /users/{id}"][0].Config.Name)
	})

	t.Run("concurrent_access", func(t *testing.T) {
		registry := NewMiddlewareRegistry()

		// Test concurrent access to registry
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func(id int) {
				middleware := func(next http.Handler) http.Handler { return next }
				entry := MiddlewareEntry{
					Middleware: middleware,
					Config: MiddlewareConfig{
						Name:  "concurrent_test",
						Scope: ScopeGlobal,
					},
				}
				registry.RegisterGlobal(entry)
				done <- true
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}

		global := registry.GetGlobal()
		assert.Len(t, global, 10)
	})
}

// TestTypedPreMiddleware tests typed pre-middleware functionality
func TestTypedPreMiddleware(t *testing.T) {
	t.Run("successful_execution", func(t *testing.T) {
		middleware := &TestPreMiddleware{}
		ctx := context.Background()
		req := &TestRequest{UserID: "123", Action: "test"}

		newCtx, err := middleware.Before(ctx, req)

		require.NoError(t, err)
		assert.True(t, middleware.called)
		assert.Equal(t, "executed", newCtx.Value("pre_middleware"))
	})

	t.Run("error_handling", func(t *testing.T) {
		expectedErr := errors.New("pre middleware error")
		middleware := &TestPreMiddleware{err: expectedErr}
		ctx := context.Background()
		req := &TestRequest{UserID: "123", Action: "test"}

		newCtx, err := middleware.Before(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.True(t, middleware.called)
		assert.Equal(t, ctx, newCtx) // Context should be unchanged on error
	})
}

// TestTypedPostMiddleware tests typed post-middleware functionality
func TestTypedPostMiddleware(t *testing.T) {
	t.Run("successful_execution", func(t *testing.T) {
		middleware := &TestPostMiddleware{}
		ctx := context.Background()
		resp := &TestResponse{Result: "original", Status: "pending"}

		newResp, err := middleware.After(ctx, resp)

		require.NoError(t, err)
		assert.True(t, middleware.called)
		assert.Equal(t, "original", newResp.Result)
		assert.Equal(t, "processed", newResp.Status)
	})

	t.Run("error_handling", func(t *testing.T) {
		expectedErr := errors.New("post middleware error")
		middleware := &TestPostMiddleware{err: expectedErr}
		ctx := context.Background()
		resp := &TestResponse{Result: "original", Status: "pending"}

		newResp, err := middleware.After(ctx, resp)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.True(t, middleware.called)
		assert.Equal(t, resp, newResp) // Response should be unchanged on error
	})
}

// TestTypedFullMiddleware tests typed full middleware functionality
func TestTypedFullMiddleware(t *testing.T) {
	t.Run("successful_execution", func(t *testing.T) {
		middleware := &TestFullMiddleware{}
		ctx := context.Background()
		req := &TestRequest{UserID: "123", Action: "test"}
		resp := &TestResponse{Result: "original", Status: "pending"}

		// Test Before phase
		newCtx, err := middleware.Before(ctx, req)
		require.NoError(t, err)
		assert.True(t, middleware.beforeCalled)
		assert.Equal(t, "before", newCtx.Value("full_middleware"))

		// Test After phase
		newResp, err := middleware.After(newCtx, req, resp, nil)
		require.NoError(t, err)
		assert.True(t, middleware.afterCalled)
		assert.Equal(t, "enhanced", newResp.Result)
		assert.Equal(t, "pending", newResp.Status)
	})

	t.Run("before_error_handling", func(t *testing.T) {
		expectedErr := errors.New("before error")
		middleware := &TestFullMiddleware{beforeErr: expectedErr}
		ctx := context.Background()
		req := &TestRequest{UserID: "123", Action: "test"}

		newCtx, err := middleware.Before(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.True(t, middleware.beforeCalled)
		assert.False(t, middleware.afterCalled)
		assert.Equal(t, ctx, newCtx)
	})

	t.Run("after_error_handling", func(t *testing.T) {
		expectedErr := errors.New("after error")
		middleware := &TestFullMiddleware{afterErr: expectedErr}
		ctx := context.Background()
		req := &TestRequest{UserID: "123", Action: "test"}
		resp := &TestResponse{Result: "original", Status: "pending"}

		// Before should succeed
		newCtx, err := middleware.Before(ctx, req)
		require.NoError(t, err)

		// After should fail
		newResp, err := middleware.After(newCtx, req, resp, nil)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.True(t, middleware.afterCalled)
		assert.Equal(t, resp, newResp)
	})

	t.Run("with_handler_error", func(t *testing.T) {
		middleware := &TestFullMiddleware{}
		ctx := context.Background()
		req := &TestRequest{UserID: "123", Action: "test"}
		resp := &TestResponse{Result: "original", Status: "pending"}
		handlerErr := errors.New("handler error")

		// Before should succeed
		newCtx, err := middleware.Before(ctx, req)
		require.NoError(t, err)

		// After should still process but preserve handler error
		newResp, err := middleware.After(newCtx, req, resp, handlerErr)
		assert.Error(t, err)
		assert.Equal(t, handlerErr, err)
		assert.True(t, middleware.afterCalled)
		assert.Equal(t, "enhanced", newResp.Result) // Middleware should still enhance response
	})
}

// TestMiddlewareScope tests middleware scope enumeration
func TestMiddlewareScope(t *testing.T) {
	tests := []struct {
		name     string
		scope    MiddlewareScope
		expected MiddlewareScope
	}{
		{"global", ScopeGlobal, 0},
		{"group", ScopeGroup, 1},
		{"handler", ScopeHandler, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.scope)
		})
	}
}

// TestConditionalFunc tests conditional middleware execution
func TestConditionalFunc(t *testing.T) {
	t.Run("header_based_condition", func(t *testing.T) {
		condition := func(r *http.Request) bool {
			return r.Header.Get("X-Test") == "enabled"
		}

		req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
		assert.False(t, condition(req1))

		req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
		req2.Header.Set("X-Test", "enabled")
		assert.True(t, condition(req2))

		req3 := httptest.NewRequest(http.MethodGet, "/test", nil)
		req3.Header.Set("X-Test", "disabled")
		assert.False(t, condition(req3))
	})

	t.Run("path_based_condition", func(t *testing.T) {
		condition := func(r *http.Request) bool {
			return r.URL.Path == "/api/v1/secure"
		}

		req1 := httptest.NewRequest(http.MethodGet, "/api/v1/secure", nil)
		assert.True(t, condition(req1))

		req2 := httptest.NewRequest(http.MethodGet, "/api/v1/public", nil)
		assert.False(t, condition(req2))
	})

	t.Run("method_based_condition", func(t *testing.T) {
		condition := func(r *http.Request) bool {
			return r.Method == http.MethodPost || r.Method == http.MethodPut
		}

		req1 := httptest.NewRequest(http.MethodPost, "/test", nil)
		assert.True(t, condition(req1))

		req2 := httptest.NewRequest(http.MethodPut, "/test", nil)
		assert.True(t, condition(req2))

		req3 := httptest.NewRequest(http.MethodGet, "/test", nil)
		assert.False(t, condition(req3))
	})
}
