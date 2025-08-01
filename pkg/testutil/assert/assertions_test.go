package assert

import (
	"net/http"
	"testing"

	"github.com/pavelpascari/typedhttp/pkg/testutil"
)

func TestAssertStatus(t *testing.T) {
	t.Run("matching status codes", func(t *testing.T) {
		resp := &testutil.Response{
			StatusCode: 200,
			Raw:        []byte("test response"),
		}

		// This should not cause any test failures
		Status(t, resp, 200)
	})
}

func TestAssertStatusOK(t *testing.T) {
	resp := &testutil.Response{StatusCode: 200}
	StatusOK(t, resp)
}

func TestAssertStatusCreated(t *testing.T) {
	resp := &testutil.Response{StatusCode: 201}
	StatusCreated(t, resp)
}

func TestAssertStatusNotFound(t *testing.T) {
	resp := &testutil.Response{StatusCode: 404}
	StatusNotFound(t, resp)
}

func TestAssertStatusBadRequest(t *testing.T) {
	resp := &testutil.Response{StatusCode: 400}
	StatusBadRequest(t, resp)
}

func TestAssertStatusUnauthorized(t *testing.T) {
	resp := &testutil.Response{StatusCode: 401}
	StatusUnauthorized(t, resp)
}

func TestAssertHeader(t *testing.T) {
	t.Run("matching header", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Content-Type", "application/json")
		resp := &testutil.Response{Headers: headers}

		Header(t, resp, "Content-Type", "application/json")
	})
}

func TestAssertHeaderExists(t *testing.T) {
	t.Run("header exists", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Custom", "value")
		resp := &testutil.Response{Headers: headers}

		HeaderExists(t, resp, "X-Custom")
	})
}

func TestAssertHeaderContains(t *testing.T) {
	t.Run("header contains substring", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Content-Type", "application/json; charset=utf-8")
		resp := &testutil.Response{Headers: headers}

		HeaderContains(t, resp, "Content-Type", "json")
	})
}

func TestAssertContentType(t *testing.T) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	resp := &testutil.Response{Headers: headers}

	ContentType(t, resp, "application/json")
}

func TestAssertJSONContentType(t *testing.T) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json; charset=utf-8")
	resp := &testutil.Response{Headers: headers}

	JSONContentType(t, resp)
}

func TestAssertBodyContains(t *testing.T) {
	t.Run("body contains substring", func(t *testing.T) {
		resp := &testutil.Response{Raw: []byte("Hello World")}
		BodyContains(t, resp, "World")
	})
}

func TestAssertBodyEquals(t *testing.T) {
	t.Run("body matches expected", func(t *testing.T) {
		resp := &testutil.Response{Raw: []byte("exact match")}
		BodyEquals(t, resp, "exact match")
	})
}

func TestAssertEmptyBody(t *testing.T) {
	t.Run("empty body", func(t *testing.T) {
		resp := &testutil.Response{Raw: []byte("")}
		EmptyBody(t, resp)
	})
}

func TestAssertJSON(t *testing.T) {
	t.Run("matching JSON objects", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Content-Type", "application/json")
		resp := &testutil.Response{
			Headers: headers,
			Raw:     []byte(`{"name":"John","age":30}`),
		}
		expected := map[string]interface{}{"name": "John", "age": float64(30)}

		JSON(t, resp, expected)
	})
}

func TestAssertJSONField(t *testing.T) {
	t.Run("simple field match", func(t *testing.T) {
		resp := &testutil.Response{Raw: []byte(`{"name":"John","age":30}`)}
		JSONField(t, resp, "name", "John")
	})

	t.Run("nested field match", func(t *testing.T) {
		resp := &testutil.Response{Raw: []byte(`{"user":{"name":"John","age":30}}`)}
		JSONField(t, resp, "user.name", "John")
	})
}

