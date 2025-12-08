package handlers

import (
	"context"
	"time"

	"github.com/pavelpascari/typedhttp/examples/microservice-patterns/internal/models"
)

// AdminAPIHandler handles admin API operations
type AdminAPIHandler struct {
	// In a real application, this would have dependencies like:
	// metricsCollector MetricsCollector
	// logger           *slog.Logger
}

// GetSystemStatsHandler implements the TypedHTTP Handler interface for GetSystemStats
type GetSystemStatsHandler struct {
	handler *AdminAPIHandler
}

// NewAdminAPIHandler creates a new admin API handler
func NewAdminAPIHandler() *AdminAPIHandler {
	return &AdminAPIHandler{}
}

// NewGetSystemStatsHandler creates a new GetSystemStats handler
func NewGetSystemStatsHandler() *GetSystemStatsHandler {
	return &GetSystemStatsHandler{handler: NewAdminAPIHandler()}
}

// Handle implements the TypedHTTP Handler interface for GetSystemStats
func (h *GetSystemStatsHandler) Handle(ctx context.Context, req models.SystemStatsRequest) (models.GetSystemStatsResponse, error) {
	return h.handler.GetSystemStats(ctx, req)
}

// GetSystemStats implements the business logic for getting system statistics
func (h *AdminAPIHandler) GetSystemStats(ctx context.Context, req models.SystemStatsRequest) (models.GetSystemStatsResponse, error) {
	stats := models.SystemStats{
		Service:    "api-gateway",
		Uptime:     24 * time.Hour,
		Requests:   1500000,
		Errors:     1200,
		AvgLatency: 45 * time.Millisecond,
		Memory:     512,
		CPU:        23.5,
		Endpoints: map[string]int64{
			"/users/{id}": 850000,
			"/products":   425000,
			"/orders":     225000,
		},
		LastRestart: time.Now().Add(-24 * time.Hour),
	}

	return models.GetSystemStatsResponse{Stats: stats}, nil
}
