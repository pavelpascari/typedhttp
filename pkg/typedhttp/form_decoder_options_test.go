package typedhttp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewFormDecoderWithOptions tests the NewFormDecoderWithOptions function.
func TestNewFormDecoderWithOptions(t *testing.T) {
	decoder := NewFormDecoderWithOptions[ValidationTestRequest](nil, 1024, false)
	assert.NotNil(t, decoder)
	assert.Equal(t, int64(1024), decoder.maxMemory)
	assert.False(t, decoder.allowFiles)
}
