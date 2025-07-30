package typedhttp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCookieDecoderMethods tests cookie decoder Decode method.
func TestCookieDecoderMethods(t *testing.T) {
	type CookieTestRequest struct {
		SessionID string `cookie:"session_id"`
		UserID    string `cookie:"user_id"`
	}

	decoder := NewCookieDecoder[CookieTestRequest](nil)

	// Test with cookies
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "abc123"})
	req.AddCookie(&http.Cookie{Name: "user_id", Value: "user456"})

	result, err := decoder.Decode(req)
	require.NoError(t, err)
	assert.Equal(t, "abc123", result.SessionID)
	assert.Equal(t, "user456", result.UserID)
}

// TestCookieHelperFunctions tests cookie helper functions.
func TestCookieHelperFunctions(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.AddCookie(&http.Cookie{Name: "test", Value: "value"})
	req.AddCookie(&http.Cookie{Name: "multi", Value: "value1"})
	req.AddCookie(&http.Cookie{Name: "multi", Value: "value2"})

	// Test GetAllCookies
	cookies := GetAllCookies(req)
	assert.NotEmpty(t, cookies)
	assert.Contains(t, cookies, "test")

	// Test GetCookieWithDefault
	value := GetCookieWithDefault(req, "test", "default")
	assert.Equal(t, "value", value)

	value = GetCookieWithDefault(req, "nonexistent", "default")
	assert.Equal(t, "default", value)
}

// TestSecureCookieDecoder tests SecureCookieDecoder.
func TestSecureCookieDecoder(t *testing.T) {
	decoder := NewSecureCookieDecoder[ValidationTestRequest](nil, "secret-key")
	assert.NotNil(t, decoder)

	// Test ContentTypes
	contentTypes := decoder.ContentTypes()
	assert.Contains(t, contentTypes, "*/*")
}
