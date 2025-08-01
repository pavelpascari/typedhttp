package testutil

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"
)

var (
	errNetwork  = errors.New("network error")
	errTest     = errors.New("test error")
	errRegular  = errors.New("regular error")
	errOriginal = errors.New("original error")
)

// Mock HTTP client for testing helpers.
type mockHTTPClient struct {
	response *Response
	err      error
	delay    time.Duration
}

//nolint:gocritic // Request struct size is acceptable for this usage
func (m *mockHTTPClient) Execute(ctx context.Context, req Request) (*Response, error) {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
		}
	}

	return m.response, m.err
}

func TestTryExecute(t *testing.T) {
	tests := []struct {
		name     string
		response *Response
		err      error
	}{
		{
			name: "successful execution",
			response: &Response{
				StatusCode: 200,
				Raw:        []byte("success"),
			},
			err: nil,
		},
		{
			name:     "execution error",
			response: nil,
			err:      errNetwork,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockHTTPClient{
				response: tt.response,
				err:      tt.err,
			}

			req := Request{Method: "GET", Path: "/test"}
			resp, err := TryExecute(client, req)

			if resp != tt.response {
				t.Errorf("Expected response %v, got %v", tt.response, resp)
			}
			if !errors.Is(err, tt.err) {
				t.Errorf("Expected error %v, got %v", tt.err, err)
			}
		})
	}
}

func TestTryExecuteWithContext(t *testing.T) {
	tests := []struct {
		name     string
		response *Response
		err      error
		delay    time.Duration
		timeout  time.Duration
	}{
		{
			name: "successful execution with context",
			response: &Response{
				StatusCode: 200,
				Raw:        []byte("success"),
			},
			err:     nil,
			delay:   0,
			timeout: time.Second,
		},
		{
			name:     "context timeout",
			response: nil,
			err:      context.DeadlineExceeded,
			delay:    10 * time.Millisecond,
			timeout:  1 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockHTTPClient{
				response: tt.response,
				err:      tt.err,
				delay:    tt.delay,
			}

			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			req := Request{Method: "GET", Path: "/test"}
			resp, err := TryExecuteWithContext(ctx, client, req)

			if tt.name == "context timeout" {
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Errorf("Expected context.DeadlineExceeded, got %v", err)
				}
			} else {
				if resp != tt.response {
					t.Errorf("Expected response %v, got %v", tt.response, resp)
				}
				if !errors.Is(err, tt.err) {
					t.Errorf("Expected error %v, got %v", tt.err, err)
				}
			}
		})
	}
}

func TestTryExecuteWithTimeout(t *testing.T) {
	tests := []struct {
		name     string
		response *Response
		err      error
		delay    time.Duration
		timeout  time.Duration
	}{
		{
			name: "successful execution within timeout",
			response: &Response{
				StatusCode: 200,
				Raw:        []byte("success"),
			},
			err:     nil,
			delay:   1 * time.Millisecond,
			timeout: 10 * time.Millisecond,
		},
		{
			name:     "execution exceeds timeout",
			response: nil,
			err:      context.DeadlineExceeded,
			delay:    10 * time.Millisecond,
			timeout:  1 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockHTTPClient{
				response: tt.response,
				err:      tt.err,
				delay:    tt.delay,
			}

			req := Request{Method: "GET", Path: "/test"}
			resp, err := TryExecuteWithTimeout(client, req, tt.timeout)

			if tt.name == "execution exceeds timeout" {
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Errorf("Expected context.DeadlineExceeded, got %v", err)
				}
			} else {
				if resp != tt.response {
					t.Errorf("Expected response %v, got %v", tt.response, resp)
				}
				if !errors.Is(err, tt.err) {
					t.Errorf("Expected error %v, got %v", tt.err, err)
				}
			}
		})
	}
}

func TestTimeoutConstants(t *testing.T) {
	// Test that timeout constants have reasonable values
	if DefaultTimeout != 30*time.Second {
		t.Errorf("Expected DefaultTimeout to be 30s, got %v", DefaultTimeout)
	}
	if ShortTimeout != 5*time.Second {
		t.Errorf("Expected ShortTimeout to be 5s, got %v", ShortTimeout)
	}
	if LongTimeout != 60*time.Second {
		t.Errorf("Expected LongTimeout to be 60s, got %v", LongTimeout)
	}
}

