package testutil

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// Test helper functions following Go testing conventions with proper t.Helper() usage

// MustExecute executes a request and fails the test on error.
// Uses context.Background() as default - for custom contexts use ExecuteWithContext.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func MustExecute(t *testing.T, client HTTPClient, req Request) *Response {
	t.Helper()
	ctx := context.Background()
	resp, err := client.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Request %s %s failed: %v", req.Method, req.Path, err)
	}

	return resp
}

// ExecuteExpectingError executes a request expecting an error response (4xx/5xx).
// Returns the response and any execution error. Tests should check both.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func ExecuteExpectingError(t *testing.T, client HTTPClient, req Request) (*Response, error) {
	t.Helper()
	ctx := context.Background()
	resp, err := client.Execute(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("execute request failed: %w", err)
	}
	if resp.StatusCode < ClientErrorThreshold {
		t.Fatalf("Expected error status code (>=400), got %d for %s %s",
			resp.StatusCode, req.Method, req.Path)
	}

	return resp, nil
}

// ExecuteExpectingErrorWithContext executes a request with context expecting an error response.
func ExecuteExpectingErrorWithContext(
	t *testing.T,
	ctx context.Context,
	client HTTPClient,
	req Request, //nolint:gocritic
) (*Response, error) {
	t.Helper()
	resp, err := client.Execute(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("execute request failed: %w", err)
	}
	if resp.StatusCode < ClientErrorThreshold {
		t.Fatalf("Expected error status code (>=400), got %d for %s %s",
			resp.StatusCode, req.Method, req.Path)
	}

	return resp, nil
}

// TryExecute executes a request and returns both response and error without failing the test.
// Useful when you want to handle errors in test logic rather than failing immediately.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func TryExecute(client HTTPClient, req Request) (*Response, error) {
	ctx := context.Background()

	resp, err := client.Execute(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("execute request failed: %w", err)
	}

	return resp, nil
}

// TryExecuteWithContext executes request with context without failing the test.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func TryExecuteWithContext(ctx context.Context, client HTTPClient, req Request) (*Response, error) {
	resp, err := client.Execute(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("execute request failed: %w", err)
	}

	return resp, nil
}

// TryExecuteWithTimeout executes request with timeout without failing the test.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func TryExecuteWithTimeout(client HTTPClient, req Request, timeout time.Duration) (*Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resp, err := client.Execute(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("execute request failed: %w", err)
	}

	return resp, nil
}

// Common timeout values for convenience.
const (
	DefaultTimeout = 30 * time.Second
	ShortTimeout   = 5 * time.Second
	LongTimeout    = 60 * time.Second
)

// HTTP status code thresholds.
const (
	// ClientErrorThreshold defines the minimum HTTP status code considered a client error (4xx).
	ClientErrorThreshold = 400
)

// ExecuteWithShortTimeout is a convenience function for short timeout requests.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func ExecuteWithShortTimeout(t *testing.T, client HTTPClient, req Request) *Response {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), ShortTimeout)
	defer cancel()

	resp, err := client.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Request %s %s with short timeout failed: %v", req.Method, req.Path, err)
	}

	return resp
}

// ExecuteWithLongTimeout is a convenience function for long timeout requests.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func ExecuteWithLongTimeout(t *testing.T, client HTTPClient, req Request) *Response {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), LongTimeout)
	defer cancel()

	resp, err := client.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Request %s %s with long timeout failed: %v", req.Method, req.Path, err)
	}

	return resp
}
