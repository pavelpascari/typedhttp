package middleware

import (
	"context"
	"fmt"
	"log"
	"time"
)

// RequestTrackingMiddleware tracks requests with unique IDs
type RequestTrackingMiddleware struct{}

// Before implements the Middleware interface
func (m *RequestTrackingMiddleware) Before(ctx context.Context, req *interface{}) (context.Context, error) {
	requestID := fmt.Sprintf("req-%d", time.Now().UnixNano())
	log.Printf("Tracking: Request %s started", requestID)
	return context.WithValue(ctx, "request_id", requestID), nil
}