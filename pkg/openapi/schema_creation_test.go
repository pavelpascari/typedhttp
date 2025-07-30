package openapi

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateSchemaFromType_ComplexTypes tests createSchemaFromType with various Go types.
func TestCreateSchemaFromType_ComplexTypes(t *testing.T) {
	generator := NewGenerator(&Config{})

	tests := []struct {
		name         string
		inputType    interface{}
		expectedType string
	}{
		{
			name:         "string slice",
			inputType:    []string{},
			expectedType: "array",
		},
		{
			name:         "int slice",
			inputType:    []int{},
			expectedType: "array",
		},
		{
			name:         "map[string]interface{}",
			inputType:    map[string]interface{}{},
			expectedType: "object",
		},
		{
			name: "custom struct",
			inputType: struct {
				Name string `json:"name"`
			}{},
			expectedType: "object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := generator.createSchemaFromType(reflect.TypeOf(tt.inputType))
			require.NoError(t, err)
			require.NotNil(t, schema.Value)
			require.NotNil(t, schema.Value.Type)
			assert.Equal(t, tt.expectedType, (*schema.Value.Type)[0])
		})
	}
}
