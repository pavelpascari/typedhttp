package middleware

import (
	"github.com/pavelpascari/typedhttp/examples/microservice-patterns/internal/models"
	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
)

// CreatePublicAPIMiddleware creates the middleware stack for public API endpoints
func CreatePublicAPIMiddleware() []typedhttp.MiddlewareEntry {
	return []typedhttp.MiddlewareEntry{
		// Security first
		{
			Middleware: &SecurityHeadersMiddleware{},
			Config: typedhttp.MiddlewareConfig{
				Name:     "security_headers",
				Priority: 100,
				Scope:    typedhttp.ScopeGlobal,
			},
		},
		// Rate limiting for public endpoints
		{
			Middleware: &RateLimitMiddleware{
				RequestsPerMinute: 100,
				BurstSize:         10,
			},
			Config: typedhttp.MiddlewareConfig{
				Name:     "rate_limit",
				Priority: 90,
				Scope:    typedhttp.ScopeGlobal,
			},
		},
		// Request tracking
		{
			Middleware: &RequestTrackingMiddleware{},
			Config: typedhttp.MiddlewareConfig{
				Name:     "request_tracking",
				Priority: 80,
				Scope:    typedhttp.ScopeGlobal,
			},
		},
		// Response envelope for consistency
		{
			Middleware: typedhttp.NewResponseEnvelopeMiddleware[any](
				typedhttp.WithRequestID(true),
				typedhttp.WithTimestamp(true),
				typedhttp.WithMeta(true),
			),
			Config: typedhttp.MiddlewareConfig{
				Name:     "response_envelope",
				Priority: 70,
				Scope:    typedhttp.ScopeGlobal,
			},
		},
	}
}

// CreateInternalServiceMiddleware creates the middleware stack for internal services
func CreateInternalServiceMiddleware() []typedhttp.MiddlewareEntry {
	return []typedhttp.MiddlewareEntry{
		// Basic request tracking
		{
			Middleware: &RequestTrackingMiddleware{},
			Config: typedhttp.MiddlewareConfig{
				Name:     "request_tracking",
				Priority: 100,
				Scope:    typedhttp.ScopeGlobal,
			},
		},
		// Simple response format
		{
			Middleware: &SimpleResponseMiddleware[any]{},
			Config: typedhttp.MiddlewareConfig{
				Name:     "simple_response",
				Priority: 50,
				Scope:    typedhttp.ScopeGlobal,
			},
		},
	}
}

// CreateAdminAPIMiddleware creates the middleware stack for admin API endpoints
func CreateAdminAPIMiddleware() []typedhttp.MiddlewareEntry {
	return []typedhttp.MiddlewareEntry{
		// Enhanced security
		{
			Middleware: &SecurityHeadersMiddleware{},
			Config: typedhttp.MiddlewareConfig{
				Name:     "security_headers",
				Priority: 100,
				Scope:    typedhttp.ScopeGlobal,
			},
		},
		// Admin authentication
		{
			Middleware: &AdminAuthMiddleware{},
			Config: typedhttp.MiddlewareConfig{
				Name:     "admin_auth",
				Priority: 90,
				Scope:    typedhttp.ScopeGlobal,
			},
		},
		// Audit logging
		{
			Middleware: &AuditLoggingMiddleware[any, any]{},
			Config: typedhttp.MiddlewareConfig{
				Name:     "audit_logging",
				Priority: 80,
				Scope:    typedhttp.ScopeGlobal,
			},
		},
		// Request tracking
		{
			Middleware: &RequestTrackingMiddleware{},
			Config: typedhttp.MiddlewareConfig{
				Name:     "request_tracking",
				Priority: 70,
				Scope:    typedhttp.ScopeGlobal,
			},
		},
		// Envelope with admin metadata
		{
			Middleware: &AdminResponseEnvelopeMiddleware[any]{},
			Config: typedhttp.MiddlewareConfig{
				Name:     "admin_envelope",
				Priority: 60,
				Scope:    typedhttp.ScopeGlobal,
			},
		},
	}
}

// CreateHealthCheckMiddleware creates the middleware stack for health check endpoints
func CreateHealthCheckMiddleware() []typedhttp.MiddlewareEntry {
	return []typedhttp.MiddlewareEntry{
		// Basic tracking only
		{
			Middleware: &RequestTrackingMiddleware{},
			Config: typedhttp.MiddlewareConfig{
				Name:     "request_tracking",
				Priority: 100,
				Scope:    typedhttp.ScopeGlobal,
			},
		},
	}
}

// GetMiddlewareForServiceType returns the appropriate middleware stack for a service type
func GetMiddlewareForServiceType(serviceType models.ServiceType) []typedhttp.MiddlewareEntry {
	switch serviceType {
	case models.PublicAPI:
		return CreatePublicAPIMiddleware()
	case models.InternalService:
		return CreateInternalServiceMiddleware()
	case models.AdminAPI:
		return CreateAdminAPIMiddleware()
	case models.HealthCheckService:
		return CreateHealthCheckMiddleware()
	default:
		return CreateInternalServiceMiddleware()
	}
}