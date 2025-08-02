package auth

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Common errors
var (
	ErrTokenMissing     = errors.New("authentication token missing")
	ErrTokenInvalid     = errors.New("authentication token invalid")
	ErrTokenExpired     = errors.New("authentication token expired")
	ErrInvalidSignature = errors.New("invalid token signature")
	ErrInvalidClaims    = errors.New("invalid token claims")
)

// User represents an authenticated user
type User struct {
	ID    string   `json:"id"`
	Email string   `json:"email"`
	Roles []string `json:"roles"`
}

// TokenPair represents access and refresh token pair
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// Context keys
type contextKey string

const (
	UserContextKey contextKey = "user"
)

// JWTConfig holds JWT middleware configuration
type JWTConfig struct {
	Secret         []byte
	PrivateKey     *rsa.PrivateKey
	PublicKey      *rsa.PublicKey
	TokenHeader    string
	TokenPrefix    string
	SigningMethod  jwt.SigningMethod
	TokenExpiry    time.Duration
	ClaimsExtractor func(jwt.MapClaims) (*User, error)
	RefreshSupport bool
}

// JWTMiddleware provides JWT authentication middleware
type JWTMiddleware struct {
	config JWTConfig
}

// JWTOption configures JWT middleware
type JWTOption func(*JWTConfig)

// WithTokenHeader sets the header name for token extraction
func WithTokenHeader(header string) JWTOption {
	return func(c *JWTConfig) {
		c.TokenHeader = header
	}
}

// WithTokenPrefix sets the token prefix
func WithTokenPrefix(prefix string) JWTOption {
	return func(c *JWTConfig) {
		c.TokenPrefix = prefix
	}
}

// WithSigningMethod sets the JWT signing method
func WithSigningMethod(method jwt.SigningMethod) JWTOption {
	return func(c *JWTConfig) {
		c.SigningMethod = method
	}
}

// WithTokenExpiry sets the token expiry duration
func WithTokenExpiry(expiry time.Duration) JWTOption {
	return func(c *JWTConfig) {
		c.TokenExpiry = expiry
	}
}

// WithRSAKeys sets RSA private and public keys
func WithRSAKeys(privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey) JWTOption {
	return func(c *JWTConfig) {
		c.PrivateKey = privateKey
		c.PublicKey = publicKey
	}
}

// WithClaimsExtractor sets a custom claims extractor function
func WithClaimsExtractor(extractor func(jwt.MapClaims) (*User, error)) JWTOption {
	return func(c *JWTConfig) {
		c.ClaimsExtractor = extractor
	}
}

// WithRefreshTokenSupport enables refresh token support
func WithRefreshTokenSupport(enabled bool) JWTOption {
	return func(c *JWTConfig) {
		c.RefreshSupport = enabled
	}
}

// NewJWTMiddleware creates a new JWT middleware with the given secret and options
func NewJWTMiddleware(secret []byte, opts ...JWTOption) *JWTMiddleware {
	config := JWTConfig{
		Secret:        secret,
		TokenHeader:   "Authorization",
		TokenPrefix:   "Bearer ",
		SigningMethod: jwt.SigningMethodHS256,
		TokenExpiry:   1 * time.Hour,
		ClaimsExtractor: defaultClaimsExtractor,
	}

	for _, opt := range opts {
		opt(&config)
	}

	return &JWTMiddleware{config: config}
}

// GetConfig returns the middleware configuration
func (m *JWTMiddleware) GetConfig() JWTConfig {
	return m.config
}

// ExtractToken extracts JWT token from HTTP request
func (m *JWTMiddleware) ExtractToken(r *http.Request) (string, bool) {
	authHeader := r.Header.Get(m.config.TokenHeader)
	if authHeader == "" {
		return "", false
	}

	if !strings.HasPrefix(authHeader, m.config.TokenPrefix) {
		return "", false
	}

	token := strings.TrimPrefix(authHeader, m.config.TokenPrefix)
	if token == "" {
		return "", false
	}

	return token, true
}

// ValidateToken validates a JWT token and returns claims
func (m *JWTMiddleware) ValidateToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if token.Method != m.config.SigningMethod {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Return the appropriate key based on signing method
		switch m.config.SigningMethod {
		case jwt.SigningMethodHS256, jwt.SigningMethodHS512:
			return m.config.Secret, nil
		case jwt.SigningMethodRS256:
			return m.config.PublicKey, nil
		default:
			return nil, fmt.Errorf("unsupported signing method: %v", m.config.SigningMethod)
		}
	})

	if err != nil {
		if strings.Contains(err.Error(), "expired") {
			return nil, ErrTokenExpired
		}
		if strings.Contains(err.Error(), "signature") {
			return nil, ErrInvalidSignature
		}
		return nil, ErrTokenInvalid
	}

	if !token.Valid {
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidClaims
	}

	return claims, nil
}

