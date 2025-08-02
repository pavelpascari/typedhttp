package handlers

import (
	"context"
	"time"

	"github.com/pavelpascari/typedhttp/examples/microservice-patterns/internal/models"
)

// InternalServiceHandler handles internal service operations
type InternalServiceHandler struct {
	// In a real application, this would have dependencies like:
	// dataProcessor DataProcessor
	// logger        *slog.Logger
}

// ProcessDataHandler implements the TypedHTTP Handler interface for ProcessData
type ProcessDataHandler struct {
	handler *InternalServiceHandler
}

// NewInternalServiceHandler creates a new internal service handler
func NewInternalServiceHandler() *InternalServiceHandler {
	return &InternalServiceHandler{}
}

// NewProcessDataHandler creates a new ProcessData handler
func NewProcessDataHandler() *ProcessDataHandler {
	return &ProcessDataHandler{handler: NewInternalServiceHandler()}
}

// Handle implements the TypedHTTP Handler interface for ProcessData
func (h *ProcessDataHandler) Handle(ctx context.Context, req models.ProcessDataRequest) (models.ProcessDataResponse, error) {
	return h.handler.ProcessData(ctx, req)
}

// ProcessData implements the business logic for data processing
func (h *InternalServiceHandler) ProcessData(ctx context.Context, req models.ProcessDataRequest) (models.ProcessDataResponse, error) {
	start := time.Now()

	// Simulate data processing
	time.Sleep(10 * time.Millisecond)

	result := models.ProcessingResult{
		DataID:    req.DataID,
		Action:    req.Action,
		Status:    "completed",
		Duration:  time.Since(start),
		Processed: 1000,
	}

	return models.ProcessDataResponse{Result: result}, nil
}
