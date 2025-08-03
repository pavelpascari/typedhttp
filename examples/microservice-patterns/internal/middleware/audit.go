package middleware

import (
	"context"
	"log"
)

// AuditLoggingMiddleware logs audit events for admin operations
type AuditLoggingMiddleware[TRequest, TResponse any] struct{}

// Before implements the Middleware interface
func (m *AuditLoggingMiddleware[TRequest, TResponse]) Before(ctx context.Context, req *TRequest) (context.Context, error) {
	log.Printf("Audit: Admin operation attempted - %T", *req)
	return ctx, nil
}

// After implements the Middleware interface
func (m *AuditLoggingMiddleware[TRequest, TResponse]) After(ctx context.Context, req *TRequest, resp *TResponse, err error) (*TResponse, error) {
	if err != nil {
		log.Printf("Audit: Admin operation failed - %T: %v", *req, err)
	} else {
		log.Printf("Audit: Admin operation succeeded - %T", *req)
	}
	return resp, err
}
