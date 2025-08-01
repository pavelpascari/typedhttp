// Package assert provides comprehensive assertion functions for HTTP testing
// with detailed error reporting and proper Go testing conventions.
package assert

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/pavelpascari/typedhttp/pkg/testutil"
)

const (
	// Response truncation lengths for different assertion types.
	shortTruncateLength  = 200
	mediumTruncateLength = 300
	longTruncateLength   = 500
)

var (
	errFieldNotFound = fmt.Errorf("field not found")
	errInvalidAccess = fmt.Errorf("cannot access field on non-object type")
)

// Status verifies the HTTP status code with detailed error reporting.
func Status(t *testing.T, resp *testutil.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		t.Errorf("Status code mismatch:\n  Expected: %d (%s)\n  Actual:   %d (%s)\n  Response: %s",
			expected, http.StatusText(expected),
			resp.StatusCode, http.StatusText(resp.StatusCode),
			truncateResponse(resp.Raw, shortTruncateLength))
	}
}

// StatusOK verifies the response has 200 OK status (common case shorthand).
func StatusOK(t *testing.T, resp *testutil.Response) {
	t.Helper()
	Status(t, resp, http.StatusOK)
}

// StatusCreated verifies the response has 201 Created status.
func StatusCreated(t *testing.T, resp *testutil.Response) {
	t.Helper()
	Status(t, resp, http.StatusCreated)
}

// StatusNotFound verifies the response has 404 Not Found status.
func StatusNotFound(t *testing.T, resp *testutil.Response) {
	t.Helper()
	Status(t, resp, http.StatusNotFound)
}

// StatusBadRequest verifies the response has 400 Bad Request status.
func StatusBadRequest(t *testing.T, resp *testutil.Response) {
	t.Helper()
	Status(t, resp, http.StatusBadRequest)
}

// StatusUnauthorized verifies the response has 401 Unauthorized status.
func StatusUnauthorized(t *testing.T, resp *testutil.Response) {
	t.Helper()
	Status(t, resp, http.StatusUnauthorized)
}

// Header verifies a response header value.
func Header(t *testing.T, resp *testutil.Response, key, expected string) {
	t.Helper()
	actual := resp.Headers.Get(key)
	if actual != expected {
		t.Errorf("Header %q mismatch:\n  Expected: %q\n  Actual:   %q",
			key, expected, actual)
	}
}

// HeaderExists verifies a response header exists.
func HeaderExists(t *testing.T, resp *testutil.Response, key string) {
	t.Helper()
	if resp.Headers.Get(key) == "" {
		t.Errorf("Expected header %q to exist, but it was not found", key)
	}
}

// HeaderContains verifies a response header contains a substring.
func HeaderContains(t *testing.T, resp *testutil.Response, key, substring string) {
	t.Helper()
	actual := resp.Headers.Get(key)
	if !strings.Contains(actual, substring) {
		t.Errorf("Header %q should contain %q:\n  Actual: %q",
			key, substring, actual)
	}
}

// ContentType verifies the Content-Type header.
func ContentType(t *testing.T, resp *testutil.Response, expected string) {
	t.Helper()
	Header(t, resp, "Content-Type", expected)
}

// JSONContentType verifies the response has JSON content type.
func JSONContentType(t *testing.T, resp *testutil.Response) {
	t.Helper()
	HeaderContains(t, resp, "Content-Type", "application/json")
}

// BodyContains verifies the response body contains a substring.
func BodyContains(t *testing.T, resp *testutil.Response, substring string) {
	t.Helper()
	body := string(resp.Raw)
	if !strings.Contains(body, substring) {
		t.Errorf("Response body should contain %q:\n  Body: %s",
			substring, truncateResponse(resp.Raw, longTruncateLength))
	}
}

// BodyEquals verifies the response body equals expected content.
func BodyEquals(t *testing.T, resp *testutil.Response, expected string) {
	t.Helper()
	actual := string(resp.Raw)
	if actual != expected {
		t.Errorf("Response body mismatch:\n  Expected: %q\n  Actual:   %q",
			expected, actual)
	}
}

// EmptyBody verifies the response body is empty.
func EmptyBody(t *testing.T, resp *testutil.Response) {
	t.Helper()
	if len(resp.Raw) > 0 {
		t.Errorf("Expected empty response body, got: %s",
			truncateResponse(resp.Raw, shortTruncateLength))
	}
}

