package typedhttp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestFormHelperFunctions tests GetFormValue and related helper functions.
func TestFormHelperFunctions(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/test", http.NoBody)
	req.Form = map[string][]string{
		"name":  {"John"},
		"empty": {""},
		"multi": {"value1", "value2"},
	}

	// Test GetFormValue
	assert.Equal(t, "John", GetFormValue(req, "name", "default"))
	assert.Equal(t, "default", GetFormValue(req, "nonexistent", "default"))
	assert.Equal(t, "default", GetFormValue(req, "empty", "default"))

	// Test GetFormValues
	values := GetFormValues(req, "multi")
	assert.Equal(t, []string{"value1", "value2"}, values)
	assert.Nil(t, GetFormValues(req, "nonexistent"))

	// Test with nil form
	reqNilForm := httptest.NewRequest(http.MethodPost, "/test", http.NoBody)
	assert.Equal(t, "default", GetFormValue(reqNilForm, "name", "default"))
	assert.Nil(t, GetFormValues(reqNilForm, "name"))
}

// TestGetFormInfo tests the GetFormInfo function.
func TestGetFormInfo(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/test", http.NoBody)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Form = map[string][]string{
		"name": {"John"},
		"age":  {"30"},
	}

	info := GetFormInfo(req)
	assert.Equal(t, "application/x-www-form-urlencoded", info.ContentType)
	assert.Equal(t, 2, info.FieldCount)
	assert.Equal(t, 0, info.FileCount)
	assert.Equal(t, int64(0), info.TotalSize)
	assert.Contains(t, info.Fields, "name")
	assert.Contains(t, info.Fields, "age")
}
