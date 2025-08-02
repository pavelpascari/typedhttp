package typedhttp

import (
	"context"
	"testing"
	"time"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types for resource pattern testing
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type GetUserRequest struct {
	ID string `path:"id" validate:"required"`
}

type GetUserResponse struct {
	User User `json:"user"`
}

type ListUsersRequest struct {
	Page  int `query:"page" default:"1"`
	Limit int `query:"limit" default:"10"`
}

type ListUsersResponse struct {
	Users []User `json:"users"`
	Total int    `json:"total"`
}

type CreateUserRequest struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
}

type CreateUserResponse struct {
	User    User   `json:"user"`
	Message string `json:"message"`
}

type UpdateUserRequest struct {
	ID    string `path:"id" validate:"required"`
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

type UpdateUserResponse struct {
	User    User   `json:"user"`
	Message string `json:"message"`
}

type DeleteUserRequest struct {
	ID string `path:"id" validate:"required"`
}

type DeleteUserResponse struct {
	Message string `json:"message"`
}

// Mock user service for testing
type mockUserService struct {
	users map[string]User
}

func newMockUserService() *mockUserService {
	return &mockUserService{
		users: map[string]User{
			"1": {
				ID:        "1",
				Name:      "John Doe",
				Email:     "john@example.com",
				CreatedAt: time.Now(),
			},
		},
	}
}

func (s *mockUserService) Get(ctx context.Context, req GetUserRequest) (GetUserResponse, error) {
	user, exists := s.users[req.ID]
	if !exists {
		return GetUserResponse{}, NewNotFoundError("user", req.ID)
	}
	return GetUserResponse{User: user}, nil
}

func (s *mockUserService) List(ctx context.Context, req ListUsersRequest) (ListUsersResponse, error) {
	var users []User
	for _, user := range s.users {
		users = append(users, user)
	}
	return ListUsersResponse{
		Users: users,
		Total: len(users),
	}, nil
}

func (s *mockUserService) Create(ctx context.Context, req CreateUserRequest) (CreateUserResponse, error) {
	user := User{
		ID:        "2",
		Name:      req.Name,
		Email:     req.Email,
		CreatedAt: time.Now(),
	}
	s.users[user.ID] = user
	return CreateUserResponse{
		User:    user,
		Message: "User created successfully",
	}, nil
}

func (s *mockUserService) Update(ctx context.Context, req UpdateUserRequest) (UpdateUserResponse, error) {
	user, exists := s.users[req.ID]
	if !exists {
		return UpdateUserResponse{}, NewNotFoundError("user", req.ID)
	}
	
	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	
	s.users[req.ID] = user
	return UpdateUserResponse{
		User:    user,
		Message: "User updated successfully",
	}, nil
}

func (s *mockUserService) Delete(ctx context.Context, req DeleteUserRequest) (DeleteUserResponse, error) {
	_, exists := s.users[req.ID]
	if !exists {
		return DeleteUserResponse{}, NewNotFoundError("user", req.ID)
	}
	
	delete(s.users, req.ID)
	return DeleteUserResponse{
		Message: "User deleted successfully",
	}, nil
}

func TestDomainRouter_Creation(t *testing.T) {
	// Test basic domain router creation
	router := NewDomainRouter("/api/v1")
	assert.NotNil(t, router)
	assert.Equal(t, "/api/v1", router.pathPrefix)
	assert.NotNil(t, router.TypedRouter)
}

func TestDomainRouter_WithMiddleware(t *testing.T) {
	// Test domain router creation with middleware
	middleware := []MiddlewareEntry{
		{
			Middleware: NewResponseEnvelopeMiddleware[any](),
			Config: MiddlewareConfig{
				Name:     "envelope",
				Priority: 90,
			},
		},
	}
	
	router := NewDomainRouter("/api/v1", middleware...)
	assert.NotNil(t, router)
	assert.Equal(t, "/api/v1", router.pathPrefix)
	assert.Len(t, router.middleware, 1)
}

func TestResource_Registration(t *testing.T) {
	// Test that resource registration creates the expected handlers
	router := NewDomainRouter("/api/v1")
	service := newMockUserService()
	
	config := ResourceConfig{
		Tags: []string{"users"},
		Operations: map[string]OperationConfig{
			"GET": {
				Summary: "Get user by ID",
				Enabled: true,
			},
			"LIST": {
				Summary: "List all users",
				Enabled: true,
			},
			"POST": {
				Summary: "Create a new user",
				Enabled: true,
			},
			"PUT": {
				Summary: "Update user",
				Enabled: true,
			},
			"DELETE": {
				Summary: "Delete user",
				Enabled: true,
			},
		},
	}
	
	Resource(router, "/users", service, config)
	
	// Check that handlers were registered
	handlers := router.GetHandlers()
	require.Len(t, handlers, 5) // GET, LIST, POST, PUT, DELETE
	
	// Verify path generation and method distribution
	pathMethodCounts := make(map[string]map[string]int)
	for _, handler := range handlers {
		if pathMethodCounts[handler.Path] == nil {
			pathMethodCounts[handler.Path] = make(map[string]int)
		}
		pathMethodCounts[handler.Path][handler.Method]++
		
		// Check that metadata is properly set
		assert.Contains(t, handler.Metadata.Tags, "users")
		assert.NotEmpty(t, handler.Metadata.Summary)
	}
	
	// Verify we have the expected path/method combinations
	// Collection path: /api/v1/users - should have GET (list) and POST (create)
	collectionPath := "/api/v1/users"
	assert.Contains(t, pathMethodCounts, collectionPath)
	assert.Equal(t, 1, pathMethodCounts[collectionPath]["GET"])  // LIST operation
	assert.Equal(t, 1, pathMethodCounts[collectionPath]["POST"]) // CREATE operation
	
	// Item path: /api/v1/users/{id} - should have GET (item), PUT, DELETE
	itemPath := "/api/v1/users/{id}"
	assert.Contains(t, pathMethodCounts, itemPath)
	assert.Equal(t, 1, pathMethodCounts[itemPath]["GET"])    // GET item operation
	assert.Equal(t, 1, pathMethodCounts[itemPath]["PUT"])    // UPDATE operation
	assert.Equal(t, 1, pathMethodCounts[itemPath]["DELETE"]) // DELETE operation
}

func TestResource_SelectiveOperations(t *testing.T) {
	// Test that only enabled operations are registered
	router := NewDomainRouter("/api/v1")
	service := newMockUserService()
	
	config := ResourceConfig{
		Tags: []string{"users"},
		Operations: map[string]OperationConfig{
			"GET": {
				Summary: "Get user by ID",
				Enabled: true,
			},
			"LIST": {
				Summary: "List all users", 
				Enabled: true,
			},
			"POST": {
				Enabled: false, // Disabled
			},
			"PUT": {
				Enabled: false, // Disabled
			},
			"DELETE": {
				Enabled: false, // Disabled
			},
		},
	}
	
	Resource(router, "/users", service, config)
	
	// Should only have 2 handlers (GET and LIST)
	handlers := router.GetHandlers()
	assert.Len(t, handlers, 2)
	
	// Verify only GET and LIST operations
	methods := make(map[string]bool)
	for _, handler := range handlers {
		methods[handler.Method] = true
	}
	
	assert.True(t, methods["GET"])
	assert.False(t, methods["POST"])
	assert.False(t, methods["PUT"])
	assert.False(t, methods["DELETE"])
}

func TestResource_DefaultConfiguration(t *testing.T) {
	// Test resource registration with minimal configuration
	router := NewDomainRouter("/api/v1")
	service := newMockUserService()
	
	config := ResourceConfig{
		Tags: []string{"users"},
	}
	
	Resource(router, "/users", service, config)
	
	// All operations should be enabled by default
	handlers := router.GetHandlers()
	assert.Len(t, handlers, 5)
	
	// Check that default summaries are generated
	for _, handler := range handlers {
		assert.NotEmpty(t, handler.Metadata.Summary)
		assert.Contains(t, handler.Metadata.Tags, "users")
	}
}

func TestResourceHandler_MethodDelegation(t *testing.T) {
	// Test that the resource handler correctly delegates to service methods
	service := newMockUserService()
	
	// Test Get operation
	getHandler := &resourceHandler[GetUserRequest, GetUserResponse]{
		service:   service,
		operation: "Get",
	}
	
	ctx := context.Background()
	req := GetUserRequest{ID: "1"}
	
	resp, err := getHandler.Handle(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "1", resp.User.ID)
	assert.Equal(t, "John Doe", resp.User.Name)
	
	// Test Create operation
	createHandler := &resourceHandler[CreateUserRequest, CreateUserResponse]{
		service:   service,
		operation: "Create",
	}
	
	createReq := CreateUserRequest{
		Name:  "Jane Doe",
		Email: "jane@example.com",
	}
	
	createResp, err := createHandler.Handle(ctx, createReq)
	require.NoError(t, err)
	assert.Equal(t, "Jane Doe", createResp.User.Name)
	assert.Equal(t, "jane@example.com", createResp.User.Email)
	assert.Equal(t, "User created successfully", createResp.Message)
}

func TestResourceHandler_ErrorHandling(t *testing.T) {
	// Test error handling in resource handler
	service := newMockUserService()
	
	handler := &resourceHandler[GetUserRequest, GetUserResponse]{
		service:   service,
		operation: "Get",
	}
	
	ctx := context.Background()
	req := GetUserRequest{ID: "nonexistent"}
	
	_, err := handler.Handle(ctx, req)
	require.Error(t, err)
	
	// Should be a NotFoundError
	var notFoundErr *NotFoundError
	assert.ErrorAs(t, err, &notFoundErr)
	assert.Equal(t, "user", notFoundErr.Resource)
	assert.Equal(t, "nonexistent", notFoundErr.ID)
}

func TestResourceHandler_InvalidMethod(t *testing.T) {
	// Test behavior with invalid method name
	service := newMockUserService()
	
	handler := &resourceHandler[GetUserRequest, GetUserResponse]{
		service:   service,
		operation: "NonexistentMethod",
	}
	
	ctx := context.Background()
	req := GetUserRequest{ID: "1"}
	
	_, err := handler.Handle(ctx, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "method NonexistentMethod not found")
}

func TestBuildOperationOptions(t *testing.T) {
	// Test operation options building
	config := ResourceConfig{
		Tags: []string{"users"},
		Operations: map[string]OperationConfig{
			"GET": {
				Summary:     "Custom get summary",
				Description: "Custom description",
				Tags:        []string{"custom"},
			},
		},
	}
	
	// Test with existing operation config
	opts := buildOperationOptions("GET", config)
	assert.Len(t, opts, 4) // base tags + summary + description + operation tags
	
	// Test with default operation (no config)
	opts = buildOperationOptions("POST", config)
	assert.Len(t, opts, 2) // base tags + default summary
}

func TestInferResourceName(t *testing.T) {
	// Test resource name inference
	config := ResourceConfig{
		Tags: []string{"users"},
	}
	
	name := inferResourceName(config)
	assert.Equal(t, "user", name) // Should remove trailing 's'
	
	// Test with no tags
	config = ResourceConfig{}
	name = inferResourceName(config)
	assert.Equal(t, "resource", name) // Should use default
}