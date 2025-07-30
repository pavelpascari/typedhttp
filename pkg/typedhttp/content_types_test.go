package typedhttp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestContentTypesMethods tests the ContentTypes methods that are currently uncovered.
func TestContentTypesMethods(t *testing.T) {
	// Test FormDecoder ContentTypes
	formDecoder := NewFormDecoder[ValidationTestRequest](nil)
	contentTypes := formDecoder.ContentTypes()
	assert.Contains(t, contentTypes, "application/x-www-form-urlencoded")
	assert.Contains(t, contentTypes, "multipart/form-data")

	// Test HeaderDecoder ContentTypes
	headerDecoder := NewHeaderDecoder[ValidationTestRequest](nil)
	headerContentTypes := headerDecoder.ContentTypes()
	assert.Contains(t, headerContentTypes, "*/*")

	// Test CookieDecoder ContentTypes
	cookieDecoder := NewCookieDecoder[ValidationTestRequest](nil)
	cookieContentTypes := cookieDecoder.ContentTypes()
	assert.Contains(t, cookieContentTypes, "*/*")
}
