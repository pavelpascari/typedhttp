package models

import "time"

// Order represents an order in the e-commerce system
type Order struct {
	ID         string    `json:"id" validate:"required,uuid"`
	UserID     string    `json:"user_id" validate:"required,uuid"`
	ProductIDs []string  `json:"product_ids" validate:"required,min=1"`
	Total      float64   `json:"total" validate:"required,min=0"`
	Status     string    `json:"status" validate:"required,oneof=pending confirmed shipped delivered cancelled"`
	CreatedAt  time.Time `json:"created_at"`
}

// CreateOrderRequest represents the request to create a new order
type CreateOrderRequest struct {
	//openapi:description=User ID placing the order,example=550e8400-e29b-41d4-a716-446655440000
	UserID string `json:"user_id" validate:"required,uuid"`

	//openapi:description=List of product IDs to order,example=["550e8400-e29b-41d4-a716-446655440001"]
	ProductIDs []string `json:"product_ids" validate:"required,min=1"`
}

// CreateOrderResponse represents the response when creating an order
type CreateOrderResponse struct {
	Order   Order  `json:"order"`
	Message string `json:"message"`
}

// GetOrderRequest represents the request to get an order by ID
type GetOrderRequest struct {
	ID string `path:"id" validate:"required,uuid"`
}

// GetOrderResponse represents the response when getting an order
type GetOrderResponse struct {
	Order Order `json:"order"`
}