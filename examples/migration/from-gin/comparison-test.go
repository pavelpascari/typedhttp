package main

import (
	"context"
	"testing"

	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file demonstrates the testing differences between Gin and TypedHTTP

// TypedHTTP tests - Direct function testing, no HTTP layer needed
func TestGetUserHandler_Success(t *testing.T) {
	handler := &GetUserHandler{}
	req := GetUserRequest{ID: "1"}

	resp, err := handler.Handle(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "1", resp.ID)
	assert.Equal(t, "Alice Smith", resp.Name)
	assert.Equal(t, "alice@example.com", resp.Email)
}

func TestGetUserHandler_NotFound(t *testing.T) {
	handler := &GetUserHandler{}
	req := GetUserRequest{ID: "999"}

	_, err := handler.Handle(context.Background(), req)

	require.Error(t, err)
	var notFoundErr *typedhttp.NotFoundError
	assert.ErrorAs(t, err, &notFoundErr)
	assert.Equal(t, "user", notFoundErr.Resource)
	assert.Equal(t, "999", notFoundErr.ID)
}

func TestListUsersHandler_WithLimit(t *testing.T) {
	handler := &ListUsersHandler{}
	req := ListUsersRequest{Limit: 1}

	resp, err := handler.Handle(context.Background(), req)

	require.NoError(t, err)
	assert.Len(t, resp.Users, 1)
	assert.Equal(t, 2, resp.Total) // Total users in storage
	assert.Equal(t, 1, resp.Limit)
}

func TestCreateUserHandler_Success(t *testing.T) {
	handler := &CreateUserHandler{}
	req := CreateUserRequest{
		Name:  "Charlie Brown",
		Email: "charlie@example.com",
	}

	resp, err := handler.Handle(context.Background(), req)

	require.NoError(t, err)
	assert.NotEmpty(t, resp.ID)
	assert.Equal(t, "Charlie Brown", resp.Name)
	assert.Equal(t, "charlie@example.com", resp.Email)

	// Verify user was actually created
	_, exists := users[resp.ID]
	assert.True(t, exists)
}

func TestCreateUserHandler_DuplicateEmail(t *testing.T) {
	handler := &CreateUserHandler{}
	req := CreateUserRequest{
		Name:  "Duplicate Alice",
		Email: "alice@example.com", // Already exists
	}

	_, err := handler.Handle(context.Background(), req)

	require.Error(t, err)
	var conflictErr *typedhttp.ConflictError
	assert.ErrorAs(t, err, &conflictErr)
	assert.Contains(t, conflictErr.Message, "email already exists")
}

// Validation testing - automatic with TypedHTTP
func TestCreateUserHandler_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		request CreateUserRequest
		wantErr string
	}{
		{
			name:    "empty name",
			request: CreateUserRequest{Name: "", Email: "test@example.com"},
			wantErr: "validation failed", // Would be caught by validation
		},
		{
			name:    "short name",
			request: CreateUserRequest{Name: "A", Email: "test@example.com"},
			wantErr: "validation failed", // Would be caught by validation
		},
		{
			name:    "invalid email",
			request: CreateUserRequest{Name: "Valid Name", Email: "invalid-email"},
			wantErr: "validation failed", // Would be caught by validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: In a real TypedHTTP implementation, validation would happen
			// before the handler is called, so these would be integration tests
			// rather than unit tests. For demonstration, we'll just show the pattern.
			
			handler := &CreateUserHandler{}
			_, err := handler.Handle(context.Background(), tt.request)
			
			// In practice, validation errors would be caught by the framework
			// before reaching the handler, but this shows the testing pattern
			if tt.wantErr != "" {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Performance benchmark comparison
func BenchmarkGetUserHandler(b *testing.B) {
	handler := &GetUserHandler{}
	req := GetUserRequest{ID: "1"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := handler.Handle(context.Background(), req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Compare to Gin testing (for reference - would require HTTP setup):
/*
func TestGinGetUser(t *testing.T) {
    gin.SetMode(gin.TestMode)
    r := gin.Default()
    r.GET("/users/:id", GetUser)

    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/users/1", nil)
    r.ServeHTTP(w, req)

    assert.Equal(t, 200, w.Code)
    
    var user User
    err := json.Unmarshal(w.Body.Bytes(), &user)
    require.NoError(t, err)
    assert.Equal(t, "1", user.ID)
}
*/

// Key testing advantages of TypedHTTP:
// 1. Direct function testing - no HTTP layer needed
// 2. Type-safe requests/responses - no JSON marshaling in tests
// 3. Proper error types - specific error assertions
// 4. Faster tests - no HTTP server setup/teardown
// 5. Better test coverage - can test edge cases more easily
// 6. Validation testing - framework handles validation automatically