func TestErrorTypes(t *testing.T) {
	t.Run("IsRequestError", func(t *testing.T) {
		reqErr := &RequestError{
			Method: "GET",
			Path:   "/test",
			Err:    errTest,
		}

		if !IsRequestError(reqErr) {
			t.Error("Should identify RequestError")
		}

		regularErr := errRegular
		if IsRequestError(regularErr) {
			t.Error("Should not identify regular error as RequestError")
		}
	})

	t.Run("IsValidationError", func(t *testing.T) {
		valErr := &ValidationError{
			Field:   "email",
			Message: "required",
		}

		if !IsValidationError(valErr) {
			t.Error("Should identify ValidationError")
		}

		regularErr := errRegular
		if IsValidationError(regularErr) {
			t.Error("Should not identify regular error as ValidationError")
		}
	})
}

func TestRequestErrorMethods(t *testing.T) {
	originalErr := errOriginal
	reqErr := &RequestError{
		Method: "POST",
		Path:   "/users",
		Err:    originalErr,
	}

	errorMsg := reqErr.Error()
	expectedMsg := "request POST /users failed: original error"
	if errorMsg != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, errorMsg)
	}

	unwrappedErr := reqErr.Unwrap()
	if !errors.Is(unwrappedErr, errOriginal) {
		t.Errorf("Expected unwrapped error %v, got %v", errOriginal, unwrappedErr)
	}
}

func TestValidationErrorMethods(t *testing.T) {
	valErr := &ValidationError{
		Field:   "email",
		Message: "must be a valid email",
	}

	errorMsg := valErr.Error()
	expectedMsg := "validation failed for field \"email\": must be a valid email"
	if errorMsg != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, errorMsg)
	}

	// ValidationError doesn't wrap another error
	unwrappedErr := valErr.Unwrap()
	if unwrappedErr != nil {
		t.Errorf("Expected no unwrapped error, got %v", unwrappedErr)
	}
}

// Test the helper functions that require testing.T by creating subtests.
// We can't easily mock testing.T but we can test the functions work.
func TestHelperFunctions(t *testing.T) {
	client := &mockHTTPClient{
		response: &Response{StatusCode: 200, Raw: []byte("success")},
		err:      nil,
	}

	req := Request{Method: "GET", Path: "/test"}

	t.Run("MustExecute with success", func(t *testing.T) {
		resp := MustExecute(t, client, req)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("ExecuteWithShortTimeout with success", func(t *testing.T) {
		resp := ExecuteWithShortTimeout(t, client, req)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("ExecuteWithLongTimeout with success", func(t *testing.T) {
		resp := ExecuteWithLongTimeout(t, client, req)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})
}

func TestHelperFunctionsWithErrorResponse(t *testing.T) {
	client := &mockHTTPClient{
		response: &Response{StatusCode: http.StatusBadRequest, Raw: []byte("bad request")},
		err:      nil,
	}

	req := Request{Method: "POST", Path: "/invalid"}

	t.Run("ExecuteExpectingError with error response", func(t *testing.T) {
		resp, err := ExecuteExpectingError(t, client, req)
		if err != nil {
			t.Errorf("Should not return execution error: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("ExecuteExpectingErrorWithContext with error response", func(t *testing.T) {
		ctx := context.Background()
		resp, err := ExecuteExpectingErrorWithContext(t, ctx, client, req)
		if err != nil {
			t.Errorf("Should not return execution error: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})
}

func TestHelperFunctionsWithNetworkError(t *testing.T) {
	client := &mockHTTPClient{
		response: nil,
		err:      errNetwork,
	}

	req := Request{Method: "GET", Path: "/test"}

	t.Run("ExecuteExpectingError with network error", func(t *testing.T) {
		resp, err := ExecuteExpectingError(t, client, req)
		if err == nil {
			t.Error("Expected network error to be returned")
		}
		if resp != nil {
			t.Error("Expected nil response on network error")
		}
	})

	t.Run("ExecuteExpectingErrorWithContext with network error", func(t *testing.T) {
		ctx := context.Background()
		resp, err := ExecuteExpectingErrorWithContext(t, ctx, client, req)
		if err == nil {
			t.Error("Expected network error to be returned")
		}
		if resp != nil {
			t.Error("Expected nil response on network error")
		}
	})
}
