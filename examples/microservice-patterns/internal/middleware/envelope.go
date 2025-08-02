package middleware

import (
	"context"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

// SimpleResponseMiddleware provides simple response wrapping for internal services
type SimpleResponseMiddleware[TResponse any] struct{}

// SimpleResponse wraps responses with minimal metadata
type SimpleResponse[T any] struct {
	Data      T      `json:"data"`
	RequestID string `json:"request_id"`
}

// After implements the ResponseSchemaModifier interface
func (m *SimpleResponseMiddleware[TResponse]) After(ctx context.Context, resp *TResponse) (*SimpleResponse[TResponse], error) {
	requestID, _ := ctx.Value("request_id").(string)
	return &SimpleResponse[TResponse]{
		Data:      *resp,
		RequestID: requestID,
	}, nil
}

// ModifyResponseSchema implements the ResponseSchemaModifier interface
func (m *SimpleResponseMiddleware[TResponse]) ModifyResponseSchema(ctx context.Context, originalSchema *openapi3.SchemaRef) (*openapi3.SchemaRef, error) {
	return &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			Type:        &openapi3.Types{"object"},
			Description: "Simple internal service response",
			Properties: map[string]*openapi3.SchemaRef{
				"data": {
					Value: &openapi3.Schema{
						Description: "Response data",
						OneOf:       []*openapi3.SchemaRef{originalSchema},
					},
				},
				"request_id": {
					Value: &openapi3.Schema{
						Type:        &openapi3.Types{"string"},
						Description: "Internal request identifier",
					},
				},
			},
			Required: []string{"data", "request_id"},
		},
	}, nil
}

// AdminResponseEnvelopeMiddleware provides admin response wrapping with audit info
type AdminResponseEnvelopeMiddleware[TResponse any] struct{}

// AdminResponse wraps responses with admin metadata
type AdminResponse[T any] struct {
	Data        T      `json:"data"`
	RequestID   string `json:"request_id"`
	AdminUser   string `json:"admin_user"`
	Timestamp   string `json:"timestamp"`
	Environment string `json:"environment"`
}

// After implements the ResponseSchemaModifier interface
func (m *AdminResponseEnvelopeMiddleware[TResponse]) After(ctx context.Context, resp *TResponse) (*AdminResponse[TResponse], error) {
	requestID, _ := ctx.Value("request_id").(string)
	adminUser, _ := ctx.Value("admin_user").(string)

	return &AdminResponse[TResponse]{
		Data:        *resp,
		RequestID:   requestID,
		AdminUser:   adminUser,
		Timestamp:   time.Now().Format(time.RFC3339),
		Environment: "production",
	}, nil
}

// ModifyResponseSchema implements the ResponseSchemaModifier interface
func (m *AdminResponseEnvelopeMiddleware[TResponse]) ModifyResponseSchema(ctx context.Context, originalSchema *openapi3.SchemaRef) (*openapi3.SchemaRef, error) {
	return &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			Type:        &openapi3.Types{"object"},
			Description: "Admin API response with audit information",
			Properties: map[string]*openapi3.SchemaRef{
				"data": {
					Value: &openapi3.Schema{
						Description: "Response data",
						OneOf:       []*openapi3.SchemaRef{originalSchema},
					},
				},
				"request_id": {
					Value: &openapi3.Schema{
						Type:        &openapi3.Types{"string"},
						Description: "Request identifier for tracing",
					},
				},
				"admin_user": {
					Value: &openapi3.Schema{
						Type:        &openapi3.Types{"string"},
						Description: "Admin user who made the request",
					},
				},
				"timestamp": {
					Value: &openapi3.Schema{
						Type:        &openapi3.Types{"string"},
						Format:      "date-time",
						Description: "Request timestamp",
					},
				},
				"environment": {
					Value: &openapi3.Schema{
						Type:        &openapi3.Types{"string"},
						Description: "Environment where request was processed",
					},
				},
			},
			Required: []string{"data", "request_id", "admin_user", "timestamp", "environment"},
		},
	}, nil
}