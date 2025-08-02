package router

import (
	"github.com/pavelpascari/typedhttp/examples/comprehensive-architecture/internal/handlers"
	"github.com/pavelpascari/typedhttp/examples/comprehensive-architecture/internal/middleware"
	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
)

// Setup configures and returns a router with all routes and middleware
func Setup() *typedhttp.TypedRouter {
	router := typedhttp.NewRouter()

	// Register user routes
	registerUserRoutes(router)

	// Register product routes
	registerProductRoutes(router)

	// Register order routes
	registerOrderRoutes(router)

	// Apply middleware to all handlers
	applyMiddleware(router)

	return router
}

// registerUserRoutes registers all user-related routes
func registerUserRoutes(router *typedhttp.TypedRouter) {
	// Create handler instances that implement the Handler interface directly
	getUserHandler := handlers.NewGetUserHandler()
	createUserHandler := handlers.NewCreateUserHandler()
	listUsersHandler := handlers.NewListUsersHandler()

	typedhttp.GET(router, "/users/{id}", getUserHandler,
		typedhttp.WithTags("users"),
		typedhttp.WithSummary("Get user by ID"),
		typedhttp.WithDescription("Retrieves a user by their unique identifier"),
	)

	typedhttp.POST(router, "/users", createUserHandler,
		typedhttp.WithTags("users"),
		typedhttp.WithSummary("Create a new user"),
		typedhttp.WithDescription("Creates a new user with the provided information"),
	)

	typedhttp.GET(router, "/users", listUsersHandler,
		typedhttp.WithTags("users"),
		typedhttp.WithSummary("List users"),
		typedhttp.WithDescription("Lists users with pagination and filtering options"),
	)
}

// registerProductRoutes registers all product-related routes
func registerProductRoutes(router *typedhttp.TypedRouter) {
	getProductHandler := handlers.NewGetProductHandler()
	createProductHandler := handlers.NewCreateProductHandler()

	typedhttp.GET(router, "/products/{id}", getProductHandler,
		typedhttp.WithTags("products"),
		typedhttp.WithSummary("Get product by ID"),
		typedhttp.WithDescription("Retrieves a product by its unique identifier"),
	)

	typedhttp.POST(router, "/products", createProductHandler,
		typedhttp.WithTags("products"),
		typedhttp.WithSummary("Create a new product"),
		typedhttp.WithDescription("Creates a new product in the catalog"),
	)
}

// registerOrderRoutes registers all order-related routes
func registerOrderRoutes(router *typedhttp.TypedRouter) {
	createOrderHandler := handlers.NewCreateOrderHandler()
	getOrderHandler := handlers.NewGetOrderHandler()

	typedhttp.POST(router, "/orders", createOrderHandler,
		typedhttp.WithTags("orders"),
		typedhttp.WithSummary("Create a new order"),
		typedhttp.WithDescription("Creates a new order for the specified products"),
	)

	typedhttp.GET(router, "/orders/{id}", getOrderHandler,
		typedhttp.WithTags("orders"),
		typedhttp.WithSummary("Get order by ID"),
		typedhttp.WithDescription("Retrieves an order by its unique identifier"),
	)
}

// applyMiddleware applies the middleware stack to all registered handlers
func applyMiddleware(router *typedhttp.TypedRouter) {
	middlewareStack := middleware.SetupMiddlewareStack()
	routeHandlers := router.GetHandlers()

	for i := range routeHandlers {
		routeHandlers[i].MiddlewareEntries = middlewareStack
	}
}
