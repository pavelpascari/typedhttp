package router

import (
	"github.com/pavelpascari/typedhttp/examples/microservice-patterns/internal/handlers"
	"github.com/pavelpascari/typedhttp/examples/microservice-patterns/internal/middleware"
	"github.com/pavelpascari/typedhttp/examples/microservice-patterns/internal/models"
	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
)

// Setup creates and configures a router for a specific service type
func Setup(serviceType models.ServiceType) *typedhttp.TypedRouter {
	router := typedhttp.NewRouter()

	// Register routes based on service type
	switch serviceType {
	case models.PublicAPI:
		registerPublicAPIRoutes(router)
	case models.InternalService:
		registerInternalServiceRoutes(router)
	case models.AdminAPI:
		registerAdminAPIRoutes(router)
	case models.HealthCheckService:
		registerHealthCheckRoutes(router)
	}

	// Apply middleware appropriate for the service type
	applyMiddleware(router, serviceType)

	return router
}

// registerPublicAPIRoutes registers public API routes
func registerPublicAPIRoutes(router *typedhttp.TypedRouter) {
	getUserProfileHandler := handlers.NewGetUserProfileHandler()

	typedhttp.GET(router, "/users/{user_id}/profile", getUserProfileHandler,
		typedhttp.WithTags("users"),
		typedhttp.WithSummary("Get user profile"),
		typedhttp.WithDescription("Retrieves a user profile from the public API"),
	)
}

// registerInternalServiceRoutes registers internal service routes
func registerInternalServiceRoutes(router *typedhttp.TypedRouter) {
	processDataHandler := handlers.NewProcessDataHandler()

	typedhttp.POST(router, "/internal/data/{data_id}/process", processDataHandler,
		typedhttp.WithTags("internal"),
		typedhttp.WithSummary("Process data"),
		typedhttp.WithDescription("Processes data in the internal service"),
	)
}

// registerAdminAPIRoutes registers admin API routes
func registerAdminAPIRoutes(router *typedhttp.TypedRouter) {
	getSystemStatsHandler := handlers.NewGetSystemStatsHandler()

	typedhttp.GET(router, "/admin/stats", getSystemStatsHandler,
		typedhttp.WithTags("admin"),
		typedhttp.WithSummary("Get system stats"),
		typedhttp.WithDescription("Retrieves system statistics for administrators"),
	)
}

// registerHealthCheckRoutes registers health check routes
func registerHealthCheckRoutes(router *typedhttp.TypedRouter) {
	healthCheckHandler := handlers.NewHealthCheckHandler()

	typedhttp.GET(router, "/health", healthCheckHandler,
		typedhttp.WithTags("health"),
		typedhttp.WithSummary("Health check"),
		typedhttp.WithDescription("Returns service health status"),
	)
}

// applyMiddleware applies the appropriate middleware stack for the service type
func applyMiddleware(router *typedhttp.TypedRouter, serviceType models.ServiceType) {
	middlewareStack := middleware.GetMiddlewareForServiceType(serviceType)
	routeHandlers := router.GetHandlers()

	for i := range routeHandlers {
		routeHandlers[i].MiddlewareEntries = middlewareStack
	}
}