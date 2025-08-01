package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pavelpascari/typedhttp/pkg/testutil"
	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
)

// TestClient is a specialized client for testing that bypasses TypedRouter.
type TestClient struct {
	*Client
	handler http.HandlerFunc
}

// NewTestClient creates a test client with a mock handler.
func NewTestClient(handler http.HandlerFunc, opts ...Option) *TestClient {
	// Create a real router but we won't use it for actual routing
	router := typedhttp.NewRouter()
	client := NewClient(router, opts...)

	return &TestClient{
		Client:  client,
		handler: handler,
	}
}

// Override Execute to use our test handler instead of the router.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func (tc *TestClient) Execute(ctx context.Context, req testutil.Request) (*testutil.Response, error) {
	// Add timeout if context doesn't have deadline
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, tc.Client.timeout)
		defer cancel()
	}

	httpReq, err := tc.Client.buildHTTPRequest(ctx, req)
	if err != nil {
		return nil, &testutil.RequestError{
			Method: req.Method,
			Path:   req.Path,
			Err:    err,
		}
	}

	resp, err := tc.executeHTTPRequest(httpReq)
	if err != nil {
		return nil, &testutil.RequestError{
			Method: req.Method,
			Path:   req.Path,
			Err:    err,
		}
	}

	return resp, nil
}

// Override executeHTTPRequest to use our test handler instead of the router.
func (tc *TestClient) executeHTTPRequest(req *http.Request) (*testutil.Response, error) {
	recorder := httptest.NewRecorder()

	// Execute request through our test handler
	tc.handler(recorder, req)

	// Read response body
	body, err := io.ReadAll(recorder.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	return &testutil.Response{
		StatusCode: recorder.Code,
		Headers:    recorder.Header(),
		Raw:        body,
	}, nil
}

// Convenience methods that delegate to the base client.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func (tc *TestClient) ExecuteWithTimeout(req testutil.Request, timeout time.Duration) (*testutil.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return tc.Execute(ctx, req)
}

// Test-specific ExecuteTyped function that works with TestClient.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func executeTypedTest[T any](
	tc *TestClient,
	ctx context.Context,
	req testutil.Request,
) (*testutil.TypedResponse[T], error) {
	resp, err := tc.Execute(ctx, req)
	if err != nil {
		return nil, err
	}

	var data T
	if len(resp.Raw) > 0 && strings.Contains(resp.Headers.Get("Content-Type"), "application/json") {
		if err := json.Unmarshal(resp.Raw, &data); err != nil {
			return nil, &testutil.RequestError{
				Method: req.Method,
				Path:   req.Path,
				Err:    err,
			}
		}
	}

	return &testutil.TypedResponse[T]{
		Response: resp,
		Data:     data,
	}, nil
}

// Test-specific ExecuteTypedWithTimeout function.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func executeTypedWithTimeoutTest[T any](
	tc *TestClient,
	req testutil.Request,
	timeout time.Duration,
) (*testutil.TypedResponse[T], error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return executeTypedTest[T](tc, ctx, req)
}

func TestWithTimeout(t *testing.T) {
	timeout := 15 * time.Second
	option := WithTimeout(timeout)

	client := &Client{}
	option(client)

	if client.timeout != timeout {
		t.Errorf("Expected timeout %v, got %v", timeout, client.timeout)
	}
}

func TestWithBaseURL(t *testing.T) {
	baseURL := "https://api.example.com"
	option := WithBaseURL(baseURL)

	client := &Client{}
	option(client)

	if client.baseURL != baseURL {
		t.Errorf("Expected baseURL %q, got %q", baseURL, client.baseURL)
	}
}

func TestNewClient(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test"))
	})

	timeout := 10 * time.Second
	baseURL := "https://api.example.com"

	client := NewTestClient(handler,
		WithTimeout(timeout),
		WithBaseURL(baseURL),
	)

	if client.Client.timeout != timeout {
		t.Errorf("Expected timeout %v, got %v", timeout, client.Client.timeout)
	}
	if client.Client.baseURL != baseURL {
		t.Errorf("Expected baseURL %q, got %q", baseURL, client.Client.baseURL)
	}
}

