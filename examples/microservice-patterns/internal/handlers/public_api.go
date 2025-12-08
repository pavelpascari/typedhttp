package handlers

import (
	"context"
	"time"

	"github.com/pavelpascari/typedhttp/examples/microservice-patterns/internal/models"
)

// PublicAPIHandler handles public API operations
type PublicAPIHandler struct {
	// In a real application, this would have dependencies like:
	// userRepo UserRepository
	// logger   *slog.Logger
}

// GetUserProfileHandler implements the TypedHTTP Handler interface for GetUserProfile
type GetUserProfileHandler struct {
	handler *PublicAPIHandler
}

// NewPublicAPIHandler creates a new public API handler
func NewPublicAPIHandler() *PublicAPIHandler {
	return &PublicAPIHandler{}
}

// NewGetUserProfileHandler creates a new GetUserProfile handler
func NewGetUserProfileHandler() *GetUserProfileHandler {
	return &GetUserProfileHandler{handler: NewPublicAPIHandler()}
}

// Handle implements the TypedHTTP Handler interface for GetUserProfile
func (h *GetUserProfileHandler) Handle(ctx context.Context, req models.GetUserProfileRequest) (models.GetUserProfileResponse, error) {
	return h.handler.GetUserProfile(ctx, req)
}

// GetUserProfile implements the business logic for getting a user profile
func (h *PublicAPIHandler) GetUserProfile(ctx context.Context, req models.GetUserProfileRequest) (models.GetUserProfileResponse, error) {
	// Simulate external API call to user service
	profile := models.UserProfile{
		ID:       req.UserID,
		Name:     "John Doe",
		Email:    "john@example.com",
		Avatar:   "https://example.com/avatars/john.jpg",
		LastSeen: time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
	}

	return models.GetUserProfileResponse{Profile: profile}, nil
}
