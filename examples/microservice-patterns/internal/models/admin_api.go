package models

import "time"

// Admin API related types

// SystemStatsRequest represents a request for system statistics
type SystemStatsRequest struct {
	Service   string `query:"service" validate:"omitempty,oneof=api auth data"`
	StartTime string `query:"start_time" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	EndTime   string `query:"end_time" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
}

// SystemStats represents system statistics
type SystemStats struct {
	Service     string           `json:"service"`
	Uptime      time.Duration    `json:"uptime_seconds"`
	Requests    int64            `json:"total_requests"`
	Errors      int64            `json:"total_errors"`
	AvgLatency  time.Duration    `json:"avg_latency_ms"`
	Memory      int64            `json:"memory_mb"`
	CPU         float64          `json:"cpu_percent"`
	Endpoints   map[string]int64 `json:"endpoint_hits"`
	LastRestart time.Time        `json:"last_restart"`
}

// GetSystemStatsResponse represents the response for system statistics
type GetSystemStatsResponse struct {
	Stats SystemStats `json:"stats"`
}