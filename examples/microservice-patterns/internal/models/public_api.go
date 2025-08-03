package models

// Public API related types

// GetUserProfileRequest represents a request to get a user profile
type GetUserProfileRequest struct {
	UserID string `path:"user_id" validate:"required,uuid"`
}

// UserProfile represents a user profile
type UserProfile struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar_url,omitempty"`
	LastSeen string `json:"last_seen"`
}

// GetUserProfileResponse represents the response for getting a user profile
type GetUserProfileResponse struct {
	Profile UserProfile `json:"profile"`
}
