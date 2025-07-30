package openapi

import (
	"mime/multipart"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHasFileUploads tests the hasFileUploads function.
func TestHasFileUploads(t *testing.T) {
	generator := NewGenerator(&Config{})

	type FileRequest struct {
		File *multipart.FileHeader `form:"file"`
	}

	type NoFileRequest struct {
		Name string `form:"name"`
	}

	// Test with file upload
	hasFiles := generator.hasFileUploads(reflect.TypeOf(FileRequest{}))
	assert.True(t, hasFiles)

	// Test without file upload
	noFiles := generator.hasFileUploads(reflect.TypeOf(NoFileRequest{}))
	assert.False(t, noFiles)
}
