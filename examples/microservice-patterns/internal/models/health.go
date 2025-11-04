package models

import "time"

// Health check related types

// HealthCheckRequest represents a health check request
type HealthCheckRequest struct{}

// HealthCheckResponse represents a health check response
type HealthCheckResponse struct {
	Service     ServiceInfo            `json:"service"`
	Status      string                 `json:"status"`
	Timestamp   time.Time              `json:"timestamp"`
	Checks      map[string]HealthCheck `json:"checks"`
	Version     string                 `json:"version"`
	Environment string                 `json:"environment"`
}
