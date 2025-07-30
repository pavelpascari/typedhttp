package typedhttp

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCookieDecoderBasicTypes tests all basic type conversions for cookies.
func TestCookieDecoderBasicTypes(t *testing.T) {
	type CookieRequest struct {
		StringField  string  `cookie:"str"`
		IntField     int     `cookie:"int"`
		Int8Field    int8    `cookie:"int8"`
		Int16Field   int16   `cookie:"int16"`
		Int32Field   int32   `cookie:"int32"`
		Int64Field   int64   `cookie:"int64"`
		UintField    uint    `cookie:"uint"`
		Uint8Field   uint8   `cookie:"uint8"`
		Uint16Field  uint16  `cookie:"uint16"`
		Uint32Field  uint32  `cookie:"uint32"`
		Uint64Field  uint64  `cookie:"uint64"`
		Float32Field float32 `cookie:"float32"`
		Float64Field float64 `cookie:"float64"`
		BoolField    bool    `cookie:"bool"`
	}

	decoder := NewCookieDecoder[CookieRequest](nil)

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.AddCookie(&http.Cookie{Name: "str", Value: "hello"})
	req.AddCookie(&http.Cookie{Name: "int", Value: "42"})
	req.AddCookie(&http.Cookie{Name: "int8", Value: "127"})
	req.AddCookie(&http.Cookie{Name: "int16", Value: "32767"})
	req.AddCookie(&http.Cookie{Name: "int32", Value: "2147483647"})
	req.AddCookie(&http.Cookie{Name: "int64", Value: "9223372036854775807"})
	req.AddCookie(&http.Cookie{Name: "uint", Value: "42"})
	req.AddCookie(&http.Cookie{Name: "uint8", Value: "255"})
	req.AddCookie(&http.Cookie{Name: "uint16", Value: "65535"})
	req.AddCookie(&http.Cookie{Name: "uint32", Value: "4294967295"})
	req.AddCookie(&http.Cookie{Name: "uint64", Value: "18446744073709551615"})
	req.AddCookie(&http.Cookie{Name: "float32", Value: "3.14"})
	req.AddCookie(&http.Cookie{Name: "float64", Value: "2.718281828"})
	req.AddCookie(&http.Cookie{Name: "bool", Value: "true"})

	result, err := decoder.Decode(req)
	require.NoError(t, err)

	assert.Equal(t, "hello", result.StringField)
	assert.Equal(t, 42, result.IntField)
	assert.Equal(t, int8(127), result.Int8Field)
	assert.Equal(t, int16(32767), result.Int16Field)
	assert.Equal(t, int32(2147483647), result.Int32Field)
	assert.Equal(t, int64(9223372036854775807), result.Int64Field)
	assert.Equal(t, uint(42), result.UintField)
	assert.Equal(t, uint8(255), result.Uint8Field)
	assert.Equal(t, uint16(65535), result.Uint16Field)
	assert.Equal(t, uint32(4294967295), result.Uint32Field)
	assert.Equal(t, uint64(18446744073709551615), result.Uint64Field)
	assert.Equal(t, float32(3.14), result.Float32Field)
	assert.Equal(t, 2.718281828, result.Float64Field)
	assert.True(t, result.BoolField)
}