// JSON verifies the response body matches expected JSON structure.
func JSON(t *testing.T, resp *testutil.Response, expected interface{}) {
	t.Helper()

	// Ensure response is JSON
	JSONContentType(t, resp)

	// Parse actual response
	var actual interface{}
	if err := json.Unmarshal(resp.Raw, &actual); err != nil {
		t.Fatalf("Failed to parse response as JSON: %v\nResponse: %s",
			err, truncateResponse(resp.Raw, longTruncateLength))
	}

	// Compare JSON structures
	expectedJSON, _ := json.MarshalIndent(expected, "", "  ")
	actualJSON, _ := json.MarshalIndent(actual, "", "  ")

	if !bytes.Equal(expectedJSON, actualJSON) {
		t.Errorf("JSON response mismatch:\n  Expected:\n%s\n  Actual:\n%s",
			string(expectedJSON), string(actualJSON))
	}
}

// JSONField verifies a specific field in JSON response using dot notation.
func JSONField(t *testing.T, resp *testutil.Response, fieldPath string, expected interface{}) {
	t.Helper()

	var data interface{}
	if err := json.Unmarshal(resp.Raw, &data); err != nil {
		t.Fatalf("Failed to parse response as JSON: %v", err)
	}

	actual, err := getJSONField(data, fieldPath)
	if err != nil {
		t.Fatalf("Failed to get field %q: %v", fieldPath, err)
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("JSON field %q mismatch:\n  Expected: %v (%T)\n  Actual:   %v (%T)",
			fieldPath, expected, expected, actual, actual)
	}
}

// JSONFieldExists verifies a specific field exists in JSON response.
func JSONFieldExists(t *testing.T, resp *testutil.Response, fieldPath string) {
	t.Helper()

	var data interface{}
	if err := json.Unmarshal(resp.Raw, &data); err != nil {
		t.Fatalf("Failed to parse response as JSON: %v", err)
	}

	_, err := getJSONField(data, fieldPath)
	if err != nil {
		t.Errorf("Expected JSON field %q to exist: %v", fieldPath, err)
	}
}

// ValidationError verifies validation error details in response.
func ValidationError(t *testing.T, resp *testutil.Response, field, expectedError string) {
	t.Helper()

	// First ensure it's a bad request
	StatusBadRequest(t, resp)

	// Parse error response
	var errorResp map[string]interface{}
	if err := json.Unmarshal(resp.Raw, &errorResp); err != nil {
		t.Fatalf("Failed to parse error response as JSON: %v", err)
	}

	// Look for validation errors in common error response formats
	if errors, ok := errorResp["errors"].(map[string]interface{}); ok {
		if fieldError, exists := errors[field]; exists {
			if !strings.Contains(fmt.Sprintf("%v", fieldError), expectedError) {
				t.Errorf("Validation error for field %q should contain %q, got: %v",
					field, expectedError, fieldError)
			}
		} else {
			t.Errorf("Expected validation error for field %q, but field not found in errors: %v",
				field, errors)
		}
	} else {
		t.Errorf("Expected validation errors in response, got: %s",
			truncateResponse(resp.Raw, mediumTruncateLength))
	}
}

// HasValidationError verifies that validation error exists for a field.
func HasValidationError(t *testing.T, resp *testutil.Response, field string) {
	t.Helper()

	StatusBadRequest(t, resp)

	var errorResp map[string]interface{}
	if err := json.Unmarshal(resp.Raw, &errorResp); err != nil {
		t.Fatalf("Failed to parse error response as JSON: %v", err)
	}

	if errors, ok := errorResp["errors"].(map[string]interface{}); ok {
		if _, exists := errors[field]; !exists {
			t.Errorf("Expected validation error for field %q, available fields: %v",
				field, getMapKeys(errors))
		}
	} else {
		t.Errorf("Expected validation errors in response, got: %s",
			truncateResponse(resp.Raw, mediumTruncateLength))
	}
}

// Helper functions

// getJSONField extracts a field from JSON data using dot notation (e.g., "user.name").
func getJSONField(data interface{}, path string) (interface{}, error) {
	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		switch value := current.(type) {
		case map[string]interface{}:
			if val, ok := value[part]; ok {
				current = val
			} else {
				return nil, fmt.Errorf("field %q: %w", part, errFieldNotFound)
			}
		default:
			return nil, fmt.Errorf("field %q on type %T: %w", part, value, errInvalidAccess)
		}
	}

	return current, nil
}

// truncateResponse truncates response body for error messages.
func truncateResponse(body []byte, maxLen int) string {
	if len(body) <= maxLen {
		return string(body)
	}

	return string(body[:maxLen]) + "... (truncated)"
}

// getMapKeys returns the keys of a map as a slice (for error reporting).
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	return keys
}
