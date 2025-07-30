package typedhttp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestValidateFileUpload tests the ValidateFileUpload function.
func TestValidateFileUpload(t *testing.T) {
	// Test not allowing files
	optionsNoFiles := FormOptions{AllowFiles: false}
	err := ValidateFileUpload(nil, optionsNoFiles)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file uploads not allowed")
}
