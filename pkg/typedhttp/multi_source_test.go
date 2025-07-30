package typedhttp

import (
	"bytes"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test struct for multi-source extraction
type MultiSourceRequest struct {
	// Single source fields
	ID       string `path:"id"`
	Name     string `query:"name"`
	Auth     string `header:"Authorization"`
	Session  string `cookie:"session"`
	Content  string `form:"content"`
	Metadata string `json:"metadata"`

	// Multi-source fields with precedence
	UserID   string `header:"X-User-ID" cookie:"user_id" precedence:"header,cookie"`
	TraceID  string `header:"X-Trace-ID" query:"trace_id" precedence:"header,query"`
	Language string `cookie:"lang" header:"Accept-Language" default:"en" precedence:"cookie,header"`

	// Fields with transformations and formats
	ClientIP net.IP `header:"X-Forwarded-For" transform:"first_ip"`
	IsAdmin  bool   `header:"X-User-Role" transform:"is_admin"`

	// Complex fields
	Page  int    `query:"page" default:"1" validate:"min=1"`
	Limit int    `query:"limit" default:"20" validate:"min=1,max=100"`
	Sort  string `query:"sort" default:"created_at"`

	// File upload
	Avatar *multipart.FileHeader `form:"avatar"`
}

// Simpler test struct for validation tests
type ValidationTestRequest struct {
	UserID string `header:"X-User-ID" validate:"required"`
}

// Simple test struct without validation for basic tests
type SimpleTestRequest struct {
	Auth     string `header:"Authorization"`
	UserID   string `header:"X-User-ID" cookie:"user_id" precedence:"header,cookie"`
	Session  string `cookie:"session"`
	Content  string `form:"content"`
	ClientIP net.IP `header:"X-Forwarded-For" transform:"first_ip"`
	IsAdmin  bool   `header:"X-User-Role" transform:"is_admin"`
}

func TestHeaderDecoder_Success(t *testing.T) {
	validator := validator.New()
	decoder := NewHeaderDecoder[SimpleTestRequest](validator)

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("X-User-ID", "user456")
	req.Header.Set("X-Forwarded-For", "192.168.1.100, 10.0.0.1")
	req.Header.Set("X-User-Role", "admin")

	result, err := decoder.Decode(req)
	require.NoError(t, err)

	assert.Equal(t, "Bearer token123", result.Auth)
	assert.Equal(t, "user456", result.UserID)
	assert.Equal(t, "192.168.1.100", result.ClientIP.String())
	assert.True(t, result.IsAdmin)
}

func TestCookieDecoder_Success(t *testing.T) {
	validator := validator.New()
	decoder := NewCookieDecoder[SimpleTestRequest](validator)

	req, _ := http.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "sess789"})
	req.AddCookie(&http.Cookie{Name: "user_id", Value: "cookie_user"})

	result, err := decoder.Decode(req)
	require.NoError(t, err)

	assert.Equal(t, "sess789", result.Session)
	assert.Equal(t, "cookie_user", result.UserID)
}

