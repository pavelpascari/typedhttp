package typedhttp

import (
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetFileHeaderFunctions tests GetFileHeader and GetFileHeaders functions.
func TestGetFileHeaderFunctions(t *testing.T) {
	// Test with no multipart form
	req := httptest.NewRequest(http.MethodPost, "/test", http.NoBody)

	_, err := GetFileHeader(req, "file")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no multipart form data")

	_, err = GetFileHeaders(req, "files")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no multipart form data")
}

// TestMoreFormHelpers tests additional form helper functions.
func TestMoreFormHelpers(t *testing.T) {
	// Create a request with multipart form but no files
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("name=test"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_ = req.ParseForm() // ignoring error for test

	// Test with form that has no multipart data
	_, err := GetFileHeader(req, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no multipart form data")

	// Test form with no files for field
	req.MultipartForm = &multipart.Form{
		File: make(map[string][]*multipart.FileHeader),
	}

	_, err = GetFileHeader(req, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no file found")

	_, err = GetFileHeaders(req, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no files found")
}
