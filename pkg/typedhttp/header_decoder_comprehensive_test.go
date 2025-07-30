package typedhttp

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHeaderDecoderBasicTypes tests all basic type conversions.
func TestHeaderDecoderBasicTypes(t *testing.T) {
	type HeaderRequest struct {
		StringField  string  `header:"X-String"`
		IntField     int     `header:"X-Int"`
		Int8Field    int8    `header:"X-Int8"`
		Int16Field   int16   `header:"X-Int16"`
		Int32Field   int32   `header:"X-Int32"`
		Int64Field   int64   `header:"X-Int64"`
		UintField    uint    `header:"X-Uint"`
		Uint8Field   uint8   `header:"X-Uint8"`
		Uint16Field  uint16  `header:"X-Uint16"`
		Uint32Field  uint32  `header:"X-Uint32"`
		Uint64Field  uint64  `header:"X-Uint64"`
		Float32Field float32 `header:"X-Float32"`
		Float64Field float64 `header:"X-Float64"`
		BoolField    bool    `header:"X-Bool"`
	}

	decoder := NewHeaderDecoder[HeaderRequest](nil)

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("X-String", "hello")
	req.Header.Set("X-Int", "42")
	req.Header.Set("X-Int8", "127")
	req.Header.Set("X-Int16", "32767")
	req.Header.Set("X-Int32", "2147483647")
	req.Header.Set("X-Int64", "9223372036854775807")
	req.Header.Set("X-Uint", "42")
	req.Header.Set("X-Uint8", "255")
	req.Header.Set("X-Uint16", "65535")
	req.Header.Set("X-Uint32", "4294967295")
	req.Header.Set("X-Uint64", "18446744073709551615")
	req.Header.Set("X-Float32", "3.14")
	req.Header.Set("X-Float64", "2.718281828")
	req.Header.Set("X-Bool", "true")

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

// TestHeaderDecoderSpecialTypes tests IP and Time special types.
func TestHeaderDecoderSpecialTypes(t *testing.T) {
	type HeaderRequest struct {
		IPField   net.IP    `header:"X-IP"`
		TimeField time.Time `header:"X-Time"`
	}

	decoder := NewHeaderDecoder[HeaderRequest](nil)

	tests := []struct {
		name             string
		ip               string
		timeStr          string
		expectIP         net.IP
		expectTimeFormat string
	}{
		{
			name:             "IPv4 and RFC3339 time",
			ip:               "192.168.1.1",
			timeStr:          "2023-12-25T15:30:45Z",
			expectIP:         net.ParseIP("192.168.1.1"),
			expectTimeFormat: time.RFC3339,
		},
		{
			name:             "IPv6 and datetime format",
			ip:               "2001:db8::1",
			timeStr:          "2023-12-25 15:30:45",
			expectIP:         net.ParseIP("2001:db8::1"),
			expectTimeFormat: "2006-01-02 15:04:05",
		},
		{
			name:             "IPv4-mapped IPv6 and date format",
			ip:               "::ffff:192.168.1.1",
			timeStr:          "2023-12-25",
			expectIP:         net.ParseIP("::ffff:192.168.1.1"),
			expectTimeFormat: "2006-01-02",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
			req.Header.Set("X-IP", tt.ip)
			req.Header.Set("X-Time", tt.timeStr)

			result, err := decoder.Decode(req)
			require.NoError(t, err)

			assert.True(t, tt.expectIP.Equal(result.IPField))

			expectedTime, err := time.Parse(tt.expectTimeFormat, tt.timeStr)
			require.NoError(t, err)
			assert.True(t, expectedTime.Equal(result.TimeField))
		})
	}
}