// TestCookieDecoderTransformations tests all transformation functions for cookies.
func TestCookieDecoderTransformations(t *testing.T) {
	type CookieRequest struct {
		FirstIP   string `cookie:"forwarded_ips" transform:"first_ip"`
		LowerCase string `cookie:"upper_text" transform:"to_lower"`
		UpperCase string `cookie:"lower_text" transform:"to_upper"`
		Trimmed   string `cookie:"spaces" transform:"trim_space"`
		AdminFlag string `cookie:"user_role" transform:"is_admin"`
	}

	decoder := NewCookieDecoder[CookieRequest](nil)

	tests := []struct {
		name            string
		cookies         map[string]string
		expectedFirstIP string
		expectedLower   string
		expectedUpper   string
		expectedTrimmed string
		expectedAdmin   string
	}{
		{
			name: "basic transformations",
			cookies: map[string]string{
				"forwarded_ips": "192.168.1.1, 10.0.0.1, 172.16.0.1",
				"upper_text":    "HELLO WORLD",
				"lower_text":    "hello world",
				"spaces":        "  trimmed  ",
				"user_role":     "admin",
			},
			expectedFirstIP: "192.168.1.1",
			expectedLower:   "hello world",
			expectedUpper:   "HELLO WORLD",
			expectedTrimmed: "trimmed",
			expectedAdmin:   "true",
		},
		{
			name: "admin case insensitive",
			cookies: map[string]string{
				"forwarded_ips": "203.0.113.1",
				"upper_text":    "Test",
				"lower_text":    "TEST",
				"spaces":        "no-spaces",
				"user_role":     "ADMIN",
			},
			expectedFirstIP: "203.0.113.1",
			expectedLower:   "test",
			expectedUpper:   "TEST",
			expectedTrimmed: "no-spaces",
			expectedAdmin:   "true",
		},
		{
			name: "non-admin role",
			cookies: map[string]string{
				"forwarded_ips": "10.0.0.1, 192.168.1.1",
				"upper_text":    "mixed Case",
				"lower_text":    "Mixed Case",
				"spaces":        "\ttab-trimmed\t",
				"user_role":     "user",
			},
			expectedFirstIP: "10.0.0.1",
			expectedLower:   "mixed case",
			expectedUpper:   "MIXED CASE",
			expectedTrimmed: "tab-trimmed",
			expectedAdmin:   "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
			for name, value := range tt.cookies {
				req.AddCookie(&http.Cookie{Name: name, Value: value})
			}

			result, err := decoder.Decode(req)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedFirstIP, result.FirstIP)
			assert.Equal(t, tt.expectedLower, result.LowerCase)
			assert.Equal(t, tt.expectedUpper, result.UpperCase)
			assert.Equal(t, tt.expectedTrimmed, result.Trimmed)
			assert.Equal(t, tt.expectedAdmin, result.AdminFlag)
		})
	}
}

// TestCookieDecoderFormats tests custom format parsing for cookies.
func TestCookieDecoderFormats(t *testing.T) {
	type CookieRequest struct {
		UnixTime     time.Time `cookie:"unix_time" format:"unix"`
		RFC3339Time  time.Time `cookie:"rfc3339_time" format:"rfc3339"`
		RFC822Time   time.Time `cookie:"rfc822_time" format:"rfc822"`
		DateOnly     time.Time `cookie:"date" format:"2006-01-02"`
		DateTime     time.Time `cookie:"datetime" format:"2006-01-02 15:04:05"`
		CustomFormat time.Time `cookie:"custom" format:"02/Jan/2006:15:04:05 -0700"`
	}

	decoder := NewCookieDecoder[CookieRequest](nil)

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.AddCookie(&http.Cookie{Name: "unix_time", Value: "1703518245"}) // 2023-12-25 15:30:45 UTC
	req.AddCookie(&http.Cookie{Name: "rfc3339_time", Value: "2023-12-25T15:30:45Z"})
	req.AddCookie(&http.Cookie{Name: "rfc822_time", Value: "25 Dec 23 15:30 UTC"})
	req.AddCookie(&http.Cookie{Name: "date", Value: "2023-12-25"})
	req.AddCookie(&http.Cookie{Name: "datetime", Value: "2023-12-25 15:30:45"})
	req.AddCookie(&http.Cookie{Name: "custom", Value: "25/Dec/2023:15:30:45 +0000"})

	result, err := decoder.Decode(req)
	require.NoError(t, err)

	// Verify unix timestamp
	expectedUnix := time.Unix(1703518245, 0).UTC()
	assert.True(t, expectedUnix.Equal(result.UnixTime.UTC()))

	// Verify RFC3339
	expectedRFC3339, _ := time.Parse(time.RFC3339, "2023-12-25T15:30:45Z")
	assert.True(t, expectedRFC3339.Equal(result.RFC3339Time))

	// Verify RFC822
	expectedRFC822, _ := time.Parse(time.RFC822, "25 Dec 23 15:30 UTC")
	assert.True(t, expectedRFC822.Equal(result.RFC822Time))

	// Verify date only
	expectedDate, _ := time.Parse("2006-01-02", "2023-12-25")
	assert.True(t, expectedDate.Equal(result.DateOnly))

	// Verify datetime
	expectedDateTime, _ := time.Parse("2006-01-02 15:04:05", "2023-12-25 15:30:45")
	assert.True(t, expectedDateTime.Equal(result.DateTime))

	// Verify custom format
	expectedCustom, _ := time.Parse("02/Jan/2006:15:04:05 -0700", "25/Dec/2023:15:30:45 +0000")
	assert.True(t, expectedCustom.Equal(result.CustomFormat))
}

