package handlers

import (
	"context"
	"time"

	"github.com/pavelpascari/typedhttp/examples/microservice-patterns/internal/models"
)

// HealthHandler handles health check operations
type HealthHandler struct {
	// In a real application, this would have dependencies like:
	// healthChecker HealthChecker
	// logger        *slog.Logger
}

// HealthCheckHandler implements the TypedHTTP Handler interface for GetHealth
type HealthCheckHandler struct {
	handler *HealthHandler
}

// NewHealthHandler creates a new health handler
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// NewHealthCheckHandler creates a new HealthCheck handler
func NewHealthCheckHandler() *HealthCheckHandler {
	return &HealthCheckHandler{handler: NewHealthHandler()}
}

// Handle implements the TypedHTTP Handler interface for GetHealth
func (h *HealthCheckHandler) Handle(ctx context.Context, req models.HealthCheckRequest) (models.HealthCheckResponse, error) {
	return h.handler.GetHealth(ctx, req)
}

// GetHealth implements the business logic for health checks
func (h *HealthHandler) GetHealth(ctx context.Context, req models.HealthCheckRequest) (models.HealthCheckResponse, error) {
	checks := map[string]models.HealthCheck{
		"database": {
			Status:    "healthy",
			Duration:  2 * time.Millisecond,
			Message:   "Connection pool: 8/10 active",
			LastCheck: time.Now(),
		},
		"redis": {
			Status:    "healthy",
			Duration:  1 * time.Millisecond,
			Message:   "Ping successful",
			LastCheck: time.Now(),
		},
		"external_api": {
			Status:    "degraded",
			Duration:  500 * time.Millisecond,
			Message:   "High latency detected",
			LastCheck: time.Now(),
		},
	}

	return models.HealthCheckResponse{
		Service: models.ServiceInfo{
			Name:    "ecommerce-api",
			Version: "1.2.3",
			Health:  "healthy",
		},
		Status:      "healthy",
		Timestamp:   time.Now(),
		Checks:      checks,
		Version:     "1.2.3",
		Environment: "production",
	}, nil
}