package openapi

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestParseDefaultValue tests the parseDefaultValue function with various Go types.
func TestParseDefaultValue(t *testing.T) {
	generator := NewGenerator(&Config{})

	tests := []struct {
		name         string
		defaultValue string
		fieldType    reflect.Type
		expected     interface{}
	}{
		{
			name:         "string value",
			defaultValue: "test",
			fieldType:    reflect.TypeOf(""),
			expected:     "test",
		},
		{
			name:         "int value",
			defaultValue: "42",
			fieldType:    reflect.TypeOf(0),
			expected:     int64(42),
		},
		{
			name:         "uint value",
			defaultValue: "42",
			fieldType:    reflect.TypeOf(uint(0)),
			expected:     uint64(42),
		},
		{
			name:         "float value",
			defaultValue: "3.14",
			fieldType:    reflect.TypeOf(0.0),
			expected:     3.14,
		},
		{
			name:         "bool value true",
			defaultValue: "true",
			fieldType:    reflect.TypeOf(false),
			expected:     true,
		},
		{
			name:         "bool value false",
			defaultValue: "false",
			fieldType:    reflect.TypeOf(false),
			expected:     false,
		},
		{
			name:         "invalid int fallback to string",
			defaultValue: "invalid",
			fieldType:    reflect.TypeOf(0),
			expected:     "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.parseDefaultValue(tt.defaultValue, tt.fieldType)
			assert.Equal(t, tt.expected, result)
		})
	}
}
