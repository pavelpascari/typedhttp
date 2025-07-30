package typedhttp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMoreRouterFunctionality tests extended router functionality.
func TestMoreRouterFunctionality(t *testing.T) {
	router := NewRouter()

	// Test all HTTP methods
	POST(router, "/post", &SimpleTestHandler{})
	PUT(router, "/put", &SimpleTestHandler{})
	PATCH(router, "/patch", &SimpleTestHandler{})
	DELETE(router, "/delete", &SimpleTestHandler{})
	HEAD(router, "/head", &SimpleTestHandler{})
	OPTIONS(router, "/options", &SimpleTestHandler{})

	handlers := router.GetHandlers()
	assert.Len(t, handlers, 6)

	// Verify all methods were registered
	methods := make(map[string]bool)
	for _, handler := range handlers {
		methods[handler.Method] = true
	}

	assert.True(t, methods["POST"])
	assert.True(t, methods["PUT"])
	assert.True(t, methods["PATCH"])
	assert.True(t, methods["DELETE"])
	assert.True(t, methods["HEAD"])
	assert.True(t, methods["OPTIONS"])
}