// TestCookieDecoderDefaultValues tests default value handling for cookies.
func TestCookieDecoderDefaultValues(t *testing.T) {
	type CookieRequest struct {
		WithDefault    string    `cookie:"missing" default:"default-value"`
		WithNowDefault time.Time `cookie:"now_time" default:"now"`
		WithUUID       string    `cookie:"uuid_field" default:"generate_uuid"`
		NoDefault      string    `cookie:"none"`
	}

	decoder := NewCookieDecoder[CookieRequest](nil)

	// Request with no cookies set
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)

	result, err := decoder.Decode(req)
	require.NoError(t, err)

	assert.Equal(t, "default-value", result.WithDefault)
	assert.False(t, result.WithNowDefault.IsZero()) // Should be set to current time
	assert.Contains(t, result.WithUUID, "uuid-")    // Should contain uuid prefix
	assert.Equal(t, "", result.NoDefault)           // Should remain empty
}

// TestCookieDecoderValidation tests validation integration for cookies.
func TestCookieDecoderValidation(t *testing.T) {
	type CookieRequest struct {
		Required  string `cookie:"required_cookie" validate:"required"`
		MinLength string `cookie:"min_cookie" validate:"min=3"`
		Email     string `cookie:"email_cookie" validate:"email"`
		Range     int    `cookie:"range_cookie" validate:"min=1,max=100"`
	}

	validator := validator.New()
	decoder := NewCookieDecoder[CookieRequest](validator)

	tests := []struct {
		name        string
		cookies     map[string]string
		shouldError bool
		errorField  string
	}{
		{
			name: "valid cookies",
			cookies: map[string]string{
				"required_cookie": "present",
				"min_cookie":      "long enough",
				"email_cookie":    "test@example.com",
				"range_cookie":    "50",
			},
			shouldError: false,
		},
		{
			name: "missing required field",
			cookies: map[string]string{
				"min_cookie":   "long enough",
				"email_cookie": "test@example.com",
				"range_cookie": "50",
			},
			shouldError: true,
			errorField:  "required",
		},
		{
			name: "too short field",
			cookies: map[string]string{
				"required_cookie": "present",
				"min_cookie":      "hi",
				"email_cookie":    "test@example.com",
				"range_cookie":    "50",
			},
			shouldError: true,
			errorField:  "min",
		},
		{
			name: "invalid email",
			cookies: map[string]string{
				"required_cookie": "present",
				"min_cookie":      "long enough",
				"email_cookie":    "not-an-email",
				"range_cookie":    "50",
			},
			shouldError: true,
			errorField:  "email",
		},
		{
			name: "out of range",
			cookies: map[string]string{
				"required_cookie": "present",
				"min_cookie":      "long enough",
				"email_cookie":    "test@example.com",
				"range_cookie":    "150",
			},
			shouldError: true,
			errorField:  "max",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
			for name, value := range tt.cookies {
				req.AddCookie(&http.Cookie{Name: name, Value: value})
			}

			result, err := decoder.Decode(req)

			if tt.shouldError {
				require.Error(t, err)
				var validationErr *ValidationError
				assert.True(t, errors.As(err, &validationErr))
				assert.Contains(t, err.Error(), "Cookie validation failed")
				// Check that the error fields contain the expected validation tag
				found := false
				for _, field := range validationErr.Fields {
					if field == tt.errorField {
						found = true

						break
					}
				}
				assert.True(t, found, "Expected validation error for field: %s", tt.errorField)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result.Required)
			}
		})
	}
}

