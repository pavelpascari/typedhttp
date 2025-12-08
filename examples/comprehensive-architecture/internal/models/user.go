package models

import "time"

// User represents a user in the system
type User struct {
	ID        string    `json:"id" validate:"required,uuid"`
	Name      string    `json:"name" validate:"required,min=2,max=50"`
	Email     string    `json:"email" validate:"required,email"`
	Role      string    `json:"role" validate:"required,oneof=admin user"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// GetUserRequest represents the request to get a user by ID
type GetUserRequest struct {
	ID string `path:"id" validate:"required,uuid"`
}

// GetUserResponse represents the response when getting a user
type GetUserResponse struct {
	User User `json:"user"`
}

// CreateUserRequest represents the request to create a new user
type CreateUserRequest struct {
	//openapi:description=User's full name,example=John Doe
	Name string `json:"name" validate:"required,min=2,max=50"`

	//openapi:description=User's email address,example=john@example.com
	Email string `json:"email" validate:"required,email"`

	//openapi:description=User's role in the system,example=user
	Role string `json:"role" validate:"required,oneof=admin user"`
}

// CreateUserResponse represents the response when creating a user
type CreateUserResponse struct {
	User    User   `json:"user"`
	Message string `json:"message"`
}

// ListUsersRequest represents the request to list users with pagination and filtering
type ListUsersRequest struct {
	//openapi:description=Page number for pagination,example=1
	Page int `query:"page" default:"1" validate:"min=1,max=1000"`

	//openapi:description=Number of items per page,example=20
	Limit int `query:"limit" default:"20" validate:"min=1,max=100"`

	//openapi:description=Filter by user role,example=user
	Role string `query:"role" validate:"omitempty,oneof=admin user"`

	//openapi:description=Sort field,example=created_at
	Sort string `query:"sort" default:"created_at" validate:"oneof=created_at name email"`

	//openapi:description=Sort direction,example=desc
	Order string `query:"order" default:"desc" validate:"oneof=asc desc"`
}

// ListUsersResponse represents the response when listing users
type ListUsersResponse struct {
	Users []User `json:"users"`
	Total int    `json:"total"`
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
}
