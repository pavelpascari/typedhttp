package handlers

import (
	"context"
	"time"

	"github.com/pavelpascari/typedhttp/examples/comprehensive-architecture/internal/models"
)

// UserHandler handles user-related operations
type UserHandler struct {
	// In a real application, this would have dependencies like:
	// repo UserRepository
	// logger *slog.Logger
	// validator *validator.Validate
}

// GetUserHandler implements the TypedHTTP Handler interface for GetUser
type GetUserHandler struct {
	handler *UserHandler
}

// CreateUserHandler implements the TypedHTTP Handler interface for CreateUser
type CreateUserHandler struct {
	handler *UserHandler
}

// ListUsersHandler implements the TypedHTTP Handler interface for ListUsers
type ListUsersHandler struct {
	handler *UserHandler
}

// NewUserHandler creates a new user handler
func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

// NewGetUserHandler creates a new GetUser handler
func NewGetUserHandler() *GetUserHandler {
	return &GetUserHandler{handler: NewUserHandler()}
}

// NewCreateUserHandler creates a new CreateUser handler
func NewCreateUserHandler() *CreateUserHandler {
	return &CreateUserHandler{handler: NewUserHandler()}
}

// NewListUsersHandler creates a new ListUsers handler
func NewListUsersHandler() *ListUsersHandler {
	return &ListUsersHandler{handler: NewUserHandler()}
}

// Handle implements the TypedHTTP Handler interface for GetUser
func (h *GetUserHandler) Handle(ctx context.Context, req models.GetUserRequest) (models.GetUserResponse, error) {
	return h.handler.GetUser(ctx, req)
}

// GetUser implements the business logic for getting a user
func (h *UserHandler) GetUser(ctx context.Context, req models.GetUserRequest) (models.GetUserResponse, error) {
	// Simulate user lookup
	user := models.User{
		ID:        req.ID,
		Name:      "John Doe",
		Email:     "john.doe@example.com",
		Role:      "user",
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now(),
	}

	return models.GetUserResponse{User: user}, nil
}

// Handle implements the TypedHTTP Handler interface for CreateUser
func (h *CreateUserHandler) Handle(ctx context.Context, req models.CreateUserRequest) (models.CreateUserResponse, error) {
	return h.handler.CreateUser(ctx, req)
}

// CreateUser implements the business logic for user creation
func (h *UserHandler) CreateUser(ctx context.Context, req models.CreateUserRequest) (models.CreateUserResponse, error) {
	// Simulate user creation
	user := models.User{
		ID:        "550e8400-e29b-41d4-a716-446655440000",
		Name:      req.Name,
		Email:     req.Email,
		Role:      req.Role,
		CreatedAt: time.Now(),
	}

	return models.CreateUserResponse{
		User:    user,
		Message: "User created successfully",
	}, nil
}

// Handle implements the TypedHTTP Handler interface for ListUsers
func (h *ListUsersHandler) Handle(ctx context.Context, req models.ListUsersRequest) (models.ListUsersResponse, error) {
	return h.handler.ListUsers(ctx, req)
}

// ListUsers implements the business logic for user listing with pagination and filtering
func (h *UserHandler) ListUsers(ctx context.Context, req models.ListUsersRequest) (models.ListUsersResponse, error) {
	// Simulate user listing
	users := []models.User{
		{
			ID:        "550e8400-e29b-41d4-a716-446655440000",
			Name:      "John Doe",
			Email:     "john@example.com",
			Role:      "user",
			CreatedAt: time.Now().Add(-24 * time.Hour),
		},
		{
			ID:        "550e8400-e29b-41d4-a716-446655440001",
			Name:      "Jane Smith",
			Email:     "jane@example.com",
			Role:      "admin",
			CreatedAt: time.Now().Add(-48 * time.Hour),
		},
	}

	// Apply filters
	if req.Role != "" {
		filtered := []models.User{}
		for _, user := range users {
			if user.Role == req.Role {
				filtered = append(filtered, user)
			}
		}
		users = filtered
	}

	return models.ListUsersResponse{
		Users: users,
		Total: len(users),
		Page:  req.Page,
		Limit: req.Limit,
	}, nil
}