// TestCookieDecoderErrorHandling tests various error conditions for cookies.
func TestCookieDecoderErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		requestBuilder func() *http.Request
		expectedError  string
	}{
		{
			name: "invalid integer",
			requestBuilder: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
				req.AddCookie(&http.Cookie{Name: "int_cookie", Value: "not-a-number"})

				return req
			},
			expectedError: "invalid integer value",
		},
		{
			name: "invalid boolean",
			requestBuilder: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
				req.AddCookie(&http.Cookie{Name: "bool_cookie", Value: "not-a-bool"})

				return req
			},
			expectedError: "invalid boolean value",
		},
		{
			name: "unknown transformation",
			requestBuilder: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
				req.AddCookie(&http.Cookie{Name: "transform_cookie", Value: "value"})

				return req
			},
			expectedError: "unknown transformation",
		},
		{
			name: "invalid unix timestamp",
			requestBuilder: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
				req.AddCookie(&http.Cookie{Name: "unix_cookie", Value: "not-unix"})

				return req
			},
			expectedError: "invalid unix timestamp",
		},
		{
			name: "unsupported format for type",
			requestBuilder: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
				req.AddCookie(&http.Cookie{Name: "format_cookie", Value: "value"})

				return req
			},
			expectedError: "format not supported for type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Define test types for each error case
			var decoder interface{}
			var result interface{}
			var err error

			switch tt.name {
			case "invalid integer":
				type IntRequest struct {
					Int int `cookie:"int_cookie"`
				}
				d := NewCookieDecoder[IntRequest](nil)
				r, e := d.Decode(tt.requestBuilder())
				decoder, result, err = d, r, e

			case "invalid boolean":
				type BoolRequest struct {
					Bool bool `cookie:"bool_cookie"`
				}
				d := NewCookieDecoder[BoolRequest](nil)
				r, e := d.Decode(tt.requestBuilder())
				decoder, result, err = d, r, e

			case "unknown transformation":
				type TransformRequest struct {
					Value string `cookie:"transform_cookie" transform:"unknown"`
				}
				d := NewCookieDecoder[TransformRequest](nil)
				r, e := d.Decode(tt.requestBuilder())
				decoder, result, err = d, r, e

			case "invalid unix timestamp":
				type UnixRequest struct {
					Unix time.Time `cookie:"unix_cookie" format:"unix"`
				}
				d := NewCookieDecoder[UnixRequest](nil)
				r, e := d.Decode(tt.requestBuilder())
				decoder, result, err = d, r, e

			case "unsupported format for type":
				type FormatRequest struct {
					Value string `cookie:"format_cookie" format:"unsupported"`
				}
				d := NewCookieDecoder[FormatRequest](nil)
				r, e := d.Decode(tt.requestBuilder())
				decoder, result, err = d, r, e
			}

			_ = decoder // Avoid unused variable
			_ = result  // Avoid unused variable

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

// TestCookieDecoderHelperFunctions tests all cookie helper functions.
func TestCookieDecoderHelperFunctions(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.AddCookie(&http.Cookie{Name: "session", Value: "abc123"})
	req.AddCookie(&http.Cookie{Name: "user", Value: "john"})
	req.AddCookie(&http.Cookie{Name: "preferences", Value: "dark-mode"})

	// Test GetAllCookies
	allCookies := GetAllCookies(req)
	assert.Len(t, allCookies, 3)
	assert.Equal(t, "abc123", allCookies["session"])
	assert.Equal(t, "john", allCookies["user"])
	assert.Equal(t, "dark-mode", allCookies["preferences"])

	// Test GetCookieWithDefault - existing cookie
	sessionValue := GetCookieWithDefault(req, "session", "default-session")
	assert.Equal(t, "abc123", sessionValue)

	// Test GetCookieWithDefault - non-existing cookie
	missingValue := GetCookieWithDefault(req, "nonexistent", "fallback")
	assert.Equal(t, "fallback", missingValue)

	// Test GetCookieWithDefault - empty default
	emptyDefault := GetCookieWithDefault(req, "missing", "")
	assert.Equal(t, "", emptyDefault)
}

// TestCookieDecoderSecurityFeatures tests security-related cookie features.
func TestCookieDecoderSecurityFeatures(t *testing.T) {
	secret := "super-secret-key"

	// Test ParseSignedCookie
	t.Run("ParseSignedCookie", func(t *testing.T) {
		cookieValue := "user123"
		parsed, err := ParseSignedCookie(cookieValue, secret)
		require.NoError(t, err)
		// Currently just returns the value as-is (placeholder implementation)
		assert.Equal(t, cookieValue, parsed)
	})

	// Test SecureCookieDecoder creation
	t.Run("SecureCookieDecoder creation", func(t *testing.T) {
		decoder := NewSecureCookieDecoder[ValidationTestRequest](nil, secret)
		assert.NotNil(t, decoder)

		// Test that it has the correct secret
		assert.Equal(t, secret, decoder.secret)

		// Test that it wraps a regular CookieDecoder
		assert.NotNil(t, decoder.decoder)
	})

	// Test SecureCookieDecoder ContentTypes
	t.Run("SecureCookieDecoder ContentTypes", func(t *testing.T) {
		decoder := NewSecureCookieDecoder[ValidationTestRequest](nil, secret)
		contentTypes := decoder.ContentTypes()
		assert.Contains(t, contentTypes, "*/*")
	})

	// Test SecureCookieDecoder Decode functionality
	t.Run("SecureCookieDecoder Decode", func(t *testing.T) {
		type SecureRequest struct {
			SessionID string `cookie:"secure_session"`
			UserID    string `cookie:"user_id"`
		}

		decoder := NewSecureCookieDecoder[SecureRequest](nil, secret)

		req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
		req.AddCookie(&http.Cookie{Name: "secure_session", Value: "encrypted_session_123"})
		req.AddCookie(&http.Cookie{Name: "user_id", Value: "user456"})

		result, err := decoder.Decode(req)
		require.NoError(t, err)
		assert.Equal(t, "encrypted_session_123", result.SessionID)
		assert.Equal(t, "user456", result.UserID)
	})
}

