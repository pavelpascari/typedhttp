package middleware

import (
	"context"
	"log"
)

// AuditMiddleware demonstrates middleware that logs operations without modifying responses
type AuditMiddleware[TRequest, TResponse any] struct{}

// NewAuditMiddleware creates a new audit middleware
func NewAuditMiddleware[TRequest, TResponse any]() *AuditMiddleware[TRequest, TResponse] {
	return &AuditMiddleware[TRequest, TResponse]{}
}

// Before implements the TypedPreMiddleware interface
func (m *AuditMiddleware[TRequest, TResponse]) Before(ctx context.Context, req *TRequest) (context.Context, error) {
	log.Printf("Audit: Request received for %T", *req)
	return ctx, nil
}

// After implements the TypedPostMiddleware interface
func (m *AuditMiddleware[TRequest, TResponse]) After(ctx context.Context, req *TRequest, resp *TResponse, err error) (*TResponse, error) {
	if err != nil {
		log.Printf("Audit: Request failed for %T: %v", *req, err)
	} else {
		log.Printf("Audit: Request succeeded for %T", *req)
	}
	return resp, err
}
