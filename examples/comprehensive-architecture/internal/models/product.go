package models

import "time"

// Product represents a product in the e-commerce system
type Product struct {
	ID          string    `json:"id" validate:"required,uuid"`
	Name        string    `json:"name" validate:"required,min=2,max=100"`
	Description string    `json:"description,omitempty"`
	Price       float64   `json:"price" validate:"required,min=0"`
	CategoryID  string    `json:"category_id" validate:"required,uuid"`
	InStock     bool      `json:"in_stock"`
	CreatedAt   time.Time `json:"created_at"`
}

// GetProductRequest represents the request to get a product by ID
type GetProductRequest struct {
	ID string `path:"id" validate:"required,uuid"`
}

// GetProductResponse represents the response when getting a product
type GetProductResponse struct {
	Product Product `json:"product"`
}

// CreateProductRequest represents the request to create a new product
type CreateProductRequest struct {
	//openapi:description=Product name,example=iPhone 13 Pro
	Name string `json:"name" validate:"required,min=2,max=100"`

	//openapi:description=Product description,example=Latest iPhone with advanced camera system
	Description string `json:"description,omitempty"`

	//openapi:description=Product price in USD,example=999.99
	Price float64 `json:"price" validate:"required,min=0"`

	//openapi:description=Product category ID,example=550e8400-e29b-41d4-a716-446655440001
	CategoryID string `json:"category_id" validate:"required,uuid"`

	//openapi:description=Whether product is in stock,example=true
	InStock bool `json:"in_stock"`
}

// CreateProductResponse represents the response when creating a product
type CreateProductResponse struct {
	Product Product `json:"product"`
	Message string  `json:"message"`
}