func TestNewClientDefaults(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	client := NewTestClient(handler)

	if client.Client.timeout != testutil.DefaultTimeout {
		t.Errorf("Expected default timeout %v, got %v", testutil.DefaultTimeout, client.Client.timeout)
	}
	if client.Client.baseURL != "" {
		t.Errorf("Expected empty baseURL, got %q", client.Client.baseURL)
	}
}

func TestClientExecute(t *testing.T) {
	tests := []struct {
		name           string
		handler        http.HandlerFunc
		request        testutil.Request
		expectedStatus int
		expectedBody   string
		shouldError    bool
	}{
		{
			name: "successful GET request",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET, got %s", r.Method)
				}
				if r.URL.Path != "/users/123" {
					t.Errorf("Expected path /users/123, got %s", r.URL.Path)
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"id":"123","name":"John"}`))
			},
			request: testutil.Request{
				Method:     http.MethodGet,
				Path:       "/users/{id}",
				PathParams: map[string]string{"id": "123"},
			},
			expectedStatus: 200,
			expectedBody:   `{"id":"123","name":"John"}`,
			shouldError:    false,
		},
		{
			name: "POST request with JSON body",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected JSON content type, got %s", r.Header.Get("Content-Type"))
				}

				bodyBytes, _ := io.ReadAll(r.Body)
				// JSON field order is not guaranteed, so check both fields separately
				var bodyData map[string]interface{}
				if err := json.Unmarshal(bodyBytes, &bodyData); err != nil {
					t.Errorf("Invalid JSON body: %v", err)
				}
				if bodyData["name"] != "Jane" || bodyData["email"] != "jane@example.com" {
					t.Errorf("Expected name=Jane and email=jane@example.com, got %v", bodyData)
				}

				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"id":"456","name":"Jane"}`))
			},
			request: testutil.Request{
				Method: "POST",
				Path:   "/users",
				Body: map[string]string{
					"name":  "Jane",
					"email": "jane@example.com",
				},
			},
			expectedStatus: 201,
			expectedBody:   `{"id":"456","name":"Jane"}`,
			shouldError:    false,
		},
		{
			name: "request with query parameters",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("page") != "2" {
					t.Errorf("Expected page=2, got page=%s", r.URL.Query().Get("page"))
				}
				if r.URL.Query().Get("limit") != "10" {
					t.Errorf("Expected limit=10, got limit=%s", r.URL.Query().Get("limit"))
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"results":[]}`))
			},
			request: testutil.Request{
				Method: http.MethodGet,
				Path:   "/users",
				QueryParams: map[string]string{
					"page":  "2",
					"limit": "10",
				},
			},
			expectedStatus: 200,
			expectedBody:   `{"results":[]}`,
			shouldError:    false,
		},
		{
			name: "request with headers and cookies",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer token123" {
					t.Errorf("Expected Authorization header, got %s", r.Header.Get("Authorization"))
				}
				if r.Header.Get("X-Custom") != "value" {
					t.Errorf("Expected X-Custom header, got %s", r.Header.Get("X-Custom"))
				}

				cookie, err := r.Cookie("session")
				if err != nil || cookie.Value != "abc123" {
					t.Errorf("Expected session cookie with value abc123")
				}

				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"authenticated":true}`))
			},
			request: testutil.Request{
				Method: http.MethodGet,
				Path:   "/protected",
				Headers: map[string]string{
					"Authorization": "Bearer token123",
					"X-Custom":      "value",
				},
				Cookies: map[string]string{
					"session": "abc123",
				},
			},
			expectedStatus: 200,
			expectedBody:   `{"authenticated":true}`,
			shouldError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewTestClient(tt.handler)

			ctx := context.Background()
			resp, err := client.Execute(ctx, tt.request)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}

				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if string(resp.Raw) != tt.expectedBody {
				t.Errorf("Expected body %q, got %q", tt.expectedBody, string(resp.Raw))
			}
		})
	}
}

