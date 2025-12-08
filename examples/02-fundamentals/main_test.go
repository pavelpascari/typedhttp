package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserStore_Get(t *testing.T) {
	store := NewUserStore()

	// Test existing user
	user, exists := store.Get("1")
	assert.True(t, exists)
	assert.Equal(t, "1", user.ID)
	assert.Equal(t, "Alice Smith", user.Name)

	// Test non-existing user
	_, exists = store.Get("999")
	assert.False(t, exists)
}

func TestUserStore_Create(t *testing.T) {
	store := NewUserStore()

	// Test successful creation
	user, err := store.Create("Charlie Brown", "charlie@example.com")
	require.NoError(t, err)
	assert.NotEmpty(t, user.ID)
	assert.Equal(t, "Charlie Brown", user.Name)
	assert.Equal(t, "charlie@example.com", user.Email)

	// Verify user was stored
	stored, exists := store.Get(user.ID)
	require.True(t, exists)
	assert.Equal(t, user, stored)

	// Test duplicate email
	_, err = store.Create("Duplicate Charlie", "charlie@example.com")
	require.Error(t, err)
	var conflictErr *typedhttp.ConflictError
	assert.ErrorAs(t, err, &conflictErr)
}

func TestUserStore_Update(t *testing.T) {
	store := NewUserStore()

	// Test successful update
	updated, err := store.Update("1", "Alice Updated", "")
	require.NoError(t, err)
	assert.Equal(t, "1", updated.ID)
	assert.Equal(t, "Alice Updated", updated.Name)
	assert.Equal(t, "alice@example.com", updated.Email) // Email unchanged

	// Test partial update (email only)
	updated, err = store.Update("1", "", "alice.new@example.com")
	require.NoError(t, err)
	assert.Equal(t, "Alice Updated", updated.Name) // Name unchanged
	assert.Equal(t, "alice.new@example.com", updated.Email)

	// Test non-existing user
	_, err = store.Update("999", "Test", "test@example.com")
	require.Error(t, err)
	var notFoundErr *typedhttp.NotFoundError
	assert.ErrorAs(t, err, &notFoundErr)

	// Test duplicate email
	_, err = store.Update("1", "", "bob@example.com") // Bob's email
	require.Error(t, err)
	var conflictErr *typedhttp.ConflictError
	assert.ErrorAs(t, err, &conflictErr)
}

func TestUserStore_Delete(t *testing.T) {
	store := NewUserStore()

	// Test successful deletion
	err := store.Delete("1")
	require.NoError(t, err)

	// Verify user is gone
	_, exists := store.Get("1")
	assert.False(t, exists)

	// Test deleting non-existing user
	err = store.Delete("999")
	require.Error(t, err)
	var notFoundErr *typedhttp.NotFoundError
	assert.ErrorAs(t, err, &notFoundErr)
}

func TestUserStore_List(t *testing.T) {
	store := NewUserStore()

	// Test basic listing
	users, total := store.List(10, 0, "")
	assert.Len(t, users, 2)
	assert.Equal(t, 2, total)

	// Test pagination
	users, total = store.List(1, 0, "")
	assert.Len(t, users, 1)
	assert.Equal(t, 2, total)

	users, total = store.List(1, 1, "")
	assert.Len(t, users, 1)
	assert.Equal(t, 2, total)

	// Test offset beyond range
	users, total = store.List(10, 5, "")
	assert.Len(t, users, 0)
	assert.Equal(t, 2, total)

	// Test search
	users, total = store.List(10, 0, "Alice")
	assert.Len(t, users, 1)
	assert.Equal(t, 1, total)
	assert.Equal(t, "Alice Smith", users[0].Name)

	users, total = store.List(10, 0, "bob@example.com")
	assert.Len(t, users, 1)
	assert.Equal(t, 1, total)
	assert.Equal(t, "Bob Johnson", users[0].Name)

	// Test search with no results
	users, total = store.List(10, 0, "NonExistent")
	assert.Len(t, users, 0)
	assert.Equal(t, 0, total)
}

// Handler tests
func TestGetUserHandler(t *testing.T) {
	store := NewUserStore()
	handler := &GetUserHandler{store: store}

	// Test successful get
	req := GetUserRequest{ID: "1"}
	resp, err := handler.Handle(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "1", resp.ID)
	assert.Equal(t, "Alice Smith", resp.Name)

	// Test not found
	req = GetUserRequest{ID: "999"}
	_, err = handler.Handle(context.Background(), req)
	require.Error(t, err)
	var notFoundErr *typedhttp.NotFoundError
	assert.ErrorAs(t, err, &notFoundErr)
	assert.Equal(t, "user", notFoundErr.Resource)
	assert.Equal(t, "999", notFoundErr.ID)
}

