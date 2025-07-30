package typedhttp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// SimpleTestHandler is a simple test handler for router tests.
type SimpleTestHandler struct{}

func (h *SimpleTestHandler) Handle(ctx context.Context, req ValidationTestRequest) (ValidationTestRequest, error) {
	return req, nil
}

// TestRouterErrorHandling tests router error handling and basic functionality.
func TestRouterErrorHandling(t *testing.T) {
	router := NewRouter()

	// Test GetHandlers
	handlers := router.GetHandlers()
	assert.NotNil(t, handlers)
	assert.Empty(t, handlers)

	// Register a handler using the proper method
	GET(router, "/test", &SimpleTestHandler{})

	handlers = router.GetHandlers()
	assert.Len(t, handlers, 1)
	assert.Equal(t, "GET", handlers[0].Method)
	assert.Equal(t, "/test", handlers[0].Path)
}
