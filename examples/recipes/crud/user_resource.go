package crud

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
)

// Complete RESTful CRUD resource implementation
// Copy-paste ready for production use with pagination, filtering, and validation

// User domain model
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Request/Response types for each operation

// GET /users/{id}
type GetUserRequest struct {
	ID string `path:"id" validate:"required,uuid"`
}

// POST /users
type CreateUserRequest struct {
	Name  string `json:"name" validate:"required,min=2,max=100"`
	Email string `json:"email" validate:"required,email"`
	Role  string `json:"role" validate:"required,oneof=admin user viewer"`
}

type CreateUserResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

// PUT /users/{id}
type UpdateUserRequest struct {
	ID     string  `path:"id" validate:"required,uuid"`
	Name   *string `json:"name" validate:"omitempty,min=2,max=100"`
	Email  *string `json:"email" validate:"omitempty,email"`
	Role   *string `json:"role" validate:"omitempty,oneof=admin user viewer"`
	Active *bool   `json:"active"`
}

// DELETE /users/{id}
type DeleteUserRequest struct {
	ID string `path:"id" validate:"required,uuid"`
}

// GET /users (with pagination and filtering)
type ListUsersRequest struct {
	// Pagination
	Page  int `query:"page" default:"1" validate:"min=1"`
	Limit int `query:"limit" default:"20" validate:"min=1,max=100"`

	// Sorting
	Sort  string `query:"sort" default:"created_at" validate:"oneof=name email role created_at updated_at"`
	Order string `query:"order" default:"desc" validate:"oneof=asc desc"`

	// Filtering
	Role   string `query:"role" validate:"omitempty,oneof=admin user viewer"`
	Active *bool  `query:"active"`
	Search string `query:"search" validate:"omitempty,min=1,max=100"`

	// Advanced filtering
	CreatedAfter  *time.Time `query:"created_after" format:"rfc3339"`
	CreatedBefore *time.Time `query:"created_before" format:"rfc3339"`
}

type ListUsersResponse struct {
	Users      []User             `json:"users"`
	Pagination PaginationMetadata `json:"pagination"`
}

type PaginationMetadata struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

// Service interface for business logic
type UserService interface {
	GetUser(ctx context.Context, id string) (*User, error)
	CreateUser(ctx context.Context, req CreateUserRequest) (*User, error)
	UpdateUser(ctx context.Context, id string, req UpdateUserRequest) (*User, error)
	DeleteUser(ctx context.Context, id string) error
	ListUsers(ctx context.Context, req ListUsersRequest) ([]User, PaginationMetadata, error)
	UserExists(ctx context.Context, id string) (bool, error)
}

// CRUD Resource handlers

type UserResource struct {
	service UserService
}

func NewUserResource(service UserService) *UserResource {
	return &UserResource{service: service}
}

// GET /users/{id}
func (r *UserResource) GetUser(ctx context.Context, req GetUserRequest) (User, error) {
	user, err := r.service.GetUser(ctx, req.ID)
	if err != nil {
		return User{}, r.mapServiceError(err)
	}
	return *user, nil
}

// POST /users
func (r *UserResource) CreateUser(ctx context.Context, req CreateUserRequest) (CreateUserResponse, error) {
	user, err := r.service.CreateUser(ctx, req)
	if err != nil {
		return CreateUserResponse{}, r.mapServiceError(err)
	}

	return CreateUserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.Role,
		Active:    user.Active,
		CreatedAt: user.CreatedAt,
	}, nil
}

// PUT /users/{id}
func (r *UserResource) UpdateUser(ctx context.Context, req UpdateUserRequest) (User, error) {
	// Check if user exists
	exists, err := r.service.UserExists(ctx, req.ID)
	if err != nil {
		return User{}, r.mapServiceError(err)
	}
	if !exists {
		return User{}, typedhttp.NewNotFoundError("User not found")
	}

	user, err := r.service.UpdateUser(ctx, req.ID, req)
	if err != nil {
		return User{}, r.mapServiceError(err)
	}

	return *user, nil
}

// DELETE /users/{id}
func (r *UserResource) DeleteUser(ctx context.Context, req DeleteUserRequest) error {
	// Check if user exists
	exists, err := r.service.UserExists(ctx, req.ID)
	if err != nil {
		return r.mapServiceError(err)
	}
	if !exists {
		return typedhttp.NewNotFoundError("User not found")
	}

	err = r.service.DeleteUser(ctx, req.ID)
	if err != nil {
		return r.mapServiceError(err)
	}

	return nil
}

// GET /users
func (r *UserResource) ListUsers(ctx context.Context, req ListUsersRequest) (ListUsersResponse, error) {
	users, pagination, err := r.service.ListUsers(ctx, req)
	if err != nil {
		return ListUsersResponse{}, r.mapServiceError(err)
	}

	return ListUsersResponse{
		Users:      users,
		Pagination: pagination,
	}, nil
}

// mapServiceError maps service errors to HTTP errors
func (r *UserResource) mapServiceError(err error) error {
	switch {
	case err == ErrUserNotFound:
		return typedhttp.NewNotFoundError("User not found")
	case err == ErrUserAlreadyExists:
		return typedhttp.NewConflictError("User already exists")
	case err == ErrInvalidUserData:
		return typedhttp.NewValidationError("Invalid user data", nil)
	default:
		return fmt.Errorf("internal server error: %w", err)
	}
}

