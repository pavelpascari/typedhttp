package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
)

// GetUserRequest Example request and response types
type GetUserRequest struct {
	ID string `path:"id" validate:"required"`
}

type GetUserResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Message string `json:"message"`
}

type CreateUserRequest struct {
	Name  string `json:"name" validate:"required,min=2,max=50"`
	Email string `json:"email" validate:"required,email"`
}

type CreateUserResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Message string `json:"message"`
}

// Example handlers
type GetUserHandler struct{}

func (h *GetUserHandler) Handle(ctx context.Context, req GetUserRequest) (GetUserResponse, error) {
	// Simulate finding a user
	if req.ID == "not-found" {
		return GetUserResponse{}, typedhttp.NewNotFoundError("user", req.ID)
	}

	return GetUserResponse{
		ID:      req.ID,
		Name:    "John Doe",
		Email:   "john@example.com",
		Message: fmt.Sprintf("Found user with ID: %s", req.ID),
	}, nil
}

type CreateUserHandler struct{}

func (h *CreateUserHandler) Handle(ctx context.Context, req CreateUserRequest) (CreateUserResponse, error) {
	// Simulate creating a user
	if req.Email == "duplicate@example.com" {
		return CreateUserResponse{}, typedhttp.NewConflictError("User with this email already exists")
	}

	return CreateUserResponse{
		ID:      "user_12345",
		Name:    req.Name,
		Email:   req.Email,
		Message: "User created successfully",
	}, nil
}

func main() {
	// Create a new typed router
	router := typedhttp.NewRouter()

	// Create handlers
	getUserHandler := &GetUserHandler{}
	createUserHandler := &CreateUserHandler{}

	// Register handlers with different configurations
	typedhttp.GET(router, "/users/{id}", getUserHandler,
		typedhttp.WithTags("users"),
		typedhttp.WithSummary("Get user by ID"),
		typedhttp.WithDescription("Retrieves a user by their unique identifier"),
		typedhttp.WithDefaultObservability(),
	)

	typedhttp.POST(router, "/users", createUserHandler,
		typedhttp.WithTags("users"),
		typedhttp.WithSummary("Create a new user"),
		typedhttp.WithDescription("Creates a new user with the provided information"),
		typedhttp.WithDefaultObservability(),
		typedhttp.WithErrorMapper(&typedhttp.DefaultErrorMapper{}),
	)

	// Print registered handlers for demonstration
	fmt.Println("Registered handlers:")
	for _, handler := range router.GetHandlers() {
		fmt.Printf("  %s %s - %s\n", handler.Method, handler.Path, handler.Metadata.Summary)
	}

	fmt.Println("\nStarting server on :8080")
	fmt.Println("Try these endpoints:")
	fmt.Println("  GET  http://localhost:8080/users/123")
	fmt.Println("  GET  http://localhost:8080/users/not-found")
	fmt.Println("  POST http://localhost:8080/users -d '{\"name\":\"Jane\",\"email\":\"jane@example.com\"}'")
	fmt.Println("  POST http://localhost:8080/users -d '{\"name\":\"Bob\",\"email\":\"duplicate@example.com\"}'")

	log.Fatal(http.ListenAndServe(":8080", router))
}
