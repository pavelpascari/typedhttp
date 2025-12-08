package models

// UpdateUserRequest represents the request to update a user
type UpdateUserRequest struct {
	ID string `path:"id" validate:"required,uuid"`

	//openapi:description=User's full name,example=John Doe
	Name string `json:"name,omitempty" validate:"omitempty,min=2,max=50"`

	//openapi:description=User's email address,example=john@example.com
	Email string `json:"email,omitempty" validate:"omitempty,email"`

	//openapi:description=User's role in the system,example=user
	Role string `json:"role,omitempty" validate:"omitempty,oneof=admin user"`
}

// UpdateUserResponse represents the response when updating a user
type UpdateUserResponse struct {
	User    User   `json:"user"`
	Message string `json:"message"`
}

// DeleteUserRequest represents the request to delete a user
type DeleteUserRequest struct {
	ID string `path:"id" validate:"required,uuid"`
}

// DeleteUserResponse represents the response when deleting a user
type DeleteUserResponse struct {
	Message string `json:"message"`
}
