package models

import "time"

// ServiceInfo represents basic service information
type ServiceInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Health  string `json:"health"`
}

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	TraceID string `json:"trace_id"`
}

// ServiceType represents different types of microservices
type ServiceType int

const (
	PublicAPI ServiceType = iota
	InternalService
	AdminAPI
	HealthCheckService
)

// HealthCheck represents individual health check results
type HealthCheck struct {
	Status    string        `json:"status"`
	Duration  time.Duration `json:"response_time_ms"`
	Message   string        `json:"message,omitempty"`
	LastCheck time.Time     `json:"last_check"`
}