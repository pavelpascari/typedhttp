package testutil

import (
	"net/http"
	"testing"
)

const (
	usersPath   = "/users"
	usersIDPath = "/users/123"
	headerValue = "value"
)

func TestRequestHelpers(t *testing.T) {
	t.Run("GET creates correct request", func(t *testing.T) {
		req := GET(usersPath)

		if req.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", req.Method)
		}
		if req.Path != usersPath {
			t.Errorf("Expected path /users, got %s", req.Path)
		}
	})

	t.Run("POST creates correct request with body", func(t *testing.T) {
		body := map[string]string{"name": "John"}
		req := POST(usersPath, body)

		if req.Method != http.MethodPost {
			t.Errorf("Expected method POST, got %s", req.Method)
		}
		if req.Path != usersPath {
			t.Errorf("Expected path /users, got %s", req.Path)
		}
		if req.Body == nil {
			t.Errorf("Expected body to be set")
		}
	})

	t.Run("WithAuth adds authorization header", func(t *testing.T) {
		req := WithAuth(GET(usersPath), "token123")

		expected := "Bearer token123"
		if req.Headers["Authorization"] != expected {
			t.Errorf("Expected Authorization header %q, got %q",
				expected, req.Headers["Authorization"])
		}
	})

	t.Run("WithPathParam adds path parameter", func(t *testing.T) {
		req := WithPathParam(GET("/users/{id}"), "id", "123")

		if req.PathParams["id"] != "123" {
			t.Errorf("Expected PathParams[id] = 123, got %q", req.PathParams["id"])
		}
	})

	t.Run("WithQueryParam adds query parameter", func(t *testing.T) {
		req := WithQueryParam(GET(usersPath), "page", "1")

		if req.QueryParams["page"] != "1" {
			t.Errorf("Expected QueryParams[page] = 1, got %q", req.QueryParams["page"])
		}
	})

	t.Run("WithHeader adds custom header", func(t *testing.T) {
		req := WithHeader(GET(usersPath), "X-Custom", headerValue)

		if req.Headers["X-Custom"] != headerValue {
			t.Errorf("Expected Headers[X-Custom] = value, got %q", req.Headers["X-Custom"])
		}
	})

	t.Run("WithJSON sets content type", func(t *testing.T) {
		req := WithJSON(POST(usersPath, nil))

		expected := "application/json"
		if req.Headers["Content-Type"] != expected {
			t.Errorf("Expected Content-Type %q, got %q",
				expected, req.Headers["Content-Type"])
		}
	})

	t.Run("chaining request modifiers", func(t *testing.T) {
		req := WithAuth(
			WithPathParam(
				WithQueryParam(GET("/users/{id}"), "page", "1"),
				"id", "123",
			),
			"token456",
		)

		if req.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", req.Method)
		}
		if req.Path != "/users/{id}" {
			t.Errorf("Expected path /users/{id}, got %s", req.Path)
		}
		if req.PathParams["id"] != "123" {
			t.Errorf("Expected PathParams[id] = 123, got %q", req.PathParams["id"])
		}
		if req.QueryParams["page"] != "1" {
			t.Errorf("Expected QueryParams[page] = 1, got %q", req.QueryParams["page"])
		}
		if req.Headers["Authorization"] != "Bearer token456" {
			t.Errorf("Expected Authorization header Bearer token456, got %q",
				req.Headers["Authorization"])
		}
	})
}

func TestWithHeaders(t *testing.T) {
	t.Run("adds multiple headers", func(t *testing.T) {
		headers := map[string]string{
			"X-Custom-1": "value1",
			"X-Custom-2": "value2",
		}
		req := WithHeaders(GET("/test"), headers)

		for key, expected := range headers {
			if actual := req.Headers[key]; actual != expected {
				t.Errorf("Expected header %s = %q, got %q", key, expected, actual)
			}
		}
	})

	t.Run("preserves existing headers", func(t *testing.T) {
		req := WithHeader(GET("/test"), "Existing", headerValue)
		req = WithHeaders(req, map[string]string{"New": "newvalue"})

		if req.Headers["Existing"] != headerValue {
			t.Errorf("Expected existing header to be preserved")
		}
		if req.Headers["New"] != "newvalue" {
			t.Errorf("Expected new header to be added")
		}
	})
}