func TestFormDecoder_Success(t *testing.T) {
	validator := validator.New()
	decoder := NewFormDecoder[SimpleTestRequest](validator)

	// Create form data
	form := url.Values{}
	form.Add("content", "test content")

	req, _ := http.NewRequest("POST", "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	result, err := decoder.Decode(req)
	require.NoError(t, err)

	assert.Equal(t, "test content", result.Content)
}

func TestFormDecoder_MultipartWithFile(t *testing.T) {
	validator := validator.New()
	decoder := NewFormDecoder[MultiSourceRequest](validator)

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add form field
	writer.WriteField("content", "multipart content")

	// Add file
	fileWriter, err := writer.CreateFormFile("avatar", "test.jpg")
	require.NoError(t, err)
	fileWriter.Write([]byte("fake image data"))

	writer.Close()

	req, _ := http.NewRequest("POST", "/test", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	result, err := decoder.Decode(req)
	require.NoError(t, err)

	assert.Equal(t, "multipart content", result.Content)
	assert.NotNil(t, result.Avatar)
	assert.Equal(t, "test.jpg", result.Avatar.Filename)
}

func TestCombinedDecoder_MultiSource(t *testing.T) {
	validator := validator.New()
	decoder := NewCombinedDecoder[MultiSourceRequest](validator)

	// Create request with data from multiple sources
	req, _ := http.NewRequest("GET", "/users/123", nil)
	
	// Query parameters
	q := req.URL.Query()
	q.Add("name", "John Doe")
	q.Add("page", "2")
	q.Add("limit", "50")
	q.Add("trace_id", "query_trace")
	req.URL.RawQuery = q.Encode()
	
	// Headers
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("X-User-ID", "header_user")
	req.Header.Set("X-Trace-ID", "header_trace")
	req.Header.Set("X-Forwarded-For", "192.168.1.100, 10.0.0.1")
	req.Header.Set("Accept-Language", "fr")
	
	// Cookies
	req.AddCookie(&http.Cookie{Name: "session", Value: "sess789"})
	req.AddCookie(&http.Cookie{Name: "user_id", Value: "cookie_user"})
	req.AddCookie(&http.Cookie{Name: "lang", Value: "es"})

	result, err := decoder.Decode(req)
	require.NoError(t, err)

	// Test single source fields
	assert.Equal(t, "123", result.ID) // from path
	assert.Equal(t, "John Doe", result.Name) // from query
	assert.Equal(t, "Bearer token123", result.Auth) // from header
	assert.Equal(t, "sess789", result.Session) // from cookie

	// Test multi-source with precedence
	assert.Equal(t, "header_user", result.UserID) // header wins over cookie
	assert.Equal(t, "header_trace", result.TraceID) // header wins over query
	assert.Equal(t, "es", result.Language) // cookie wins over header

	// Test transformations
	assert.Equal(t, "192.168.1.100", result.ClientIP.String()) // first_ip transform

	// Test defaults and validation
	assert.Equal(t, 2, result.Page)
	assert.Equal(t, 50, result.Limit)
	assert.Equal(t, "created_at", result.Sort) // default value
}

func TestCombinedDecoder_Precedence(t *testing.T) {
	validator := validator.New()
	decoder := NewCombinedDecoder[MultiSourceRequest](validator)

	req, _ := http.NewRequest("GET", "/users/123", nil)
	
	// Add UserID to both header and cookie
	req.Header.Set("X-User-ID", "from_header")
	req.AddCookie(&http.Cookie{Name: "user_id", Value: "from_cookie"})
	
	// Add TraceID to both header and query (header should win)
	req.Header.Set("X-Trace-ID", "from_header")
	q := req.URL.Query()
	q.Add("trace_id", "from_query")
	req.URL.RawQuery = q.Encode()

	result, err := decoder.Decode(req)
	require.NoError(t, err)

	// Header should win for UserID (precedence: header,cookie)
	assert.Equal(t, "from_header", result.UserID)
	
	// Header should win for TraceID (precedence: header,query)
	assert.Equal(t, "from_header", result.TraceID)
}

func TestCombinedDecoder_Fallback(t *testing.T) {
	validator := validator.New()
	decoder := NewCombinedDecoder[MultiSourceRequest](validator)

	req, _ := http.NewRequest("GET", "/users/123", nil)
	
	// Only provide cookie for UserID (header missing)
	req.AddCookie(&http.Cookie{Name: "user_id", Value: "fallback_cookie"})
	
	// Only provide query for TraceID (header missing)
	q := req.URL.Query()
	q.Add("trace_id", "fallback_query")
	req.URL.RawQuery = q.Encode()

	result, err := decoder.Decode(req)
	require.NoError(t, err)

	// Should fallback to cookie for UserID
	assert.Equal(t, "fallback_cookie", result.UserID)
	
	// Should fallback to query for TraceID
	assert.Equal(t, "fallback_query", result.TraceID)
}

func TestCombinedDecoder_DefaultValues(t *testing.T) {
	validator := validator.New()
	decoder := NewCombinedDecoder[MultiSourceRequest](validator)

	req, _ := http.NewRequest("GET", "/users/123", nil)

	result, err := decoder.Decode(req)
	require.NoError(t, err)

	// Should use default values when no source provides value
	assert.Equal(t, "en", result.Language) // default
	assert.Equal(t, 1, result.Page) // default
	assert.Equal(t, 20, result.Limit) // default
	assert.Equal(t, "created_at", result.Sort) // default
}

func TestCombinedDecoder_Validation(t *testing.T) {
	validator := validator.New()
	decoder := NewCombinedDecoder[ValidationTestRequest](validator)

	req, _ := http.NewRequest("GET", "/users/123", nil)
	// Missing required UserID
	
	_, err := decoder.Decode(req)
	require.Error(t, err)
	
	validationErr, ok := err.(*ValidationError)
	require.True(t, ok)
	assert.Contains(t, validationErr.Fields, "userid")
}

func TestTransformations(t *testing.T) {
	tests := []struct {
		name      string
		transform string
		input     string
		expected  string
		wantErr   bool
	}{
		{"first_ip", "first_ip", "192.168.1.100, 10.0.0.1", "192.168.1.100", false},
		{"to_lower", "to_lower", "HELLO", "hello", false},
		{"to_upper", "to_upper", "hello", "HELLO", false},
		{"trim_space", "trim_space", "  hello  ", "hello", false},
		{"is_admin true", "is_admin", "admin", "true", false},
		{"is_admin false", "is_admin", "user", "false", false},
		{"unknown", "unknown_transform", "value", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := applyTransformation(tt.transform, tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestFormatParsing(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		input    string
		expected interface{}
		wantErr  bool
	}{
		{"unix timestamp", "unix", "1640995200", time.Unix(1640995200, 0), false},
		{"rfc3339", "rfc3339", "2022-01-01T00:00:00Z", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"date only", "2006-01-02", "2022-01-01", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"invalid unix", "unix", "invalid", time.Time{}, true},
		{"invalid rfc3339", "rfc3339", "invalid", time.Time{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := applyFormat(tt.format, tt.input, reflect.TypeOf(time.Time{}))
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestDefaultValueHandling(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"regular value", "test", "test"},
		{"now", "now", time.Now().Format(time.RFC3339)[:10]}, // Just check date part
		{"generate_uuid", "generate_uuid", "uuid-"},          // Just check prefix
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handleDefaultValue(tt.input)
			if tt.name == "now" {
				assert.Contains(t, result, tt.expected)
			} else if tt.name == "generate_uuid" {
				assert.Contains(t, result, tt.expected)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}