func TestListUsersHandler(t *testing.T) {
	store := NewUserStore()
	handler := &ListUsersHandler{store: store}

	// Test basic listing
	req := ListUsersRequest{Limit: 10, Offset: 0}
	resp, err := handler.Handle(context.Background(), req)
	require.NoError(t, err)
	assert.Len(t, resp.Users, 2)
	assert.Equal(t, 2, resp.Total)
	assert.Equal(t, 10, resp.Limit)
	assert.Equal(t, 0, resp.Offset)

	// Test with search
	req = ListUsersRequest{Limit: 10, Offset: 0, Search: "Alice"}
	resp, err = handler.Handle(context.Background(), req)
	require.NoError(t, err)
	assert.Len(t, resp.Users, 1)
	assert.Equal(t, 1, resp.Total)
	assert.Equal(t, "Alice Smith", resp.Users[0].Name)
}

func TestCreateUserHandler(t *testing.T) {
	store := NewUserStore()
	handler := &CreateUserHandler{store: store}

	// Test successful creation
	req := CreateUserRequest{
		Name:  "Charlie Brown",
		Email: "charlie@example.com",
	}
	resp, err := handler.Handle(context.Background(), req)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.ID)
	assert.Equal(t, "Charlie Brown", resp.Name)
	assert.Equal(t, "charlie@example.com", resp.Email)

	// Test duplicate email
	req = CreateUserRequest{
		Name:  "Duplicate Alice",
		Email: "alice@example.com", // Already exists
	}
	_, err = handler.Handle(context.Background(), req)
	require.Error(t, err)
	var conflictErr *typedhttp.ConflictError
	assert.ErrorAs(t, err, &conflictErr)
	assert.Contains(t, conflictErr.Message, "email already exists")
}

func TestUpdateUserHandler(t *testing.T) {
	store := NewUserStore()
	handler := &UpdateUserHandler{store: store}

	// Test successful update
	req := UpdateUserRequest{
		ID:   "1",
		Name: "Alice Updated",
	}
	resp, err := handler.Handle(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "1", resp.ID)
	assert.Equal(t, "Alice Updated", resp.Name)
	assert.Equal(t, "alice@example.com", resp.Email) // Unchanged

	// Test not found
	req = UpdateUserRequest{
		ID:   "999",
		Name: "Non Existent",
	}
	_, err = handler.Handle(context.Background(), req)
	require.Error(t, err)
	var notFoundErr *typedhttp.NotFoundError
	assert.ErrorAs(t, err, &notFoundErr)
}

func TestDeleteUserHandler(t *testing.T) {
	store := NewUserStore()
	handler := &DeleteUserHandler{store: store}

	// Test successful deletion
	req := DeleteUserRequest{ID: "1"}
	_, err := handler.Handle(context.Background(), req)
	require.NoError(t, err)

	// Verify user is gone
	_, exists := store.Get("1")
	assert.False(t, exists)

	// Test not found
	req = DeleteUserRequest{ID: "999"}
	_, err = handler.Handle(context.Background(), req)
	require.Error(t, err)
	var notFoundErr *typedhttp.NotFoundError
	assert.ErrorAs(t, err, &notFoundErr)
}

// Benchmark tests
func BenchmarkUserStore_Get(b *testing.B) {
	store := NewUserStore()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.Get("1")
	}
}

func BenchmarkGetUserHandler(b *testing.B) {
	store := NewUserStore()
	handler := &GetUserHandler{store: store}
	req := GetUserRequest{ID: "1"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := handler.Handle(context.Background(), req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCreateUserHandler(b *testing.B) {
	handler := &CreateUserHandler{store: NewUserStore()}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := CreateUserRequest{
			Name:  "Benchmark User",
			Email: fmt.Sprintf("bench%d@example.com", i),
		}
		_, err := handler.Handle(context.Background(), req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Table-driven test example
func TestUserStore_CreateValidation(t *testing.T) {
	tests := []struct {
		name      string
		inputName string
		email     string
		wantErr   bool
		errType   interface{}
	}{
		{
			name:      "valid user",
			inputName: "Valid User",
			email:     "valid@example.com",
			wantErr:   false,
		},
		{
			name:      "duplicate email",
			inputName: "Another Alice",
			email:     "alice@example.com", // Exists in initial data
			wantErr:   true,
			errType:   &typedhttp.ConflictError{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewUserStore()
			_, err := store.Create(tt.inputName, tt.email)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errType != nil {
					assert.ErrorAs(t, err, tt.errType)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
