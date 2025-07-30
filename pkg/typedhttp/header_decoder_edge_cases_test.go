package typedhttp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHeaderDecoderEdgeCases tests header decoder edge cases.
func TestHeaderDecoderEdgeCases(t *testing.T) {
	type HeaderTestRequest struct {
		ContentType string `header:"Content-Type"`
		UserAgent   string `header:"User-Agent"`
	}

	decoder := NewHeaderDecoder[HeaderTestRequest](nil)

	// Test with headers
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "TestAgent/1.0")

	result, err := decoder.Decode(req)
	require.NoError(t, err)
	assert.Equal(t, "application/json", result.ContentType)
	assert.Equal(t, "TestAgent/1.0", result.UserAgent)
}