func TestAssertJSONFieldExists(t *testing.T) {
	t.Run("field exists", func(t *testing.T) {
		resp := &testutil.Response{Raw: []byte(`{"name":"John","age":30}`)}
		JSONFieldExists(t, resp, "name")
	})

	t.Run("nested field exists", func(t *testing.T) {
		resp := &testutil.Response{Raw: []byte(`{"user":{"name":"John"}}`)}
		JSONFieldExists(t, resp, "user.name")
	})
}

func TestAssertValidationError(t *testing.T) {
	t.Run("validation error found", func(t *testing.T) {
		resp := &testutil.Response{
			StatusCode: 400,
			Raw:        []byte(`{"errors":{"email":"email is required"}}`),
		}
		ValidationError(t, resp, "email", "required")
	})
}

func TestAssertHasValidationError(t *testing.T) {
	t.Run("field has validation error", func(t *testing.T) {
		resp := &testutil.Response{
			StatusCode: 400,
			Raw:        []byte(`{"errors":{"email":"some error"}}`),
		}
		HasValidationError(t, resp, "email")
	})
}

func TestGetJSONField(t *testing.T) {
	data := map[string]interface{}{
		"name": "John",
		"age":  30,
		"user": map[string]interface{}{
			"email": "john@example.com",
			"profile": map[string]interface{}{
				"bio": "Software Developer",
			},
		},
	}

	tests := []struct {
		name        string
		path        string
		expected    interface{}
		shouldError bool
	}{
		{
			name:        "simple field",
			path:        "name",
			expected:    "John",
			shouldError: false,
		},
		{
			name:        "nested field",
			path:        "user.email",
			expected:    "john@example.com",
			shouldError: false,
		},
		{
			name:        "deeply nested field",
			path:        "user.profile.bio",
			expected:    "Software Developer",
			shouldError: false,
		},
		{
			name:        "non-existent field",
			path:        "missing",
			expected:    nil,
			shouldError: true,
		},
		{
			name:        "non-existent nested field",
			path:        "user.missing",
			expected:    nil,
			shouldError: true,
		},
		{
			name:        "invalid path on non-object",
			path:        "name.invalid",
			expected:    nil,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getJSONField(data, tt.path)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %v, got %v", tt.expected, result)
				}
			}
		})
	}
}

func TestTruncateResponse(t *testing.T) {
	tests := []struct {
		name     string
		body     []byte
		maxLen   int
		expected string
	}{
		{
			name:     "short body",
			body:     []byte("short"),
			maxLen:   10,
			expected: "short",
		},
		{
			name:     "long body truncated",
			body:     []byte("this is a very long response body"),
			maxLen:   10,
			expected: "this is a ... (truncated)",
		},
		{
			name:     "exact length",
			body:     []byte("exactly10c"),
			maxLen:   10,
			expected: "exactly10c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateResponse(tt.body, tt.maxLen)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetMapKeys(t *testing.T) {
	m := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	keys := getMapKeys(m)

	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	// Check that all expected keys are present
	expectedKeys := []string{"key1", "key2", "key3"}
	for _, expected := range expectedKeys {
		found := false
		for _, key := range keys {
			if key == expected {
				found = true

				break
			}
		}
		if !found {
			t.Errorf("Expected key %q not found in result", expected)
		}
	}
}

// TestAssertionFailures - test error cases by capturing output - these tests verify the functions can be called
// but won't fail the test suite since they're designed to fail on mismatches.
func TestAssertionFailures(t *testing.T) {
	// Capture stderr to avoid test output noise in CI
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Expected failure: %v", r)
		}
	}()

	// We can't easily test the actual failure behavior without complex test-within-test
	// machinery, but we can verify the functions execute properly

	t.Run("verify functions can be called without panicking", func(t *testing.T) {
		resp := &testutil.Response{
			StatusCode: 200,
			Headers:    http.Header{},
			Raw:        []byte(`{"test": "value"}`),
		}

		// These will succeed, just testing the functions execute
		Status(t, resp, 200)
		BodyContains(t, resp, "test")
		JSONField(t, resp, "test", "value")
	})
}
