package handlers

import (
	"context"
	"time"

	"github.com/pavelpascari/typedhttp/examples/comprehensive-architecture/internal/models"
)

// OrderHandler handles order-related operations
type OrderHandler struct {
	// In a real application, this would have dependencies like:
	// repo OrderRepository
	// productRepo ProductRepository
	// logger *slog.Logger
	// validator *validator.Validate
}

// CreateOrderHandler implements the TypedHTTP Handler interface for CreateOrder
type CreateOrderHandler struct {
	handler *OrderHandler
}

// GetOrderHandler implements the TypedHTTP Handler interface for GetOrder
type GetOrderHandler struct {
	handler *OrderHandler
}

// NewOrderHandler creates a new order handler
func NewOrderHandler() *OrderHandler {
	return &OrderHandler{}
}

// NewCreateOrderHandler creates a new CreateOrder handler
func NewCreateOrderHandler() *CreateOrderHandler {
	return &CreateOrderHandler{handler: NewOrderHandler()}
}

// NewGetOrderHandler creates a new GetOrder handler
func NewGetOrderHandler() *GetOrderHandler {
	return &GetOrderHandler{handler: NewOrderHandler()}
}

// Handle implements the TypedHTTP Handler interface for CreateOrder
func (h *CreateOrderHandler) Handle(ctx context.Context, req models.CreateOrderRequest) (models.CreateOrderResponse, error) {
	return h.handler.CreateOrder(ctx, req)
}

// CreateOrder implements the business logic for order creation
func (h *OrderHandler) CreateOrder(
	ctx context.Context,
	req models.CreateOrderRequest,
) (models.CreateOrderResponse, error) {
	// Simulate order creation with total calculation
	total := 999.99 * float64(len(req.ProductIDs))

	order := models.Order{
		ID:         "550e8400-e29b-41d4-a716-446655440003",
		UserID:     req.UserID,
		ProductIDs: req.ProductIDs,
		Total:      total,
		Status:     "pending",
		CreatedAt:  time.Now(),
	}

	return models.CreateOrderResponse{
		Order:   order,
		Message: "Order created successfully",
	}, nil
}

// Handle implements the TypedHTTP Handler interface for GetOrder
func (h *GetOrderHandler) Handle(ctx context.Context, req models.GetOrderRequest) (models.GetOrderResponse, error) {
	return h.handler.GetOrder(ctx, req)
}

// GetOrder implements the business logic for getting an order by ID
func (h *OrderHandler) GetOrder(ctx context.Context, req models.GetOrderRequest) (models.GetOrderResponse, error) {
	order := models.Order{
		ID:         req.ID,
		UserID:     "550e8400-e29b-41d4-a716-446655440000",
		ProductIDs: []string{"550e8400-e29b-41d4-a716-446655440002"},
		Total:      999.99,
		Status:     "confirmed",
		CreatedAt:  time.Now().Add(-1 * time.Hour),
	}

	return models.GetOrderResponse{Order: order}, nil
}