func TestWithCookies(t *testing.T) {
	t.Run("adds cookies", func(t *testing.T) {
		req := WithCookie(GET("/test"), "session", "abc123")

		if req.Cookies["session"] != "abc123" {
			t.Errorf("Expected cookie session = abc123, got %q", req.Cookies["session"])
		}
	})
}

func TestWithFiles(t *testing.T) {
	t.Run("adds file upload", func(t *testing.T) {
		content := []byte("file content")
		req := WithFile(POST("/upload", nil), "file", content)

		if string(req.Files["file"]) != "file content" {
			t.Errorf("Expected file content to be set")
		}
	})

	t.Run("adds multiple files", func(t *testing.T) {
		req := POST("/upload", nil)
		files := map[string][]byte{
			"document": []byte("document content"),
			"image":    []byte("image content"),
		}

		result := WithFiles(req, files)

		if len(result.Files) != 2 {
			t.Errorf("Expected 2 files, got %d", len(result.Files))
		}
		if string(result.Files["document"]) != "document content" {
			t.Errorf("Expected document content")
		}
		if string(result.Files["image"]) != "image content" {
			t.Errorf("Expected image content")
		}
	})
}

func TestHTTPMethods(t *testing.T) {
	t.Run("PUT method", func(t *testing.T) {
		body := map[string]string{"name": "John"}
		req := PUT(usersIDPath, body)

		if req.Method != http.MethodPut {
			t.Errorf("Expected method PUT, got %s", req.Method)
		}
		if req.Path != usersIDPath {
			t.Errorf("Expected path /users/123, got %s", req.Path)
		}
		if req.Body == nil {
			t.Errorf("Expected body to be set")
		}
	})

	t.Run("PATCH method", func(t *testing.T) {
		body := map[string]string{"status": "active"}
		req := PATCH(usersIDPath, body)

		if req.Method != http.MethodPatch {
			t.Errorf("Expected method PATCH, got %s", req.Method)
		}
		if req.Path != usersIDPath {
			t.Errorf("Expected path /users/123, got %s", req.Path)
		}
		if req.Body == nil {
			t.Errorf("Expected body to be set")
		}
	})

	t.Run("DELETE method", func(t *testing.T) {
		req := DELETE(usersIDPath)

		if req.Method != http.MethodDelete {
			t.Errorf("Expected method DELETE, got %s", req.Method)
		}
		if req.Path != usersIDPath {
			t.Errorf("Expected path /users/123, got %s", req.Path)
		}
		if req.Body != nil {
			t.Errorf("Expected no body, got %v", req.Body)
		}
	})

	t.Run("HEAD method", func(t *testing.T) {
		req := HEAD(usersIDPath)

		if req.Method != http.MethodHead {
			t.Errorf("Expected method HEAD, got %s", req.Method)
		}
		if req.Path != usersIDPath {
			t.Errorf("Expected path /users/123, got %s", req.Path)
		}
		if req.Body != nil {
			t.Errorf("Expected no body, got %v", req.Body)
		}
	})

	t.Run("OPTIONS method", func(t *testing.T) {
		req := OPTIONS(usersPath)

		if req.Method != http.MethodOptions {
			t.Errorf("Expected method OPTIONS, got %s", req.Method)
		}
		if req.Path != usersPath {
			t.Errorf("Expected path /users, got %s", req.Path)
		}
		if req.Body != nil {
			t.Errorf("Expected no body, got %v", req.Body)
		}
	})
}

func TestWithBasicAuth(t *testing.T) {
	t.Run("adds basic auth header", func(t *testing.T) {
		req := GET("/protected")
		result := WithBasicAuth(req, "admin", "secret")

		expected := "Basic admin:secret" // Note: This is a simplified implementation
		if result.Headers["Authorization"] != expected {
			t.Errorf("Expected Authorization header %s, got %s", expected, result.Headers["Authorization"])
		}
	})

	t.Run("preserves existing headers", func(t *testing.T) {
		req := WithHeader(GET("/test"), "X-Custom", headerValue)
		result := WithBasicAuth(req, "user", "pass")

		if result.Headers["X-Custom"] != headerValue {
			t.Error("Should preserve existing headers")
		}
		if result.Headers["Authorization"] == "" {
			t.Error("Should add Authorization header")
		}
	})
}