// TestHeaderDecoderTransformations tests all transformation functions.
func TestHeaderDecoderTransformations(t *testing.T) {
	type HeaderRequest struct {
		FirstIP   string `header:"X-Forwarded-For" transform:"first_ip"`
		LowerCase string `header:"X-Upper" transform:"to_lower"`
		UpperCase string `header:"X-Lower" transform:"to_upper"`
		Trimmed   string `header:"X-Spaces" transform:"trim_space"`
		AdminFlag bool   `header:"X-Role" transform:"is_admin"`
	}

	decoder := NewHeaderDecoder[HeaderRequest](nil)

	tests := []struct {
		name            string
		forwardedFor    string
		upperHeader     string
		lowerHeader     string
		spacesHeader    string
		roleHeader      string
		expectedFirstIP string
		expectedLower   string
		expectedUpper   string
		expectedTrimmed string
		expectedIsAdmin bool
	}{
		{
			name:            "basic transformations",
			forwardedFor:    "192.168.1.1, 10.0.0.1, 172.16.0.1",
			upperHeader:     "HELLO WORLD",
			lowerHeader:     "hello world",
			spacesHeader:    "  trimmed  ",
			roleHeader:      "admin",
			expectedFirstIP: "192.168.1.1",
			expectedLower:   "hello world",
			expectedUpper:   "HELLO WORLD",
			expectedTrimmed: "trimmed",
			expectedIsAdmin: true,
		},
		{
			name:            "admin case insensitive",
			forwardedFor:    "203.0.113.1",
			upperHeader:     "Test",
			lowerHeader:     "TEST",
			spacesHeader:    "no-spaces",
			roleHeader:      "ADMIN",
			expectedFirstIP: "203.0.113.1",
			expectedLower:   "test",
			expectedUpper:   "TEST",
			expectedTrimmed: "no-spaces",
			expectedIsAdmin: true,
		},
		{
			name:            "non-admin role",
			forwardedFor:    "10.0.0.1, 192.168.1.1",
			upperHeader:     "mixed Case",
			lowerHeader:     "Mixed Case",
			spacesHeader:    "\ttab-trimmed\t",
			roleHeader:      "user",
			expectedFirstIP: "10.0.0.1",
			expectedLower:   "mixed case",
			expectedUpper:   "MIXED CASE",
			expectedTrimmed: "tab-trimmed",
			expectedIsAdmin: false,
		},
		{
			name:            "single IP",
			forwardedFor:    "172.16.0.1",
			upperHeader:     "abc",
			lowerHeader:     "ABC",
			spacesHeader:    " single-space ",
			roleHeader:      "moderator",
			expectedFirstIP: "172.16.0.1",
			expectedLower:   "abc",
			expectedUpper:   "ABC",
			expectedTrimmed: "single-space",
			expectedIsAdmin: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
			req.Header.Set("X-Forwarded-For", tt.forwardedFor)
			req.Header.Set("X-Upper", tt.upperHeader)
			req.Header.Set("X-Lower", tt.lowerHeader)
			req.Header.Set("X-Spaces", tt.spacesHeader)
			req.Header.Set("X-Role", tt.roleHeader)

			result, err := decoder.Decode(req)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedFirstIP, result.FirstIP)
			assert.Equal(t, tt.expectedLower, result.LowerCase)
			assert.Equal(t, tt.expectedUpper, result.UpperCase)
			assert.Equal(t, tt.expectedTrimmed, result.Trimmed)
			assert.Equal(t, tt.expectedIsAdmin, result.AdminFlag)
		})
	}
}

// TestHeaderDecoderFormats tests custom format parsing.
func TestHeaderDecoderFormats(t *testing.T) {
	type HeaderRequest struct {
		UnixTime     time.Time `header:"X-Unix-Time" format:"unix"`
		RFC3339Time  time.Time `header:"X-RFC3339-Time" format:"rfc3339"`
		RFC822Time   time.Time `header:"X-RFC822-Time" format:"rfc822"`
		DateOnly     time.Time `header:"X-Date" format:"2006-01-02"`
		DateTime     time.Time `header:"X-DateTime" format:"2006-01-02 15:04:05"`
		CustomFormat time.Time `header:"X-Custom" format:"02/Jan/2006:15:04:05 -0700"`
	}

	decoder := NewHeaderDecoder[HeaderRequest](nil)

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("X-Unix-Time", "1703518245") // 2023-12-25 15:30:45 UTC
	req.Header.Set("X-RFC3339-Time", "2023-12-25T15:30:45Z")
	req.Header.Set("X-RFC822-Time", "25 Dec 23 15:30 UTC")
	req.Header.Set("X-Date", "2023-12-25")
	req.Header.Set("X-DateTime", "2023-12-25 15:30:45")
	req.Header.Set("X-Custom", "25/Dec/2023:15:30:45 +0000")

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

// TestHeaderDecoderDefaultValues tests default value handling.
func TestHeaderDecoderDefaultValues(t *testing.T) {
	type HeaderRequest struct {
		WithDefault    string    `header:"X-Missing" default:"default-value"`
		WithNowDefault time.Time `header:"X-Now" default:"now"`
		WithUUID       string    `header:"X-UUID" default:"generate_uuid"`
		NoDefault      string    `header:"X-None"`
	}

	decoder := NewHeaderDecoder[HeaderRequest](nil)

	// Request with no headers set
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)

	result, err := decoder.Decode(req)
	require.NoError(t, err)

	assert.Equal(t, "default-value", result.WithDefault)
	assert.False(t, result.WithNowDefault.IsZero()) // Should be set to current time
	assert.Contains(t, result.WithUUID, "uuid-")    // Should contain uuid prefix
	assert.Equal(t, "", result.NoDefault)           // Should remain empty
}

