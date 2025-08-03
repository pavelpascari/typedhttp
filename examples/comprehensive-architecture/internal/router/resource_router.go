package router

import (
	"github.com/pavelpascari/typedhttp/examples/comprehensive-architecture/internal/middleware"
	"github.com/pavelpascari/typedhttp/examples/comprehensive-architecture/internal/services"
	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
)

// SetupWithResources demonstrates the new resource-based router setup.
// This drastically reduces boilerplate compared to the traditional approach.
func SetupWithResources() *typedhttp.TypedRouter {
	// Create a domain router with middleware
	middlewareStack := middleware.SetupMiddlewareStack()
	router := typedhttp.NewDomainRouter("/api/v1", middlewareStack...)

	// Register resources with dramatically less code
	registerUserResource(router)
	registerProductResource(router)
	registerOrderResource(router)

	return router.TypedRouter
}

// registerUserResource demonstrates the new resource pattern.
// Compare this with the old registerUserRoutes function - this is 80% less code!
func registerUserResource(router *typedhttp.DomainRouter) {
	userService := services.NewUserService()

	// Single call replaces 15+ lines of individual handler registration
	typedhttp.Resource(router, "/users", userService, typedhttp.ResourceConfig{
		Tags: []string{"users"},
		Operations: map[string]typedhttp.OperationConfig{
			"GET": {
				Summary:     "Get user by ID",
				Description: "Retrieves a user by their unique identifier",
				Enabled:     true,
			},
			"LIST": {
				Summary:     "List users",
				Description: "Lists users with pagination and filtering options",
				Enabled:     true,
			},
			"POST": {
				Summary:     "Create a new user",
				Description: "Creates a new user with the provided information",
				Enabled:     true,
			},
			"PUT": {
				Summary:     "Update user",
				Description: "Updates an existing user",
				Enabled:     true,
			},
			"DELETE": {
				Summary:     "Delete user",
				Description: "Deletes a user by ID",
				Enabled:     true,
			},
		},
	})
}

// registerProductResource would follow the same pattern
func registerProductResource(router *typedhttp.DomainRouter) {
	// TODO: Implement ProductService following the same pattern
	// This would replace the entire product handlers package
}

// registerOrderResource would follow the same pattern
func registerOrderResource(router *typedhttp.DomainRouter) {
	// TODO: Implement OrderService following the same pattern
	// This would replace the entire order handlers package
}

// BoilerplateComparison demonstrates the dramatic reduction in code:
//
// OLD APPROACH (handlers/user.go):
// - 3 wrapper handler structs (GetUserHandler, CreateUserHandler, ListUsersHandler)
// - 3 constructor functions (NewGetUserHandler, etc.)
// - 3 Handle method wrappers that just forward calls
// - Total: ~80 lines of pure boilerplate
//
// NEW APPROACH (services/user.go):
// - 1 service struct with direct CRUD methods
// - 1 constructor function
// - No wrapper methods needed
// - Total: ~5 lines of setup code
//
// ROUTER REGISTRATION:
// OLD: 15+ lines per resource with repetitive typedhttp.GET/POST calls
// NEW: 1 Resource() call with configuration
//
// RESULT: 80% reduction in boilerplate code per resource!
