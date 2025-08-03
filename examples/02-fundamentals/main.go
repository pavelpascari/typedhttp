package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
)

// Domain model
type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Request/Response types with validation
type GetUserRequest struct {
	ID string `path:"id" validate:"required"`
}

type ListUsersRequest struct {
	Limit  int    `query:"limit" validate:"omitempty,min=1,max=100" default:"10"`
	Offset int    `query:"offset" validate:"omitempty,min=0" default:"0"`
	Search string `query:"search" validate:"omitempty,min=1,max=50"`
}

type ListUsersResponse struct {
	Users  []User `json:"users"`
	Total  int    `json:"total"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

type CreateUserRequest struct {
	Name  string `json:"name" validate:"required,min=2,max=50"`
	Email string `json:"email" validate:"required,email"`
}

type UpdateUserRequest struct {
	ID    string `path:"id" validate:"required"`
	Name  string `json:"name" validate:"omitempty,min=2,max=50"`
	Email string `json:"email" validate:"omitempty,email"`
}

type DeleteUserRequest struct {
	ID string `path:"id" validate:"required"`
}

// In-memory storage with thread safety
type UserStore struct {
	mu     sync.RWMutex
	users  map[string]User
	nextID int
}

func NewUserStore() *UserStore {
	return &UserStore{
		users: map[string]User{
			"1": {ID: "1", Name: "Alice Smith", Email: "alice@example.com"},
			"2": {ID: "2", Name: "Bob Johnson", Email: "bob@example.com"},
		},
		nextID: 3,
	}
}

func (s *UserStore) Get(id string) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, exists := s.users[id]
	return user, exists
}

func (s *UserStore) List(limit, offset int, search string) ([]User, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var all []User
	for _, user := range s.users {
		if search == "" || contains(user.Name, search) || contains(user.Email, search) {
			all = append(all, user)
		}
	}

	total := len(all)
	start := offset
	end := offset + limit

	if start > total {
		return []User{}, total
	}
	if end > total {
		end = total
	}

	return all[start:end], total
}

func (s *UserStore) Create(name, email string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicate email
	for _, user := range s.users {
		if user.Email == email {
			return User{}, typedhttp.NewConflictError("user with this email already exists")
		}
	}

	id := strconv.Itoa(s.nextID)
	s.nextID++

	user := User{
		ID:    id,
		Name:  name,
		Email: email,
	}
	s.users[id] = user

	return user, nil
}

func (s *UserStore) Update(id string, name, email string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[id]
	if !exists {
		return User{}, typedhttp.NewNotFoundError("user", id)
	}

	// Check for duplicate email (excluding current user)
	if email != "" && email != user.Email {
		for _, u := range s.users {
			if u.ID != id && u.Email == email {
				return User{}, typedhttp.NewConflictError("user with this email already exists")
			}
		}
	}

	// Update fields
	if name != "" {
		user.Name = name
	}
	if email != "" {
		user.Email = email
	}

	s.users[id] = user
	return user, nil
}

func (s *UserStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[id]; !exists {
		return typedhttp.NewNotFoundError("user", id)
	}

	delete(s.users, id)
	return nil
}

// Helper function
func contains(str, substr string) bool {
	return len(str) >= len(substr) && str[:len(substr)] == substr
}

// Handlers
type UserHandlers struct {
	store *UserStore
}

func NewUserHandlers(store *UserStore) *UserHandlers {
	return &UserHandlers{store: store}
}

type GetUserHandler struct {
	store *UserStore
}

func (h *GetUserHandler) Handle(ctx context.Context, req GetUserRequest) (User, error) {
	user, exists := h.store.Get(req.ID)
	if !exists {
		return User{}, typedhttp.NewNotFoundError("user", req.ID)
	}
	return user, nil
}

type ListUsersHandler struct {
	store *UserStore
}

func (h *ListUsersHandler) Handle(ctx context.Context, req ListUsersRequest) (ListUsersResponse, error) {
	users, total := h.store.List(req.Limit, req.Offset, req.Search)
	
	return ListUsersResponse{
		Users:  users,
		Total:  total,
		Limit:  req.Limit,
		Offset: req.Offset,
	}, nil
}

type CreateUserHandler struct {
	store *UserStore
}

func (h *CreateUserHandler) Handle(ctx context.Context, req CreateUserRequest) (User, error) {
	return h.store.Create(req.Name, req.Email)
}

type UpdateUserHandler struct {
	store *UserStore
}

func (h *UpdateUserHandler) Handle(ctx context.Context, req UpdateUserRequest) (User, error) {
	return h.store.Update(req.ID, req.Name, req.Email)
}

type DeleteUserHandler struct {
	store *UserStore
}

func (h *DeleteUserHandler) Handle(ctx context.Context, req DeleteUserRequest) (struct{}, error) {
	err := h.store.Delete(req.ID)
	return struct{}{}, err
}

func main() {
	// Initialize store
	store := NewUserStore()

	// Create handlers
	getUserHandler := &GetUserHandler{store: store}
	listUsersHandler := &ListUsersHandler{store: store}
	createUserHandler := &CreateUserHandler{store: store}
	updateUserHandler := &UpdateUserHandler{store: store}
	deleteUserHandler := &DeleteUserHandler{store: store}

	// Setup router
	router := typedhttp.NewRouter()

	// Register routes with OpenAPI documentation
	typedhttp.GET(router, "/users", listUsersHandler,
		typedhttp.WithTags("users"),
		typedhttp.WithSummary("List users"),
		typedhttp.WithDescription("List users with optional search, pagination"),
	)

	typedhttp.GET(router, "/users/{id}", getUserHandler,
		typedhttp.WithTags("users"),
		typedhttp.WithSummary("Get user by ID"),
		typedhttp.WithDescription("Retrieve a specific user by their ID"),
	)

	typedhttp.POST(router, "/users", createUserHandler,
		typedhttp.WithTags("users"),
		typedhttp.WithSummary("Create user"),
		typedhttp.WithDescription("Create a new user with name and email"),
	)

	typedhttp.PUT(router, "/users/{id}", updateUserHandler,
		typedhttp.WithTags("users"),
		typedhttp.WithSummary("Update user"),
		typedhttp.WithDescription("Update user name and/or email"),
	)

	typedhttp.DELETE(router, "/users/{id}", deleteUserHandler,
		typedhttp.WithTags("users"),
		typedhttp.WithSummary("Delete user"),
		typedhttp.WithDescription("Delete a user by ID"),
	)

	// Print route information
	fmt.Println("🚀 TypedHTTP Fundamentals Example")
	fmt.Println("=================================")
	fmt.Println()
	fmt.Printf("📍 Server running on http://localhost:8080\n")
	fmt.Println()
	fmt.Println("📚 Available endpoints:")
	for _, handler := range router.GetHandlers() {
		fmt.Printf("  %-6s %-20s - %s\n", handler.Method, handler.Path, handler.Metadata.Summary)
	}
	fmt.Println()
	fmt.Println("🧪 Try these commands:")
	fmt.Println("  # List users")
	fmt.Println("  curl http://localhost:8080/users")
	fmt.Println("  curl http://localhost:8080/users?limit=1&search=Alice")
	fmt.Println()
	fmt.Println("  # Get specific user")
	fmt.Println("  curl http://localhost:8080/users/1")
	fmt.Println()
	fmt.Println("  # Create user")
	fmt.Println(`  curl -X POST http://localhost:8080/users \`)
	fmt.Println(`    -H "Content-Type: application/json" \`)
	fmt.Println(`    -d '{"name":"Jane Doe","email":"jane@example.com"}'`)
	fmt.Println()
	fmt.Println("  # Update user")
	fmt.Println(`  curl -X PUT http://localhost:8080/users/1 \`)
	fmt.Println(`    -H "Content-Type: application/json" \`)
	fmt.Println(`    -d '{"name":"Alice Updated"}'`)
	fmt.Println()
	fmt.Println("  # Delete user")
	fmt.Println("  curl -X DELETE http://localhost:8080/users/2")
	fmt.Println()
	fmt.Println("✨ Features demonstrated:")
	fmt.Println("  ✓ Complete CRUD operations")
	fmt.Println("  ✓ Request validation with struct tags")
	fmt.Println("  ✓ Query parameters (pagination, search)")
	fmt.Println("  ✓ Proper error handling with typed errors")
	fmt.Println("  ✓ Thread-safe in-memory storage")
	fmt.Println("  ✓ Automatic OpenAPI documentation")

	log.Fatal(http.ListenAndServe(":8080", router))
}