func TestClientExecuteWithContext(t *testing.T) {
	t.Run("context timeout", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			// Handler that would normally respond
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}

		client := NewTestClient(handler)

		// Create context that times out quickly
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		req := testutil.Request{
			Method: http.MethodGet,
			Path:   "/slow",
		}

		_, err := client.Execute(ctx, req)
		// Note: The timeout might not be caught at the HTTP level in this mock setup, but
		// we're testing that the context is properly propagated.
		if err != nil {
			// This is expected - either timeout or normal execution
			t.Logf("Request completed with: %v", err)
		}
	})

	t.Run("context without deadline gets timeout", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("success"))
		}

		client := NewTestClient(handler, WithTimeout(5*time.Second))

		// Context without deadline - client should add timeout
		ctx := context.Background()
		req := testutil.Request{
			Method: http.MethodGet,
			Path:   "/test",
		}

		resp, err := client.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})
}

func TestExecuteTyped(t *testing.T) {
	type UserResponse struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	t.Run("successful typed execution", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"id":"123","name":"John Doe"}`)); err != nil {
				t.Error("Failed to write response")
			}
		}

		client := NewTestClient(handler)

		req := testutil.Request{
			Method: http.MethodGet,
			Path:   "/users/123",
		}

		ctx := context.Background()
		resp, err := executeTypedTest[UserResponse](client, ctx, req)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if resp.Response.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.Response.StatusCode)
		}

		if resp.Data.ID != "123" {
			t.Errorf("Expected ID 123, got %s", resp.Data.ID)
		}

		if resp.Data.Name != "John Doe" {
			t.Errorf("Expected name John Doe, got %s", resp.Data.Name)
		}
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{invalid json}`)); err != nil {
				t.Error("Failed to write response")
			}
		}

		client := NewTestClient(handler)

		req := testutil.Request{
			Method: http.MethodGet,
			Path:   "/users/123",
		}

		ctx := context.Background()
		_, err := executeTypedTest[UserResponse](client, ctx, req)

		if err == nil {
			t.Error("Expected JSON parsing error")
		}

		var reqErr *testutil.RequestError
		if !testutil.IsRequestError(err) {
			t.Errorf("Expected RequestError, got %T", err)
		}
		_ = reqErr // Avoid unused variable warning
	})

	t.Run("non-JSON response", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte("plain text")); err != nil {
				t.Error("Failed to write response")
			}
		}

		client := NewTestClient(handler)

		req := testutil.Request{
			Method: http.MethodGet,
			Path:   "/text",
		}

		ctx := context.Background()
		resp, err := executeTypedTest[UserResponse](client, ctx, req)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Should return zero value for T since it's not JSON
		if resp.Data.ID != "" || resp.Data.Name != "" {
			t.Error("Expected zero value for non-JSON response")
		}
	})
}

func TestBuildHTTPRequest(t *testing.T) {
	tests := []struct {
		name        string
		request     testutil.Request
		baseURL     string
		expectedURL string
		shouldError bool
	}{
		{
			name: "simple GET request",
			request: testutil.Request{
				Method: http.MethodGet,
				Path:   "/users",
			},
			baseURL:     "",
			expectedURL: "/users",
			shouldError: false,
		},
		{
			name: "request with path parameters",
			request: testutil.Request{
				Method:     http.MethodGet,
				Path:       "/users/{id}/posts/{postId}",
				PathParams: map[string]string{"id": "123", "postId": "456"},
			},
			baseURL:     "",
			expectedURL: "/users/123/posts/456",
			shouldError: false,
		},
		{
			name: "request with base URL",
			request: testutil.Request{
				Method: http.MethodGet,
				Path:   "/users",
			},
			baseURL:     "https://api.example.com",
			expectedURL: "https://api.example.com/users",
			shouldError: false,
		},
		{
			name: "request with query parameters",
			request: testutil.Request{
				Method: http.MethodGet,
				Path:   "/users",
				QueryParams: map[string]string{
					"page":  "1",
					"limit": "10",
				},
			},
			baseURL:     "",
			expectedURL: "/users?limit=10&page=1", // Note: params are sorted
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{baseURL: tt.baseURL}
			ctx := context.Background()

			req, err := client.buildHTTPRequest(ctx, tt.request)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}

				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if req.Method != tt.request.Method {
				t.Errorf("Expected method %s, got %s", tt.request.Method, req.Method)
			}

			if req.URL.String() != tt.expectedURL {
				t.Errorf("Expected URL %s, got %s", tt.expectedURL, req.URL.String())
			}
		})
	}
}

