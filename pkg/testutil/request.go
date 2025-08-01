package testutil

// Helper functions for common request patterns following Go's preference
// for explicit, readable code over method chaining.

// GET creates a GET request with the specified path.
func GET(path string) Request {
	return Request{Method: "GET", Path: path}
}

// POST creates a POST request with the specified path and body.
func POST(path string, body interface{}) Request {
	return Request{Method: "POST", Path: path, Body: body}
}

// PUT creates a PUT request with the specified path and body.
func PUT(path string, body interface{}) Request {
	return Request{Method: "PUT", Path: path, Body: body}
}

// PATCH creates a PATCH request with the specified path and body.
func PATCH(path string, body interface{}) Request {
	return Request{Method: "PATCH", Path: path, Body: body}
}

// DELETE creates a DELETE request with the specified path.
func DELETE(path string) Request {
	return Request{Method: "DELETE", Path: path}
}

// HEAD creates a HEAD request with the specified path.
func HEAD(path string) Request {
	return Request{Method: "HEAD", Path: path}
}

// OPTIONS creates an OPTIONS request with the specified path.
func OPTIONS(path string) Request {
	return Request{Method: "OPTIONS", Path: path}
}

// Request modifiers that return new Request instances (functional approach)

// WithAuth adds Bearer token authentication to the request.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func WithAuth(req Request, token string) Request {
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	req.Headers["Authorization"] = "Bearer " + token

	return req
}

// WithBasicAuth adds Basic authentication to the request.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func WithBasicAuth(req Request, username, password string) Request {
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	// In a real implementation, this would properly encode the credentials
	req.Headers["Authorization"] = "Basic " + username + ":" + password

	return req
}

// WithHeaders adds headers to the request.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func WithHeaders(req Request, headers map[string]string) Request {
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	for k, v := range headers {
		req.Headers[k] = v
	}

	return req
}

// WithHeader adds a single header to the request.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func WithHeader(req Request, key, value string) Request {
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	req.Headers[key] = value

	return req
}

// WithContentType sets the Content-Type header.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func WithContentType(req Request, contentType string) Request {
	return WithHeader(req, "Content-Type", contentType)
}

// WithJSON sets the Content-Type to application/json.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func WithJSON(req Request) Request {
	return WithContentType(req, "application/json")
}

// WithPathParams sets path parameters for the request.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func WithPathParams(req Request, params map[string]string) Request {
	req.PathParams = params

	return req
}

// WithPathParam sets a single path parameter.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func WithPathParam(req Request, key, value string) Request {
	if req.PathParams == nil {
		req.PathParams = make(map[string]string)
	}
	req.PathParams[key] = value

	return req
}

// WithQueryParams sets query parameters for the request.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func WithQueryParams(req Request, params map[string]string) Request {
	req.QueryParams = params

	return req
}

// WithQueryParam sets a single query parameter.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func WithQueryParam(req Request, key, value string) Request {
	if req.QueryParams == nil {
		req.QueryParams = make(map[string]string)
	}
	req.QueryParams[key] = value

	return req
}

// WithCookies sets cookies for the request.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func WithCookies(req Request, cookies map[string]string) Request {
	req.Cookies = cookies

	return req
}

// WithCookie sets a single cookie.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func WithCookie(req Request, name, value string) Request {
	if req.Cookies == nil {
		req.Cookies = make(map[string]string)
	}
	req.Cookies[name] = value

	return req
}

// WithFiles sets file uploads for the request.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func WithFiles(req Request, files map[string][]byte) Request {
	req.Files = files

	return req
}

// WithFile sets a single file upload.
//
//nolint:gocritic // Request struct size is acceptable for this usage
func WithFile(req Request, fieldName string, content []byte) Request {
	if req.Files == nil {
		req.Files = make(map[string][]byte)
	}
	req.Files[fieldName] = content

	return req
}