// HTTPMiddleware returns HTTP middleware function
func (m *JWTMiddleware) HTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token
			tokenString, ok := m.ExtractToken(r)
			if !ok {
				m.writeError(w, http.StatusUnauthorized, "authentication token missing")
				return
			}

			// Validate token
			claims, err := m.ValidateToken(tokenString)
			if err != nil {
				m.writeError(w, http.StatusUnauthorized, "authentication token invalid: "+err.Error())
				return
			}

			// Extract user from claims
			user, err := m.config.ClaimsExtractor(claims)
			if err != nil {
				m.writeError(w, http.StatusUnauthorized, "invalid token claims: "+err.Error())
				return
			}

			// Add user to context
			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Before implements TypedPreMiddleware interface
func (m *JWTMiddleware) Before(ctx context.Context, req interface{}) (context.Context, error) {
	// Extract HTTP request from context
	httpReq, ok := ctx.Value("http_request").(*http.Request)
	if !ok {
		return ctx, errors.New("authentication failed: no HTTP request in context")
	}

	// Extract token
	tokenString, ok := m.ExtractToken(httpReq)
	if !ok {
		return ctx, errors.New("authentication failed: token missing")
	}

	// Validate token
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return ctx, fmt.Errorf("authentication failed: %w", err)
	}

	// Extract user from claims
	user, err := m.config.ClaimsExtractor(claims)
	if err != nil {
		return ctx, fmt.Errorf("authentication failed: %w", err)
	}

	// Add user to context
	return context.WithValue(ctx, UserContextKey, user), nil
}

// GenerateTokenPair generates access and refresh token pair
func (m *JWTMiddleware) GenerateTokenPair(user *User) (*TokenPair, error) {
	if !m.config.RefreshSupport {
		return nil, errors.New("refresh token support not enabled")
	}

	now := time.Now()
	expiresAt := now.Add(m.config.TokenExpiry)

	// Create access token claims
	accessClaims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"roles":   user.Roles,
		"sub":     user.ID,
		"iat":     now.Unix(),
		"exp":     expiresAt.Unix(),
	}

	// Generate access token
	accessToken := jwt.NewWithClaims(m.config.SigningMethod, accessClaims)
	accessTokenString, err := m.signToken(accessToken)
	if err != nil {
		return nil, err
	}

	// Create refresh token claims (longer expiry)
	refreshClaims := jwt.MapClaims{
		"user_id": user.ID,
		"sub":     user.ID,
		"iat":     now.Unix(),
		"exp":     now.Add(7 * 24 * time.Hour).Unix(), // 7 days
		"type":    "refresh",
	}

	// Generate refresh token
	refreshToken := jwt.NewWithClaims(m.config.SigningMethod, refreshClaims)
	refreshTokenString, err := m.signToken(refreshToken)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresAt:    expiresAt,
	}, nil
}

// RefreshAccessToken generates a new access token from refresh token
func (m *JWTMiddleware) RefreshAccessToken(refreshToken string) (*TokenPair, error) {
	// Validate refresh token
	claims, err := m.ValidateToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Verify it's a refresh token
	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "refresh" {
		return nil, errors.New("invalid refresh token: not a refresh token")
	}

	// Extract user ID
	userID, ok := claims["user_id"].(string)
	if !ok {
		return nil, errors.New("invalid refresh token: missing user_id")
	}

	// Create new access token (simplified - in real implementation you'd fetch user data)
	user := &User{
		ID: userID,
		// Note: In a real implementation, you'd fetch the full user data from database
	}

	return m.GenerateTokenPair(user)
}

// signToken signs a JWT token with the appropriate key
func (m *JWTMiddleware) signToken(token *jwt.Token) (string, error) {
	switch m.config.SigningMethod {
	case jwt.SigningMethodHS256, jwt.SigningMethodHS512:
		return token.SignedString(m.config.Secret)
	case jwt.SigningMethodRS256:
		return token.SignedString(m.config.PrivateKey)
	default:
		return "", fmt.Errorf("unsupported signing method: %v", m.config.SigningMethod)
	}
}

// writeError writes an error response
func (m *JWTMiddleware) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

// defaultClaimsExtractor is the default claims extractor
func defaultClaimsExtractor(claims jwt.MapClaims) (*User, error) {
	userID, ok := claims["user_id"].(string)
	if !ok {
		// Try 'sub' as fallback
		if sub, ok := claims["sub"].(string); ok {
			userID = sub
		} else {
			return nil, ErrInvalidClaims
		}
	}

	email, _ := claims["email"].(string)

	// Extract roles
	var roles []string
	if rolesInterface, ok := claims["roles"].([]interface{}); ok {
		for _, role := range rolesInterface {
			if roleStr, ok := role.(string); ok {
				roles = append(roles, roleStr)
			}
		}
	}

	return &User{
		ID:    userID,
		Email: email,
		Roles: roles,
	}, nil
}