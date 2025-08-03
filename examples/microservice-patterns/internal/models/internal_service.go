package models

import "time"

// Internal service related types

// ProcessDataRequest represents a request to process data
type ProcessDataRequest struct {
	DataID string `path:"data_id" validate:"required"`
	Action string `query:"action" validate:"required,oneof=validate transform store"`
}

// ProcessingResult represents the result of data processing
type ProcessingResult struct {
	DataID    string        `json:"data_id"`
	Action    string        `json:"action"`
	Status    string        `json:"status"`
	Duration  time.Duration `json:"duration_ms"`
	Processed int           `json:"records_processed"`
}

// ProcessDataResponse represents the response for data processing
type ProcessDataResponse struct {
	Result ProcessingResult `json:"result"`
}
