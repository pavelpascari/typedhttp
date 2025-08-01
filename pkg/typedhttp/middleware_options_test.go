package typedhttp

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test middleware option functions
func TestWithName(t *testing.T) {
	config := &MiddlewareConfig{}
	option := WithName("test_middleware")
	option(config)

	assert.Equal(t, "test_middleware", config.Name)
}

func TestWithPriority(t *testing.T) {
	tests := []struct {
		name     string
		priority int
	}{
		{"high_priority", 100},
		{"default_priority", 0},
		{"low_priority", -50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &MiddlewareConfig{}
			option := WithPriority(tt.priority)
			option(config)

			assert.Equal(t, tt.priority, config.Priority)
		})
	}
}

func TestWithScope(t *testing.T) {
	tests := []struct {
		name  string
		scope MiddlewareScope
	}{
		{"global_scope", ScopeGlobal},
		{"group_scope", ScopeGroup},
		{"handler_scope", ScopeHandler},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &MiddlewareConfig{}
			option := WithScope(tt.scope)
			option(config)

			assert.Equal(t, tt.scope, config.Scope)
		})
	}
}

func TestWithCondition(t *testing.T) {
	condition := func(r *http.Request) bool {
		return r.Header.Get("X-Test") == "true"
	}

	config := &MiddlewareConfig{}
	option := WithCondition(condition)
	option(config)

	assert.NotNil(t, config.Conditional)

	// Test the condition
	req := &http.Request{Header: make(http.Header)}
	assert.False(t, config.Conditional(req))

	req.Header.Set("X-Test", "true")
	assert.True(t, config.Conditional(req))
}

func TestWithMetadata(t *testing.T) {
	metadata := map[string]any{
		"role":        "admin",
		"permissions": []string{"read", "write"},
		"timeout":     30,
	}

	config := &MiddlewareConfig{}
	option := WithMetadata(metadata)
	option(config)

	assert.Equal(t, metadata, config.Metadata)
	assert.Equal(t, "admin", config.Metadata["role"])
	assert.Equal(t, []string{"read", "write"}, config.Metadata["permissions"])
	assert.Equal(t, 30, config.Metadata["timeout"])
}

func TestWithMetadataKey(t *testing.T) {
	config := &MiddlewareConfig{}

	// Test single key-value pair
	option := WithMetadataKey("version", "1.0")
	option(config)

	assert.NotNil(t, config.Metadata)
	assert.Equal(t, "1.0", config.Metadata["version"])

	// Test adding another key-value pair
	option2 := WithMetadataKey("env", "production")
	option2(config)

	assert.Equal(t, "1.0", config.Metadata["version"])
	assert.Equal(t, "production", config.Metadata["env"])
}

// Test combining multiple options
func TestCombinedOptions(t *testing.T) {
	condition := func(r *http.Request) bool {
		return r.Method == http.MethodPost
	}

	config := &MiddlewareConfig{}

	// Apply multiple options
	WithName("auth_middleware")(config)
	WithPriority(50)(config)
	WithScope(ScopeGlobal)(config)
	WithCondition(condition)(config)
	WithMetadataKey("type", "authentication")(config)
	WithMetadataKey("version", "2.0")(config)

	assert.Equal(t, "auth_middleware", config.Name)
	assert.Equal(t, 50, config.Priority)
	assert.Equal(t, ScopeGlobal, config.Scope)
	assert.NotNil(t, config.Conditional)
	assert.Equal(t, "authentication", config.Metadata["type"])
	assert.Equal(t, "2.0", config.Metadata["version"])

	// Test condition
	postReq := &http.Request{Method: http.MethodPost}
	assert.True(t, config.Conditional(postReq))

	getReq := &http.Request{Method: http.MethodGet}
	assert.False(t, config.Conditional(getReq))
}

// Test option order independence
func TestOptionOrderIndependence(t *testing.T) {
	condition := func(r *http.Request) bool { return true }

	// Create two configs with options applied in different orders
	config1 := &MiddlewareConfig{}
	WithName("test")(config1)
	WithPriority(10)(config1)
	WithScope(ScopeGroup)(config1)
	WithCondition(condition)(config1)

	config2 := &MiddlewareConfig{}
	WithCondition(condition)(config2)
	WithScope(ScopeGroup)(config2)
	WithPriority(10)(config2)
	WithName("test")(config2)

	assert.Equal(t, config1.Name, config2.Name)
	assert.Equal(t, config1.Priority, config2.Priority)
	assert.Equal(t, config1.Scope, config2.Scope)
	assert.NotNil(t, config1.Conditional)
	assert.NotNil(t, config2.Conditional)
}

// Test metadata option edge cases
func TestMetadataEdgeCases(t *testing.T) {
	t.Run("nil_metadata", func(t *testing.T) {
		config := &MiddlewareConfig{}
		option := WithMetadata(nil)
		option(config)

		assert.Nil(t, config.Metadata)
	})

	t.Run("empty_metadata", func(t *testing.T) {
		config := &MiddlewareConfig{}
		option := WithMetadata(map[string]any{})
		option(config)

		assert.NotNil(t, config.Metadata)
		assert.Empty(t, config.Metadata)
	})

	t.Run("overwrite_metadata", func(t *testing.T) {
		config := &MiddlewareConfig{}

		// Set initial metadata
		WithMetadata(map[string]any{"key1": "value1"})(config)
		assert.Equal(t, "value1", config.Metadata["key1"])

		// Overwrite with new metadata
		WithMetadata(map[string]any{"key2": "value2"})(config)
		assert.Equal(t, "value2", config.Metadata["key2"])

		// Original key should be gone
		_, exists := config.Metadata["key1"]
		assert.False(t, exists)
	})

	t.Run("metadata_key_to_existing_metadata", func(t *testing.T) {
		config := &MiddlewareConfig{}

		// Set initial metadata with WithMetadata
		WithMetadata(map[string]any{"existing": "value"})(config)

		// Add individual key with WithMetadataKey
		WithMetadataKey("new", "newvalue")(config)

		assert.Equal(t, "value", config.Metadata["existing"])
		assert.Equal(t, "newvalue", config.Metadata["new"])
	})
}