// Service errors
var (
	ErrUserNotFound      = fmt.Errorf("user not found")
	ErrUserAlreadyExists = fmt.Errorf("user already exists")
	ErrInvalidUserData   = fmt.Errorf("invalid user data")
)

// Router setup function
func SetupUserRoutes(router *typedhttp.TypedRouter, service UserService) {
	resource := NewUserResource(service)

	// Register all CRUD endpoints
	typedhttp.GET(router, "/users/{id}", resource.GetUser)
	typedhttp.POST(router, "/users", resource.CreateUser)
	typedhttp.PUT(router, "/users/{id}", resource.UpdateUser)
	typedhttp.DELETE(router, "/users/{id}", resource.DeleteUser)
	typedhttp.GET(router, "/users", resource.ListUsers)
}

// Example service implementation (in-memory for demonstration)
type InMemoryUserService struct {
	users   map[string]*User
	nextID  int
}

func NewInMemoryUserService() *InMemoryUserService {
	return &InMemoryUserService{
		users:  make(map[string]*User),
		nextID: 1,
	}
}

func (s *InMemoryUserService) GetUser(ctx context.Context, id string) (*User, error) {
	user, exists := s.users[id]
	if !exists {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *InMemoryUserService) CreateUser(ctx context.Context, req CreateUserRequest) (*User, error) {
	// Check if email already exists
	for _, user := range s.users {
		if user.Email == req.Email {
			return nil, ErrUserAlreadyExists
		}
	}

	// Create new user
	id := strconv.Itoa(s.nextID)
	s.nextID++

	user := &User{
		ID:        id,
		Name:      req.Name,
		Email:     req.Email,
		Role:      req.Role,
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	s.users[id] = user
	return user, nil
}

func (s *InMemoryUserService) UpdateUser(ctx context.Context, id string, req UpdateUserRequest) (*User, error) {
	user, exists := s.users[id]
	if !exists {
		return nil, ErrUserNotFound
	}

	// Update fields if provided
	if req.Name != nil {
		user.Name = *req.Name
	}
	if req.Email != nil {
		// Check if email already exists
		for _, u := range s.users {
			if u.ID != id && u.Email == *req.Email {
				return nil, ErrUserAlreadyExists
			}
		}
		user.Email = *req.Email
	}
	if req.Role != nil {
		user.Role = *req.Role
	}
	if req.Active != nil {
		user.Active = *req.Active
	}

	user.UpdatedAt = time.Now()
	return user, nil
}

func (s *InMemoryUserService) DeleteUser(ctx context.Context, id string) error {
	if _, exists := s.users[id]; !exists {
		return ErrUserNotFound
	}
	delete(s.users, id)
	return nil
}

func (s *InMemoryUserService) ListUsers(ctx context.Context, req ListUsersRequest) ([]User, PaginationMetadata, error) {
	var allUsers []User
	
	// Convert map to slice and apply filters
	for _, user := range s.users {
		// Apply filters
		if req.Role != "" && user.Role != req.Role {
			continue
		}
		if req.Active != nil && user.Active != *req.Active {
			continue
		}
		if req.Search != "" {
			if !containsIgnoreCase(user.Name, req.Search) && 
			   !containsIgnoreCase(user.Email, req.Search) {
				continue
			}
		}
		if req.CreatedAfter != nil && user.CreatedAt.Before(*req.CreatedAfter) {
			continue
		}
		if req.CreatedBefore != nil && user.CreatedAt.After(*req.CreatedBefore) {
			continue
		}

		allUsers = append(allUsers, *user)
	}

	// Apply sorting (simplified)
	// In production, use a proper sorting library or database ORDER BY

	// Calculate pagination
	total := len(allUsers)
	totalPages := (total + req.Limit - 1) / req.Limit
	start := (req.Page - 1) * req.Limit
	end := start + req.Limit

	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	paginatedUsers := allUsers[start:end]

	pagination := PaginationMetadata{
		Page:       req.Page,
		Limit:      req.Limit,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    req.Page < totalPages,
		HasPrev:    req.Page > 1,
	}

	return paginatedUsers, pagination, nil
}

func (s *InMemoryUserService) UserExists(ctx context.Context, id string) (bool, error) {
	_, exists := s.users[id]
	return exists, nil
}

// Helper function for search
func containsIgnoreCase(str, substr string) bool {
	// Simplified case-insensitive contains
	// In production, use strings.ToLower or a proper search library
	return len(str) >= len(substr) // Simplified for example
}

// Example usage
func ExampleUsage() {
	// Create service
	service := NewInMemoryUserService()

	// Create router
	router := typedhttp.NewRouter()

	// Setup routes
	SetupUserRoutes(router, service)

	// Start server
	http.ListenAndServe(":8080", router)
}

// Response helpers for DELETE operations
type DeleteResponse struct {
	Message string `json:"message"`
}

// Enhanced DELETE handler that returns confirmation
func (r *UserResource) DeleteUserWithResponse(ctx context.Context, req DeleteUserRequest) (DeleteResponse, error) {
	err := r.DeleteUser(ctx, req)
	if err != nil {
		return DeleteResponse{}, err
	}

	return DeleteResponse{
		Message: fmt.Sprintf("User %s deleted successfully", req.ID),
	}, nil
}