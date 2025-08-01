// Package client provides context-aware HTTP client implementation for testing TypedHTTP handlers.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"

	"github.com/pavelpascari/typedhttp/pkg/testutil"
	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
)

// Client implements HTTPClient with full context support and proper error handling.
type Client struct {
	router  *typedhttp.TypedRouter
	baseURL string
	timeout time.Duration
}

// Option configures a Client using the functional options pattern.
type Option func(*Client)

// WithTimeout sets the default timeout for requests.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// WithBaseURL sets the base URL for requests (useful for integration tests).
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// NewClient creates a new context-aware HTTP client for testing.
func NewClient(router *typedhttp.TypedRouter, opts ...Option) *Client {
	client := &Client{
		router:  router,
		timeout: testutil.DefaultTimeout,
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// Execute performs an HTTP request with explicit error handling and context support.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func (c *Client) Execute(ctx context.Context, req testutil.Request) (*testutil.Response, error) {
	// Add timeout if context doesn't have deadline
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	httpReq, err := c.buildHTTPRequest(ctx, req)
	if err != nil {
		return nil, &testutil.RequestError{
			Method: req.Method,
			Path:   req.Path,
			Err:    fmt.Errorf("building HTTP request: %w", err),
		}
	}

	resp, err := c.executeHTTPRequest(httpReq)
	if err != nil {
		return nil, &testutil.RequestError{
			Method: req.Method,
			Path:   req.Path,
			Err:    fmt.Errorf("executing HTTP request: %w", err),
		}
	}

	return resp, nil
}

// ExecuteTyped performs a typed HTTP request with JSON unmarshaling.
// Note: This is a generic function, not a method, due to Go's current limitations.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func ExecuteTyped[T any](c *Client, ctx context.Context, req testutil.Request) (*testutil.TypedResponse[T], error) {
	resp, err := c.Execute(ctx, req)
	if err != nil {
		return nil, err
	}

	var data T
	if len(resp.Raw) > 0 && strings.Contains(resp.Headers.Get("Content-Type"), "application/json") {
		if err := json.Unmarshal(resp.Raw, &data); err != nil {
			return nil, &testutil.RequestError{
				Method: req.Method,
				Path:   req.Path,
				Err:    fmt.Errorf("unmarshaling JSON response: %w", err),
			}
		}
	}

	return &testutil.TypedResponse[T]{
		Response: resp,
		Data:     data,
	}, nil
}

// buildHTTPRequest constructs an *http.Request from testutil.Request.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func (c *Client) buildHTTPRequest(ctx context.Context, req testutil.Request) (*http.Request, error) {
	path := c.buildRequestPath(req)
	body, contentType, err := c.buildRequestBody(req)
	if err != nil {
		return nil, err
	}

	fullURL := c.baseURL + path
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request: %w", err)
	}

	c.setRequestHeaders(httpReq, req.Headers, contentType)
	c.setRequestCookies(httpReq, req.Cookies)

	return httpReq, nil
}

// buildRequestPath constructs the full path with parameters and query string.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func (c *Client) buildRequestPath(req testutil.Request) string {
	// Replace path parameters
	path := req.Path
	for key, value := range req.PathParams {
		placeholder := "{" + key + "}"
		path = strings.ReplaceAll(path, placeholder, value)
	}

	// Add query parameters
	if len(req.QueryParams) > 0 {
		values := url.Values{}
		for key, value := range req.QueryParams {
			values.Set(key, value)
		}
		path += "?" + values.Encode()
	}

	return path
}

// buildRequestBody constructs the request body based on content type.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func (c *Client) buildRequestBody(req testutil.Request) (io.Reader, string, error) {
	if req.Body == nil && len(req.Files) == 0 {
		return nil, "", nil
	}

	if len(req.Files) > 0 {
		return c.buildMultipartBody(req)
	}

	return c.buildJSONBody(req)
}

// buildMultipartBody creates a multipart form data body.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func (c *Client) buildMultipartBody(req testutil.Request) (io.Reader, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add form fields from body if it's a map
	if bodyMap, ok := req.Body.(map[string]string); ok {
		for key, value := range bodyMap {
			if err := writer.WriteField(key, value); err != nil {
				return nil, "", fmt.Errorf("writing form field %s: %w", key, err)
			}
		}
	}

	// Add files
	for fieldName, content := range req.Files {
		part, err := writer.CreateFormFile(fieldName, fieldName)
		if err != nil {
			return nil, "", fmt.Errorf("creating form file %s: %w", fieldName, err)
		}
		if _, err := part.Write(content); err != nil {
			return nil, "", fmt.Errorf("writing file content for %s: %w", fieldName, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("closing multipart writer: %w", err)
	}

	return &buf, writer.FormDataContentType(), nil
}

// buildJSONBody creates a JSON request body.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func (c *Client) buildJSONBody(req testutil.Request) (io.Reader, string, error) {
	jsonData, err := json.Marshal(req.Body)
	if err != nil {
		return nil, "", fmt.Errorf("marshaling request body to JSON: %w", err)
	}

	return bytes.NewReader(jsonData), "application/json", nil
}

// setRequestHeaders sets headers on the HTTP request.
func (c *Client) setRequestHeaders(httpReq *http.Request, headers map[string]string, contentType string) {
	// Set content type if we have a body
	if contentType != "" {
		httpReq.Header.Set("Content-Type", contentType)
	}

	// Add custom headers
	for key, value := range headers {
		httpReq.Header.Set(key, value)
	}
}

// setRequestCookies sets cookies on the HTTP request.
func (c *Client) setRequestCookies(httpReq *http.Request, cookies map[string]string) {
	for name, value := range cookies {
		httpReq.AddCookie(&http.Cookie{
			Name:  name,
			Value: value,
		})
	}
}

// executeHTTPRequest executes the HTTP request using httptest.ResponseRecorder for testing.
func (c *Client) executeHTTPRequest(req *http.Request) (*testutil.Response, error) {
	recorder := httptest.NewRecorder()

	// Execute request through the TypedHTTP router
	c.router.ServeHTTP(recorder, req)

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

// Convenience methods that use default context

// ExecuteWithTimeout executes request with timeout using default context.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func (c *Client) ExecuteWithTimeout(req testutil.Request, timeout time.Duration) (*testutil.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return c.Execute(ctx, req)
}

// ExecuteTypedWithTimeout executes typed request with timeout using default context.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func ExecuteTypedWithTimeout[T any](
	client *Client,
	req testutil.Request,
	timeout time.Duration,
) (*testutil.TypedResponse[T], error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return ExecuteTyped[T](client, ctx, req)
}