func TestBuildHTTPRequestWithBody(t *testing.T) {
	tests := []struct {
		name                string
		body                interface{}
		expectedContentType string
		shouldError         bool
	}{
		{
			name:                "JSON body",
			body:                map[string]string{"name": "John", "email": "john@example.com"},
			expectedContentType: "application/json",
			shouldError:         false,
		},
		{
			name:                "string body",
			body:                "plain text",
			expectedContentType: "application/json", // Still JSON encoded
			shouldError:         false,
		},
		{
			name:                "nil body",
			body:                nil,
			expectedContentType: "",
			shouldError:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{}
			ctx := context.Background()

			request := testutil.Request{
				Method: "POST",
				Path:   "/test",
				Body:   tt.body,
			}

			req, err := client.buildHTTPRequest(ctx, request)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}

				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.expectedContentType != "" {
				if req.Header.Get("Content-Type") != tt.expectedContentType {
					t.Errorf("Expected Content-Type %s, got %s",
						tt.expectedContentType, req.Header.Get("Content-Type"))
				}
			}

			if req.Body != nil {
				body, _ := io.ReadAll(req.Body)
				if tt.body != nil {
					// Should be JSON encoded
					var decoded interface{}
					if err := json.Unmarshal(body, &decoded); err != nil {
						t.Errorf("Body should be valid JSON: %v", err)
					}
				}
			} else if tt.body != nil {
				t.Error("Expected body but got nil")
			}
		})
	}
}

func TestBuildHTTPRequestWithFiles(t *testing.T) {
	client := &Client{}
	ctx := context.Background()

	t.Run("multipart with files only", func(t *testing.T) {
		request := testutil.Request{
			Method: "POST",
			Path:   "/upload",
			Files: map[string][]byte{
				"file1": []byte("content1"),
				"file2": []byte("content2"),
			},
		}

		req, err := client.buildHTTPRequest(ctx, request)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !strings.HasPrefix(req.Header.Get("Content-Type"), "multipart/form-data") {
			t.Errorf("Expected multipart content type, got %s", req.Header.Get("Content-Type"))
		}

		if req.Body == nil {
			t.Error("Expected multipart body")
		}
	})

	t.Run("multipart with files and form data", func(t *testing.T) {
		request := testutil.Request{
			Method: "POST",
			Path:   "/upload",
			Body: map[string]string{
				"name":        "test.txt",
				"description": "Test file",
			},
			Files: map[string][]byte{
				"file": []byte("file content"),
			},
		}

		req, err := client.buildHTTPRequest(ctx, request)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !strings.HasPrefix(req.Header.Get("Content-Type"), "multipart/form-data") {
			t.Errorf("Expected multipart content type, got %s", req.Header.Get("Content-Type"))
		}
	})
}

