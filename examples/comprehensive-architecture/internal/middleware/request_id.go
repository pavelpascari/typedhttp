package middleware

import (
	"context"
	"fmt"
	"time"
)

// RequestIDMiddleware demonstrates a simple middleware that doesn't modify responses
type RequestIDMiddleware struct{}

// NewRequestIDMiddleware creates a new request ID middleware
func NewRequestIDMiddleware() *RequestIDMiddleware {
	return &RequestIDMiddleware{}
}

// Before implements the TypedPreMiddleware interface
func (m *RequestIDMiddleware) Before(ctx context.Context, req *interface{}) (context.Context, error) {
	requestID := fmt.Sprintf("req-%d", time.Now().UnixNano())
	return context.WithValue(ctx, "request_id", requestID), nil
}
