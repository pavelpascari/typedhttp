package openapi

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNeedsRequestBody tests the needsRequestBody function.
func TestNeedsRequestBody(t *testing.T) {
	generator := NewGenerator(&Config{})

	type JSONRequest struct {
		Name string `json:"name"`
	}

	type FormRequest struct {
		Name string `form:"name"`
	}

	type QueryRequest struct {
		Name string `query:"name"`
	}

	tests := []struct {
		name     string
		reqType  reflect.Type
		expected bool
	}{
		{
			name:     "JSON request needs body",
			reqType:  reflect.TypeOf(JSONRequest{}),
			expected: true,
		},
		{
			name:     "Form request needs body",
			reqType:  reflect.TypeOf(FormRequest{}),
			expected: true,
		},
		{
			name:     "Query request doesn't need body",
			reqType:  reflect.TypeOf(QueryRequest{}),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.needsRequestBody(tt.reqType)
			assert.Equal(t, tt.expected, result)
		})
	}
}