func TestWithCookiesMultiple(t *testing.T) {
	t.Run("adds multiple cookies", func(t *testing.T) {
		req := GET("/test")
		cookies := map[string]string{
			"session":    "abc123",
			"preference": "dark-mode",
			"language":   "en",
		}

		result := WithCookies(req, cookies)

		if len(result.Cookies) != 3 {
			t.Errorf("Expected 3 cookies, got %d", len(result.Cookies))
		}
		if result.Cookies["session"] != "abc123" {
			t.Error("Expected session cookie")
		}
		if result.Cookies["preference"] != "dark-mode" {
			t.Error("Expected preference cookie")
		}
		if result.Cookies["language"] != "en" {
			t.Error("Expected language cookie")
		}
	})

	t.Run("replaces existing cookies", func(t *testing.T) {
		req := WithCookie(GET("/test"), "existing", headerValue)
		cookies := map[string]string{
			"new": "cookie",
		}

		result := WithCookies(req, cookies)

		// WithCookies replaces the entire map
		if result.Cookies["existing"] != "" && result.Cookies["existing"] != headerValue {
			t.Error("Existing cookies should be replaced")
		}
		if result.Cookies["new"] != "cookie" {
			t.Error("Should add new cookies")
		}
		if len(result.Cookies) != 1 {
			t.Error("Should only have the new cookies")
		}
	})
}

func TestWithContentType(t *testing.T) {
	t.Run("sets content type header", func(t *testing.T) {
		req := POST("/upload", nil)
		result := WithContentType(req, "multipart/form-data")

		if result.Headers["Content-Type"] != "multipart/form-data" {
			t.Errorf("Expected Content-Type multipart/form-data, got %s", result.Headers["Content-Type"])
		}
	})

	t.Run("overwrites existing content type", func(t *testing.T) {
		req := WithHeader(POST("/test", nil), "Content-Type", "application/json")
		result := WithContentType(req, "text/plain")

		if result.Headers["Content-Type"] != "text/plain" {
			t.Errorf("Expected Content-Type text/plain, got %s", result.Headers["Content-Type"])
		}
	})
}

func TestWithQueryParamsMultiple(t *testing.T) {
	t.Run("adds multiple query parameters", func(t *testing.T) {
		req := GET(usersPath)
		params := map[string]string{
			"page":   "2",
			"limit":  "10",
			"sort":   "name",
			"filter": "active",
		}

		result := WithQueryParams(req, params)

		if len(result.QueryParams) != 4 {
			t.Errorf("Expected 4 query params, got %d", len(result.QueryParams))
		}
		if result.QueryParams["page"] != "2" {
			t.Error("Expected page parameter")
		}
		if result.QueryParams["limit"] != "10" {
			t.Error("Expected limit parameter")
		}
		if result.QueryParams["sort"] != "name" {
			t.Error("Expected sort parameter")
		}
		if result.QueryParams["filter"] != "active" {
			t.Error("Expected filter parameter")
		}
	})

	t.Run("replaces existing query parameters", func(t *testing.T) {
		req := WithQueryParam(GET("/test"), "existing", headerValue)
		params := map[string]string{
			"new": "param",
		}

		result := WithQueryParams(req, params)

		// WithQueryParams replaces the entire map
		if result.QueryParams["new"] != "param" {
			t.Error("Should add new query params")
		}
		if len(result.QueryParams) != 1 {
			t.Error("Should only have the new query params")
		}
	})
}

func TestWithPathParamsMultiple(t *testing.T) {
	t.Run("adds multiple path parameters", func(t *testing.T) {
		req := GET("/orgs/{orgId}/users/{userId}")
		params := map[string]string{
			"orgId":  "acme",
			"userId": "123",
		}

		result := WithPathParams(req, params)

		if len(result.PathParams) != 2 {
			t.Errorf("Expected 2 path params, got %d", len(result.PathParams))
		}
		if result.PathParams["orgId"] != "acme" {
			t.Error("Expected orgId parameter")
		}
		if result.PathParams["userId"] != "123" {
			t.Error("Expected userId parameter")
		}
	})

	t.Run("replaces existing path parameters", func(t *testing.T) {
		req := WithPathParam(GET("/test/{id}"), "id", "456")
		params := map[string]string{
			"version": "v1",
		}

		result := WithPathParams(req, params)

		// WithPathParams replaces the entire map
		if result.PathParams["version"] != "v1" {
			t.Error("Should add new path params")
		}
		if len(result.PathParams) != 1 {
			t.Error("Should only have the new path params")
		}
	})
}