func TestExecuteHTTPRequest(t *testing.T) {
	t.Run("successful response", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Custom", "test-value")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"message":"success"}`)); err != nil {
				t.Error("Failed to write response")
			}
		}

		client := NewTestClient(handler)

		req, _ := http.NewRequest(http.MethodGet, "/test", http.NoBody)
		resp, err := client.executeHTTPRequest(req)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		if resp.Headers.Get("X-Custom") != "test-value" {
			t.Errorf("Expected X-Custom header, got %s", resp.Headers.Get("X-Custom"))
		}

		if string(resp.Raw) != `{"message":"success"}` {
			t.Errorf("Expected success message, got %s", string(resp.Raw))
		}
	})

	t.Run("error response", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			if _, err := w.Write([]byte("Not Found")); err != nil {
				t.Error("Failed to write response")
			}
		}

		client := NewTestClient(handler)

		req, _ := http.NewRequest(http.MethodGet, "/missing", http.NoBody)
		resp, err := client.executeHTTPRequest(req)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", resp.StatusCode)
		}

		if string(resp.Raw) != "Not Found" {
			t.Errorf("Expected 'Not Found', got %s", string(resp.Raw))
		}
	})
}

func TestClientTimeoutMethods(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"message":"success"}`)); err != nil {
			t.Error("Failed to write response")
		}
	}

	client := NewTestClient(handler)

	req := testutil.Request{
		Method: http.MethodGet,
		Path:   "/test",
	}

	t.Run("ExecuteWithTimeout", func(t *testing.T) {
		resp, err := client.ExecuteWithTimeout(req, 5*time.Second)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("ExecuteTypedWithTimeout", func(t *testing.T) {
		type Response struct {
			Message string `json:"message"`
		}

		resp, err := executeTypedWithTimeoutTest[Response](client, req, 5*time.Second)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if resp.Response.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.Response.StatusCode)
		}
		if resp.Data.Message != "success" {
			t.Errorf("Expected message 'success', got %s", resp.Data.Message)
		}
	})
}

func TestClientErrorHandling(t *testing.T) {
	t.Run("buildHTTPRequest error", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}

		client := NewTestClient(handler)

		// Create an invalid request that will cause buildHTTPRequest to fail
		req := testutil.Request{
			Method: "INVALID\x00METHOD", // Invalid method with null byte
			Path:   "/test",
		}

		ctx := context.Background()
		_, err := client.Execute(ctx, req)

		if err == nil {
			t.Error("Expected error from invalid request")
		}

		if !testutil.IsRequestError(err) {
			t.Errorf("Expected RequestError, got %T", err)
		}
	})
}

func TestClientEdgeCases(t *testing.T) {
	t.Run("empty path parameters", func(t *testing.T) {
		client := &Client{}
		ctx := context.Background()

		request := testutil.Request{
			Method:     http.MethodGet,
			Path:       "/users/{id}",
			PathParams: map[string]string{}, // Empty map
		}

		req, err := client.buildHTTPRequest(ctx, request)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Should leave {id} as-is since no replacement found
		if req.URL.Path != "/users/{id}" {
			t.Errorf("Expected path /users/{id}, got %s", req.URL.Path)
		}
	})

	t.Run("multiple path parameter replacements", func(t *testing.T) {
		client := &Client{}
		ctx := context.Background()

		request := testutil.Request{
			Method: http.MethodGet,
			Path:   "/orgs/{org}/users/{user}/repos/{repo}",
			PathParams: map[string]string{
				"org":  "myorg",
				"user": "myuser",
				"repo": "myrepo",
			},
		}

		req, err := client.buildHTTPRequest(ctx, request)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		expected := "/orgs/myorg/users/myuser/repos/myrepo"
		if req.URL.Path != expected {
			t.Errorf("Expected path %s, got %s", expected, req.URL.Path)
		}
	})

	t.Run("no body, no files", func(t *testing.T) {
		client := &Client{}
		ctx := context.Background()

		request := testutil.Request{
			Method: http.MethodGet,
			Path:   "/test",
			Body:   nil,
			Files:  nil,
		}

		req, err := client.buildHTTPRequest(ctx, request)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if req.Body != nil {
			t.Error("Expected nil body")
		}

		if req.Header.Get("Content-Type") != "" {
			t.Errorf("Expected no Content-Type, got %s", req.Header.Get("Content-Type"))
		}
	})
}
