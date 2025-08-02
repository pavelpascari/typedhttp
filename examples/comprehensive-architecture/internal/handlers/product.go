package handlers

import (
	"context"
	"time"

	"github.com/pavelpascari/typedhttp/examples/comprehensive-architecture/internal/models"
)

// ProductHandler handles product-related operations
type ProductHandler struct {
	// In a real application, this would have dependencies like:
	// repo ProductRepository
	// logger *slog.Logger
	// validator *validator.Validate
}

// GetProductHandler implements the TypedHTTP Handler interface for GetProduct
type GetProductHandler struct {
	handler *ProductHandler
}

// CreateProductHandler implements the TypedHTTP Handler interface for CreateProduct
type CreateProductHandler struct {
	handler *ProductHandler
}

// NewProductHandler creates a new product handler
func NewProductHandler() *ProductHandler {
	return &ProductHandler{}
}

// NewGetProductHandler creates a new GetProduct handler
func NewGetProductHandler() *GetProductHandler {
	return &GetProductHandler{handler: NewProductHandler()}
}

// NewCreateProductHandler creates a new CreateProduct handler
func NewCreateProductHandler() *CreateProductHandler {
	return &CreateProductHandler{handler: NewProductHandler()}
}

// Handle implements the TypedHTTP Handler interface for GetProduct
func (h *GetProductHandler) Handle(ctx context.Context, req models.GetProductRequest) (models.GetProductResponse, error) {
	return h.handler.GetProduct(ctx, req)
}

// GetProduct implements the business logic for getting a product by ID
func (h *ProductHandler) GetProduct(ctx context.Context, req models.GetProductRequest) (models.GetProductResponse, error) {
	product := models.Product{
		ID:          req.ID,
		Name:        "iPhone 13 Pro",
		Description: "Latest iPhone with advanced camera system",
		Price:       999.99,
		CategoryID:  "550e8400-e29b-41d4-a716-446655440001",
		InStock:     true,
		CreatedAt:   time.Now().Add(-72 * time.Hour),
	}

	return models.GetProductResponse{Product: product}, nil
}

// Handle implements the TypedHTTP Handler interface for CreateProduct
func (h *CreateProductHandler) Handle(ctx context.Context, req models.CreateProductRequest) (models.CreateProductResponse, error) {
	return h.handler.CreateProduct(ctx, req)
}

// CreateProduct implements the business logic for product creation
func (h *ProductHandler) CreateProduct(ctx context.Context, req models.CreateProductRequest) (models.CreateProductResponse, error) {
	product := models.Product{
		ID:          "550e8400-e29b-41d4-a716-446655440002",
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		CategoryID:  req.CategoryID,
		InStock:     req.InStock,
		CreatedAt:   time.Now(),
	}

	return models.CreateProductResponse{
		Product: product,
		Message: "Product created successfully",
	}, nil
}