// TestCookieDecoderContentTypes tests ContentTypes method.
func TestCookieDecoderContentTypes(t *testing.T) {
	type CookieRequest struct {
		Value string `cookie:"test"`
	}

	decoder := NewCookieDecoder[CookieRequest](nil)
	contentTypes := decoder.ContentTypes()

	assert.Len(t, contentTypes, 1)
	assert.Equal(t, "*/*", contentTypes[0])
}

// TestCookieDecoderCombinedFeatures tests complex scenarios with multiple features.
func TestCookieDecoderCombinedFeatures(t *testing.T) {
	type ComplexCookieRequest struct {
		// Basic field
		SessionID string `cookie:"session_id"`

		// With transformation
		NormalizedName string `cookie:"user_name" transform:"to_lower"`

		// With transformation to boolean string
		IsAdmin string `cookie:"role" transform:"is_admin"`

		// With format and default
		LoginTime time.Time `cookie:"login_timestamp" format:"unix" default:"1703518245"`

		// With validation
		UserToken string `cookie:"auth_token" validate:"required,min=10"`

		// With case transformation
		Preferences string `cookie:"user_prefs" transform:"trim_space"`
	}

	validator := validator.New()
	decoder := NewCookieDecoder[ComplexCookieRequest](validator)

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess_12345"})
	req.AddCookie(&http.Cookie{Name: "user_name", Value: "JOHN DOE"})
	req.AddCookie(&http.Cookie{Name: "role", Value: "admin"})
	req.AddCookie(&http.Cookie{Name: "login_timestamp", Value: "1703520000"}) // Different timestamp
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: "Bearer token12345"})
	req.AddCookie(&http.Cookie{Name: "user_prefs", Value: "  dark-mode, notifications  "})

	result, err := decoder.Decode(req)
	require.NoError(t, err)

	assert.Equal(t, "sess_12345", result.SessionID)
	assert.Equal(t, "john doe", result.NormalizedName)
	assert.Equal(t, "true", result.IsAdmin)
	assert.True(t, time.Unix(1703520000, 0).Equal(result.LoginTime))
	assert.Equal(t, "Bearer token12345", result.UserToken)
	assert.Equal(t, "dark-mode, notifications", result.Preferences)
}

// TestCookieDecoderEmptyAndMissingCookies tests behavior with empty/missing cookies.
func TestCookieDecoderEmptyAndMissingCookies(t *testing.T) {
	type CookieRequest struct {
		Optional     string `cookie:"optional"`
		WithDefault  string `cookie:"default_cookie" default:"fallback"`
		EmptyAllowed string `cookie:"empty"`
	}

	decoder := NewCookieDecoder[CookieRequest](nil)

	tests := []struct {
		name     string
		cookies  map[string]string
		expected CookieRequest
	}{
		{
			name:    "no cookies",
			cookies: map[string]string{},
			expected: CookieRequest{
				Optional:     "",
				WithDefault:  "fallback",
				EmptyAllowed: "",
			},
		},
		{
			name: "empty cookie value",
			cookies: map[string]string{
				"empty": "",
			},
			expected: CookieRequest{
				Optional:     "",
				WithDefault:  "fallback",
				EmptyAllowed: "",
			},
		},
		{
			name: "some cookies present",
			cookies: map[string]string{
				"optional": "present",
			},
			expected: CookieRequest{
				Optional:     "present",
				WithDefault:  "fallback",
				EmptyAllowed: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
			for name, value := range tt.cookies {
				req.AddCookie(&http.Cookie{Name: name, Value: value})
			}

			result, err := decoder.Decode(req)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, result)
		})
	}
}