// TestHeaderDecoderValidation tests validation integration.
func TestHeaderDecoderValidation(t *testing.T) {
	type HeaderRequest struct {
		Required  string `header:"X-Required" validate:"required"`
		MinLength string `header:"X-Min" validate:"min=3"`
		Email     string `header:"X-Email" validate:"email"`
		Range     int    `header:"X-Range" validate:"min=1,max=100"`
	}

	validator := validator.New()
	decoder := NewHeaderDecoder[HeaderRequest](validator)

	tests := []struct {
		name        string
		headers     map[string]string
		shouldError bool
		errorField  string
	}{
		{
			name: "valid headers",
			headers: map[string]string{
				"X-Required": "present",
				"X-Min":      "long enough",
				"X-Email":    "test@example.com",
				"X-Range":    "50",
			},
			shouldError: false,
		},
		{
			name: "missing required field",
			headers: map[string]string{
				"X-Min":   "long enough",
				"X-Email": "test@example.com",
				"X-Range": "50",
			},
			shouldError: true,
			errorField:  "required",
		},
		{
			name: "too short field",
			headers: map[string]string{
				"X-Required": "present",
				"X-Min":      "hi",
				"X-Email":    "test@example.com",
				"X-Range":    "50",
			},
			shouldError: true,
			errorField:  "min",
		},
		{
			name: "invalid email",
			headers: map[string]string{
				"X-Required": "present",
				"X-Min":      "long enough",
				"X-Email":    "not-an-email",
				"X-Range":    "50",
			},
			shouldError: true,
			errorField:  "email",
		},
		{
			name: "out of range",
			headers: map[string]string{
				"X-Required": "present",
				"X-Min":      "long enough",
				"X-Email":    "test@example.com",
				"X-Range":    "150",
			},
			shouldError: true,
			errorField:  "max",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
			for header, value := range tt.headers {
				req.Header.Set(header, value)
			}

			result, err := decoder.Decode(req)

			if tt.shouldError {
				require.Error(t, err)
				var validationErr *ValidationError
				assert.True(t, errors.As(err, &validationErr))
				assert.Contains(t, err.Error(), "Header validation failed")
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

// TestHeaderDecoderErrorHandling tests various error conditions.
func TestHeaderDecoderErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		requestBuilder func() *http.Request
		expectedError  string
	}{
		{
			name: "invalid IP address",
			requestBuilder: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
				req.Header.Set("X-IP", "not-an-ip")

				return req
			},
			expectedError: "invalid IP address",
		},
		{
			name: "invalid time format",
			requestBuilder: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
				req.Header.Set("X-Time", "not-a-time")

				return req
			},
			expectedError: "invalid time value",
		},
		{
			name: "invalid integer",
			requestBuilder: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
				req.Header.Set("X-Int", "not-a-number")

				return req
			},
			expectedError: "invalid integer value",
		},
		{
			name: "invalid boolean",
			requestBuilder: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
				req.Header.Set("X-Bool", "not-a-bool")

				return req
			},
			expectedError: "invalid boolean value",
		},
		{
			name: "unknown transformation",
			requestBuilder: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
				req.Header.Set("X-Transform", "value")

				return req
			},
			expectedError: "unknown transformation",
		},
		{
			name: "invalid unix timestamp",
			requestBuilder: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
				req.Header.Set("X-Unix", "not-unix")

				return req
			},
			expectedError: "invalid unix timestamp",
		},
		{
			name: "unsupported format for type",
			requestBuilder: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
				req.Header.Set("X-Format", "value")

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
			case "invalid IP address":
				type IPRequest struct {
					IP net.IP `header:"X-IP"`
				}
				d := NewHeaderDecoder[IPRequest](nil)
				r, e := d.Decode(tt.requestBuilder())
				decoder, result, err = d, r, e

			case "invalid time format":
				type TimeRequest struct {
					Time time.Time `header:"X-Time"`
				}
				d := NewHeaderDecoder[TimeRequest](nil)
				r, e := d.Decode(tt.requestBuilder())
				decoder, result, err = d, r, e

			case "invalid integer":
				type IntRequest struct {
					Int int `header:"X-Int"`
				}
				d := NewHeaderDecoder[IntRequest](nil)
				r, e := d.Decode(tt.requestBuilder())
				decoder, result, err = d, r, e

			case "invalid boolean":
				type BoolRequest struct {
					Bool bool `header:"X-Bool"`
				}
				d := NewHeaderDecoder[BoolRequest](nil)
				r, e := d.Decode(tt.requestBuilder())
				decoder, result, err = d, r, e

			case "unknown transformation":
				type TransformRequest struct {
					Value string `header:"X-Transform" transform:"unknown"`
				}
				d := NewHeaderDecoder[TransformRequest](nil)
				r, e := d.Decode(tt.requestBuilder())
				decoder, result, err = d, r, e

			case "invalid unix timestamp":
				type UnixRequest struct {
					Unix time.Time `header:"X-Unix" format:"unix"`
				}
				d := NewHeaderDecoder[UnixRequest](nil)
				r, e := d.Decode(tt.requestBuilder())
				decoder, result, err = d, r, e

			case "unsupported format for type":
				type FormatRequest struct {
					Value string `header:"X-Format" format:"unsupported"`
				}
				d := NewHeaderDecoder[FormatRequest](nil)
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

// TestHeaderDecoderContentTypes tests ContentTypes method.
func TestHeaderDecoderContentTypes(t *testing.T) {
	type HeaderRequest struct {
		Value string `header:"X-Value"`
	}

	decoder := NewHeaderDecoder[HeaderRequest](nil)
	contentTypes := decoder.ContentTypes()

	assert.Len(t, contentTypes, 1)
	assert.Equal(t, "*/*", contentTypes[0])
}

// TestHeaderDecoderCombinedFeatures tests complex scenarios with multiple features.
func TestHeaderDecoderCombinedFeatures(t *testing.T) {
	type ComplexHeaderRequest struct {
		// Basic field
		UserID string `header:"X-User-ID"`

		// With transformation and IP parsing
		ClientIP net.IP `header:"X-Forwarded-For" transform:"first_ip"`

		// With transformation to boolean
		IsAdmin bool `header:"X-Role" transform:"is_admin"`

		// With format and default
		Timestamp time.Time `header:"X-Timestamp" format:"unix" default:"1703518245"`

		// With validation
		APIKey string `header:"Authorization" validate:"required,min=10"`

		// With case transformation
		Normalized string `header:"X-CasE-InSeNsItIvE" transform:"to_lower"`
	}

	validator := validator.New()
	decoder := NewHeaderDecoder[ComplexHeaderRequest](validator)

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("X-User-ID", "user123")
	req.Header.Set("X-Forwarded-For", "203.0.113.1, 192.168.1.1, 10.0.0.1")
	req.Header.Set("X-Role", "admin")
	req.Header.Set("X-Timestamp", "1703520000") // Different timestamp
	req.Header.Set("Authorization", "Bearer token12345")
	req.Header.Set("X-CasE-InSeNsItIvE", "MiXeD cAsE")

	result, err := decoder.Decode(req)
	require.NoError(t, err)

	assert.Equal(t, "user123", result.UserID)
	assert.True(t, net.ParseIP("203.0.113.1").Equal(result.ClientIP))
	assert.True(t, result.IsAdmin)
	assert.True(t, time.Unix(1703520000, 0).Equal(result.Timestamp))
	assert.Equal(t, "Bearer token12345", result.APIKey)
	assert.Equal(t, "mixed case", result.Normalized)
}

// TestHeaderDecoderEmptyAndMissingHeaders tests behavior with empty/missing headers.
func TestHeaderDecoderEmptyAndMissingHeaders(t *testing.T) {
	type HeaderRequest struct {
		Optional     string `header:"X-Optional"`
		WithDefault  string `header:"X-Default" default:"fallback"`
		EmptyAllowed string `header:"X-Empty"`
	}

	decoder := NewHeaderDecoder[HeaderRequest](nil)

	tests := []struct {
		name     string
		headers  map[string]string
		expected HeaderRequest
	}{
		{
			name:    "no headers",
			headers: map[string]string{},
			expected: HeaderRequest{
				Optional:     "",
				WithDefault:  "fallback",
				EmptyAllowed: "",
			},
		},
		{
			name: "empty header value",
			headers: map[string]string{
				"X-Empty": "",
			},
			expected: HeaderRequest{
				Optional:     "",
				WithDefault:  "fallback",
				EmptyAllowed: "",
			},
		},
		{
			name: "some headers present",
			headers: map[string]string{
				"X-Optional": "present",
			},
			expected: HeaderRequest{
				Optional:     "present",
				WithDefault:  "fallback",
				EmptyAllowed: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
			for header, value := range tt.headers {
				req.Header.Set(header, value)
			}

			result, err := decoder.Decode(req)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, result)
		})
	}
}
