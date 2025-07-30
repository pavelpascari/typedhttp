package typedhttp_test

import (
	"net/http"
	"testing"

	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidationError(t *testing.T) {
	fields := map[string]string{
		"name":  "required",
		"email": "email",
	}
	
	err := typedhttp.NewValidationError("Validation failed", fields)
	
	assert.Equal(t, "Validation failed", err.Error())
	assert.Equal(t, fields, err.Fields)
}

func TestNotFoundError(t *testing.T) {
	err := typedhttp.NewNotFoundError("user", "123")
	
	assert.Equal(t, "user with id '123' not found", err.Error())
	assert.Equal(t, "user", err.Resource)
	assert.Equal(t, "123", err.ID)
}

func TestConflictError(t *testing.T) {
	err := typedhttp.NewConflictError("Resource already exists")
	
	assert.Equal(t, "Resource already exists", err.Error())
	assert.Equal(t, "Resource already exists", err.Message)
}

func TestUnauthorizedError(t *testing.T) {
	err := typedhttp.NewUnauthorizedError("Invalid credentials")
	
	assert.Equal(t, "Invalid credentials", err.Error())
	assert.Equal(t, "Invalid credentials", err.Message)
}

func TestForbiddenError(t *testing.T) {
	err := typedhttp.NewForbiddenError("Access denied")
	
	assert.Equal(t, "Access denied", err.Error())
	assert.Equal(t, "Access denied", err.Message)
}

func TestDefaultErrorMapper_ValidationError(t *testing.T) {
	mapper := &typedhttp.DefaultErrorMapper{}
	err := typedhttp.NewValidationError("Validation failed", map[string]string{
		"name": "required",
	})
	
	statusCode, response := mapper.MapError(err)
	
	assert.Equal(t, http.StatusBadRequest, statusCode)
	
	errorResp, ok := response.(typedhttp.ErrorResponse)
	require.True(t, ok)
	assert.Equal(t, "Validation failed", errorResp.Error)
	assert.Equal(t, "VALIDATION_ERROR", errorResp.Code)
	assert.NotNil(t, errorResp.Details)
}

func TestDefaultErrorMapper_NotFoundError(t *testing.T) {
	mapper := &typedhttp.DefaultErrorMapper{}
	err := typedhttp.NewNotFoundError("user", "123")
	
	statusCode, response := mapper.MapError(err)
	
	assert.Equal(t, http.StatusNotFound, statusCode)
	
	errorResp, ok := response.(typedhttp.ErrorResponse)
	require.True(t, ok)
	assert.Equal(t, "user with id '123' not found", errorResp.Error)
	assert.Equal(t, "NOT_FOUND", errorResp.Code)
}

func TestDefaultErrorMapper_ConflictError(t *testing.T) {
	mapper := &typedhttp.DefaultErrorMapper{}
	err := typedhttp.NewConflictError("Resource conflict")
	
	statusCode, response := mapper.MapError(err)
	
	assert.Equal(t, http.StatusConflict, statusCode)
	
	errorResp, ok := response.(typedhttp.ErrorResponse)
	require.True(t, ok)
	assert.Equal(t, "Resource conflict", errorResp.Error)
	assert.Equal(t, "CONFLICT", errorResp.Code)
}

func TestDefaultErrorMapper_UnauthorizedError(t *testing.T) {
	mapper := &typedhttp.DefaultErrorMapper{}
	err := typedhttp.NewUnauthorizedError("Invalid token")
	
	statusCode, response := mapper.MapError(err)
	
	assert.Equal(t, http.StatusUnauthorized, statusCode)
	
	errorResp, ok := response.(typedhttp.ErrorResponse)
	require.True(t, ok)
	assert.Equal(t, "Invalid token", errorResp.Error)
	assert.Equal(t, "UNAUTHORIZED", errorResp.Code)
}

func TestDefaultErrorMapper_ForbiddenError(t *testing.T) {
	mapper := &typedhttp.DefaultErrorMapper{}
	err := typedhttp.NewForbiddenError("Insufficient permissions")
	
	statusCode, response := mapper.MapError(err)
	
	assert.Equal(t, http.StatusForbidden, statusCode)
	
	errorResp, ok := response.(typedhttp.ErrorResponse)
	require.True(t, ok)
	assert.Equal(t, "Insufficient permissions", errorResp.Error)
	assert.Equal(t, "FORBIDDEN", errorResp.Code)
}

func TestDefaultErrorMapper_UnknownError(t *testing.T) {
	mapper := &typedhttp.DefaultErrorMapper{}
	err := assert.AnError // Generic error
	
	statusCode, response := mapper.MapError(err)
	
	assert.Equal(t, http.StatusInternalServerError, statusCode)
	
	errorResp, ok := response.(typedhttp.ErrorResponse)
	require.True(t, ok)
	assert.Equal(t, "Internal server error", errorResp.Error)
	assert.Equal(t, "INTERNAL_ERROR", errorResp.Code)
}