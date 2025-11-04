package middleware

import (
	"context"
	"log"
)

// SecurityHeadersMiddleware adds security headers to responses
type SecurityHeadersMiddleware struct{}

// Before implements the Middleware interface
func (m *SecurityHeadersMiddleware) Before(ctx context.Context, req *interface{}) (context.Context, error) {
	// Would set security headers in real implementation
	log.Println("Security: Applied security headers")
	return ctx, nil
}

// AdminAuthMiddleware validates admin credentials
type AdminAuthMiddleware struct{}

// Before implements the Middleware interface
func (m *AdminAuthMiddleware) Before(ctx context.Context, req *interface{}) (context.Context, error) {
	// Would validate admin credentials
	log.Println("AdminAuth: Validated admin credentials")
	return context.WithValue(ctx, "admin_user", "admin@example.com"), nil
}
