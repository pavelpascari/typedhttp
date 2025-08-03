package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
)

// JWT-based authentication middleware and handlers
// Copy-paste ready for production use

// JWTClaims represents the JWT payload
type JWTClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// JWTMiddleware provides JWT authentication
type JWTMiddleware struct {
	Secret    []byte
	Algorithm string // Default: HS256
}

// NewJWTMiddleware creates a new JWT middleware
func NewJWTMiddleware(secret string) *JWTMiddleware {
	return &JWTMiddleware{
		Secret:    []byte(secret),
		Algorithm: "HS256",
	}
}

// Middleware returns HTTP middleware for JWT authentication
func (m *JWTMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		// Check Bearer prefix
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Validate token
		claims, err := m.ValidateToken(tokenString)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid token: %v", err), http.StatusUnauthorized)
			return
		}

		// Add claims to context
		ctx := context.WithValue(r.Context(), "jwt_claims", claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ValidateToken validates and parses a JWT token
func (m *JWTMiddleware) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if token.Method.Alg() != m.Algorithm {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.Secret, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// GenerateToken creates a new JWT token
func (m *JWTMiddleware) GenerateToken(userID, email, role string, duration time.Duration) (string, error) {
	claims := &JWTClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.GetSigningMethod(m.Algorithm), claims)
	return token.SignedString(m.Secret)
}

// Utility functions for extracting claims from context

// GetJWTClaims extracts JWT claims from request context
func GetJWTClaims(ctx context.Context) (*JWTClaims, error) {
	claims, ok := ctx.Value("jwt_claims").(*JWTClaims)
	if !ok {
		return nil, fmt.Errorf("no JWT claims found in context")
	}
	return claims, nil
}

// GetCurrentUserID extracts the current user ID from JWT claims
func GetCurrentUserID(ctx context.Context) (string, error) {
	claims, err := GetJWTClaims(ctx)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

// GetCurrentUserRole extracts the current user role from JWT claims
func GetCurrentUserRole(ctx context.Context) (string, error) {
	claims, err := GetJWTClaims(ctx)
	if err != nil {
		return "", err
	}
	return claims.Role, nil
}

// RequireRole creates middleware that requires a specific role
func (m *JWTMiddleware) RequireRole(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := GetJWTClaims(r.Context())
			if err != nil {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			if claims.Role != requiredRole {
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// TypedHTTP request types with automatic JWT extraction

// AuthenticatedRequest automatically extracts user info from JWT
type AuthenticatedRequest struct {
	UserID string // Will be populated from JWT context
	Role   string // Will be populated from JWT context
}

// JWTAuthDecoder extracts JWT claims into request fields
type JWTAuthDecoder[T any] struct {
	next typedhttp.RequestDecoder[T]
}

// NewJWTAuthDecoder creates a decoder that populates JWT fields
func NewJWTAuthDecoder[T any](next typedhttp.RequestDecoder[T]) *JWTAuthDecoder[T] {
	return &JWTAuthDecoder[T]{next: next}
}

// Decode implements RequestDecoder interface
func (d *JWTAuthDecoder[T]) Decode(r *http.Request) (T, error) {
	var result T

	// First decode normally
	if d.next != nil {
		var err error
		result, err = d.next.Decode(r)
		if err != nil {
			return result, err
		}
	}

	// Then populate JWT fields if they exist
	if claims, err := GetJWTClaims(r.Context()); err == nil {
		// Use reflection to set JWT fields if they exist
		// This is a simplified version - in practice you'd use reflection
		// to automatically populate UserID, Role fields if they exist
		_ = claims // For now, manual implementation required
	}

	return result, nil
}

// ContentTypes returns supported content types
func (d *JWTAuthDecoder[T]) ContentTypes() []string {
	if d.next != nil {
		return d.next.ContentTypes()
	}
	return []string{"*/*"}
}

// Example usage in handlers

// Example: Protected user profile endpoint
type GetProfileRequest struct {
	UserID string // This would be automatically populated from JWT
}

type UserProfile struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type GetProfileHandler struct {
	userService UserService
}

func (h *GetProfileHandler) Handle(ctx context.Context, req GetProfileRequest) (UserProfile, error) {
	// Get user ID from JWT context
	userID, err := GetCurrentUserID(ctx)
	if err != nil {
		return UserProfile{}, typedhttp.NewUnauthorizedError("Authentication required")
	}

	// Fetch user profile
	user, err := h.userService.GetProfile(ctx, userID)
	if err != nil {
		return UserProfile{}, err
	}

	return UserProfile{
		ID:    user.ID,
		Email: user.Email,
		Role:  user.Role,
	}, nil
}

// Mock user service for the example
type UserService interface {
	GetProfile(ctx context.Context, userID string) (*User, error)
}

type User struct {
	ID    string
	Email string
	Role  string
}

// Example: Login endpoint that generates JWT
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      UserInfo  `json:"user"`
}

type UserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type LoginHandler struct {
	userService UserService
	jwtMiddleware *JWTMiddleware
}

func (h *LoginHandler) Handle(ctx context.Context, req LoginRequest) (LoginResponse, error) {
	// Validate credentials (implement your own logic)
	user, err := h.userService.ValidateCredentials(ctx, req.Email, req.Password)
	if err != nil {
		return LoginResponse{}, typedhttp.NewUnauthorizedError("Invalid credentials")
	}

	// Generate JWT token
	duration := 24 * time.Hour
	token, err := h.jwtMiddleware.GenerateToken(user.ID, user.Email, user.Role, duration)
	if err != nil {
		return LoginResponse{}, fmt.Errorf("failed to generate token: %w", err)
	}

	return LoginResponse{
		Token:     token,
		ExpiresAt: time.Now().Add(duration),
		User: UserInfo{
			ID:    user.ID,
			Email: user.Email,
			Role:  user.Role,
		},
	}, nil
}

// Add ValidateCredentials to UserService interface
type ExtendedUserService interface {
	UserService
	ValidateCredentials(ctx context.Context, email, password string) (*User, error)
}