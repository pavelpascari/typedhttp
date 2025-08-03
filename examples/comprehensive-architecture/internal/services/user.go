package services

import (
	"context"
	"time"

	"github.com/pavelpascari/typedhttp/examples/comprehensive-architecture/internal/models"
	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
)

// UserService implements CRUD operations for users.
// This replaces all the individual handler wrappers with a single service interface.
type UserService struct {
	// In a real application, this would have dependencies like:
	// repo UserRepository
	// logger *slog.Logger
	// validator *validator.Validate
}

// NewUserService creates a new user service
func NewUserService() *UserService {
	return &UserService{}
}

// Verify that UserService implements the CRUDService interface
var _ typedhttp.CRUDService[
	models.GetUserRequest,
	models.GetUserResponse,
	models.ListUsersRequest,
	models.ListUsersResponse,
	models.CreateUserRequest,
	models.CreateUserResponse,
	models.UpdateUserRequest,
	models.UpdateUserResponse,
	models.DeleteUserRequest,
	models.DeleteUserResponse,
] = (*UserService)(nil)

// Get implements the business logic for getting a user by ID
func (s *UserService) Get(ctx context.Context, req models.GetUserRequest) (models.GetUserResponse, error) {
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

// List implements the business logic for listing users with pagination and filtering
func (s *UserService) List(ctx context.Context, req models.ListUsersRequest) (models.ListUsersResponse, error) {
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

// Create implements the business logic for user creation
func (s *UserService) Create(ctx context.Context, req models.CreateUserRequest) (models.CreateUserResponse, error) {
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

// Update implements the business logic for user updates
func (s *UserService) Update(ctx context.Context, req models.UpdateUserRequest) (models.UpdateUserResponse, error) {
	// Simulate user update
	user := models.User{
		ID:        req.ID,
		Name:      "John Doe", // Would be loaded from storage
		Email:     "john.doe@example.com",
		Role:      "user",
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now(),
	}

	// Apply updates
	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Role != "" {
		user.Role = req.Role
	}

	return models.UpdateUserResponse{
		User:    user,
		Message: "User updated successfully",
	}, nil
}

// Delete implements the business logic for user deletion
func (s *UserService) Delete(ctx context.Context, req models.DeleteUserRequest) (models.DeleteUserResponse, error) {
	// Simulate user deletion
	// In a real app, this would check if user exists and delete from storage

	return models.DeleteUserResponse{
		Message: "User deleted successfully",
	}, nil
}
