package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types for JWT middleware testing
type AuthenticatedRequest struct {
	UserID string `json:"user_id"`
	Action string `json:"action"`
}

type AuthenticatedResponse struct {
	Message string `json:"message"`
	UserID  string `json:"user_id"`
}

// Test JWT claims
type TestClaims struct {
	UserID string   `json:"user_id"`
	Email  string   `json:"email"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

// Test helper to generate JWT tokens
func generateTestJWT(t *testing.T, secret []byte, claims TestClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secret)
	require.NoError(t, err)
	return tokenString
}

// Test helper to generate RSA key pair
func generateRSAKeyPair(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return privateKey, &privateKey.PublicKey
}

// TestJWTMiddleware_Configuration tests JWT middleware configuration
func TestJWTMiddleware_Configuration(t *testing.T) {
	secret := []byte("test-secret-key")
	
	t.Run("default_configuration", func(t *testing.T) {
		middleware := NewJWTMiddleware(secret)
		assert.NotNil(t, middleware)
		
		// Should have default configuration
		config := middleware.GetConfig()
		assert.Equal(t, "Authorization", config.TokenHeader)
		assert.Equal(t, "Bearer ", config.TokenPrefix)
		assert.Equal(t, jwt.SigningMethodHS256, config.SigningMethod)
		assert.Equal(t, 1*time.Hour, config.TokenExpiry)
	})
	
	t.Run("custom_configuration", func(t *testing.T) {
		middleware := NewJWTMiddleware(secret,
			WithTokenHeader("X-Auth-Token"),
			WithTokenPrefix("Token "),
			WithSigningMethod(jwt.SigningMethodHS512),
			WithTokenExpiry(30*time.Minute),
		)
		
		config := middleware.GetConfig()
		assert.Equal(t, "X-Auth-Token", config.TokenHeader)
		assert.Equal(t, "Token ", config.TokenPrefix)
		assert.Equal(t, jwt.SigningMethodHS512, config.SigningMethod)
		assert.Equal(t, 30*time.Minute, config.TokenExpiry)
	})
	
	t.Run("rsa_key_configuration", func(t *testing.T) {
		privateKey, publicKey := generateRSAKeyPair(t)
		
		middleware := NewJWTMiddleware(nil,
			WithRSAKeys(privateKey, publicKey),
			WithSigningMethod(jwt.SigningMethodRS256),
		)
		
		config := middleware.GetConfig()
		assert.Equal(t, jwt.SigningMethodRS256, config.SigningMethod)
		assert.NotNil(t, config.PrivateKey)
		assert.NotNil(t, config.PublicKey)
	})
}

// TestJWTMiddleware_TokenExtraction tests token extraction from requests
func TestJWTMiddleware_TokenExtraction(t *testing.T) {
	secret := []byte("test-secret")
	middleware := NewJWTMiddleware(secret)
	
	tests := []struct {
		name          string
		setupRequest  func(*http.Request)
		expectedToken string
		shouldExtract bool
	}{
		{
			name: "valid_bearer_token",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer valid.jwt.token")
			},
			expectedToken: "valid.jwt.token",
			shouldExtract: true,
		},
		{
			name: "missing_authorization_header",
			setupRequest: func(r *http.Request) {
				// No Authorization header
			},
			expectedToken: "",
			shouldExtract: false,
		},
		{
			name: "invalid_token_prefix",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Basic invalid.token")
			},
			expectedToken: "",
			shouldExtract: false,
		},
		{
			name: "empty_token",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer ")
			},
			expectedToken: "",
			shouldExtract: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			tt.setupRequest(req)
			
			token, ok := middleware.ExtractToken(req)
			assert.Equal(t, tt.shouldExtract, ok)
			assert.Equal(t, tt.expectedToken, token)
		})
	}
}

// TestJWTMiddleware_TokenValidation tests JWT token validation
func TestJWTMiddleware_TokenValidation(t *testing.T) {
	secret := []byte("test-secret-key")
	middleware := NewJWTMiddleware(secret)
	
	t.Run("valid_token", func(t *testing.T) {
		claims := TestClaims{
			UserID: "user123",
			Email:  "test@example.com",
			Roles:  []string{"user", "admin"},
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				Subject:   "user123",
			},
		}
		
		tokenString := generateTestJWT(t, secret, claims)
		
		parsedClaims, err := middleware.ValidateToken(tokenString)
		require.NoError(t, err)
		assert.Equal(t, "user123", parsedClaims["user_id"])
		assert.Equal(t, "test@example.com", parsedClaims["email"])
	})
	
	t.Run("expired_token", func(t *testing.T) {
		claims := TestClaims{
			UserID: "user123",
			Email:  "test@example.com",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // Expired
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			},
		}
		
		tokenString := generateTestJWT(t, secret, claims)
		
		_, err := middleware.ValidateToken(tokenString)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})
	
	t.Run("invalid_signature", func(t *testing.T) {
		wrongSecret := []byte("wrong-secret")
		claims := TestClaims{
			UserID: "user123",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			},
		}
		
		tokenString := generateTestJWT(t, wrongSecret, claims)
		
		_, err := middleware.ValidateToken(tokenString)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "signature")
	})
	
	t.Run("malformed_token", func(t *testing.T) {
		_, err := middleware.ValidateToken("invalid.token.format")
		assert.Error(t, err)
	})
}

// TestJWTMiddleware_HTTPMiddleware tests JWT as HTTP middleware
func TestJWTMiddleware_HTTPMiddleware(t *testing.T) {
	secret := []byte("test-secret")
	middleware := NewJWTMiddleware(secret)
	
	// Test handler that checks for user context
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := r.Context().Value(UserContextKey).(*User)
		if !ok {
			http.Error(w, "No user in context", http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"user_id": user.ID,
			"email":   user.Email,
		})
	})
	
	httpMiddleware := middleware.HTTPMiddleware()
	handler := httpMiddleware(testHandler)
	
	t.Run("successful_authentication", func(t *testing.T) {
		claims := TestClaims{
			UserID: "user123",
			Email:  "test@example.com",
			Roles:  []string{"user"},
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
				Subject:   "user123",
			},
		}
		
		token := generateTestJWT(t, secret, claims)
		
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		
		handler.ServeHTTP(rr, req)
		
		assert.Equal(t, http.StatusOK, rr.Code)
		
		var response map[string]string
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "user123", response["user_id"])
		assert.Equal(t, "test@example.com", response["email"])
	})
	
	t.Run("missing_token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		rr := httptest.NewRecorder()
		
		handler.ServeHTTP(rr, req)
		
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), "missing")
	})
	
	t.Run("invalid_token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid.token")
		rr := httptest.NewRecorder()
		
		handler.ServeHTTP(rr, req)
		
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), "invalid")
	})
}

// TestJWTMiddleware_TypedMiddleware tests JWT as typed middleware
func TestJWTMiddleware_TypedMiddleware(t *testing.T) {
	secret := []byte("test-secret")
	middleware := NewJWTMiddleware(secret)
	
	t.Run("successful_typed_middleware", func(t *testing.T) {
		claims := TestClaims{
			UserID: "user456",
			Email:  "typed@example.com",
			Roles:  []string{"admin"},
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
				Subject:   "user456",
			},
		}
		
		token := generateTestJWT(t, secret, claims)
		
		// Create context with the token (simulating HTTP extraction)
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		ctx := context.WithValue(req.Context(), "http_request", req)
		
		authReq := &AuthenticatedRequest{
			UserID: "initial",
			Action: "test",
		}
		
		// Execute typed middleware
		newCtx, err := middleware.Before(ctx, authReq)
		require.NoError(t, err)
		
		// Verify user was added to context
		user, ok := newCtx.Value(UserContextKey).(*User)
		require.True(t, ok)
		assert.Equal(t, "user456", user.ID)
		assert.Equal(t, "typed@example.com", user.Email)
		assert.Contains(t, user.Roles, "admin")
	})
	
	t.Run("authentication_failure", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		// No Authorization header
		ctx := context.WithValue(req.Context(), "http_request", req)
		
		authReq := &AuthenticatedRequest{
			UserID: "initial",
			Action: "test",
		}
		
		_, err := middleware.Before(ctx, authReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "authentication")
	})
}

// TestJWTMiddleware_CustomClaimsExtractor tests custom claims extraction
func TestJWTMiddleware_CustomClaimsExtractor(t *testing.T) {
	secret := []byte("test-secret")
	
	// Custom claims extractor that maps JWT claims to User
	customExtractor := func(claims jwt.MapClaims) (*User, error) {
		userID, ok := claims["sub"].(string)
		if !ok {
			return nil, ErrInvalidClaims
		}
		
		email, _ := claims["email"].(string)
		
		// Extract roles from custom claim
		var roles []string
		if rolesInterface, ok := claims["custom_roles"].([]interface{}); ok {
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
	
	middleware := NewJWTMiddleware(secret, WithClaimsExtractor(customExtractor))
	
	t.Run("custom_claims_extraction", func(t *testing.T) {
		claims := jwt.MapClaims{
			"sub":          "custom123",
			"email":        "custom@example.com",
			"custom_roles": []interface{}{"viewer", "editor"},
			"exp":          time.Now().Add(1 * time.Hour).Unix(),
		}
		
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString(secret)
		require.NoError(t, err)
		
		parsedClaims, err := middleware.ValidateToken(tokenString)
		require.NoError(t, err)
		
		user, err := customExtractor(parsedClaims)
		require.NoError(t, err)
		
		assert.Equal(t, "custom123", user.ID)
		assert.Equal(t, "custom@example.com", user.Email)
		assert.Equal(t, []string{"viewer", "editor"}, user.Roles)
	})
}

// TestJWTMiddleware_RefreshToken tests refresh token functionality
func TestJWTMiddleware_RefreshToken(t *testing.T) {
	secret := []byte("test-secret")
	middleware := NewJWTMiddleware(secret, WithRefreshTokenSupport(true))
	
	t.Run("generate_token_pair", func(t *testing.T) {
		user := &User{
			ID:    "refresh123",
			Email: "refresh@example.com",
			Roles: []string{"user"},
		}
		
		tokenPair, err := middleware.GenerateTokenPair(user)
		require.NoError(t, err)
		
		assert.NotEmpty(t, tokenPair.AccessToken)
		assert.NotEmpty(t, tokenPair.RefreshToken)
		assert.True(t, tokenPair.ExpiresAt.After(time.Now()))
	})
	
	t.Run("refresh_access_token", func(t *testing.T) {
		user := &User{
			ID:    "refresh456",
			Email: "refresh2@example.com",
			Roles: []string{"admin"},
		}
		
		// Generate initial token pair
		tokenPair, err := middleware.GenerateTokenPair(user)
		require.NoError(t, err)
		
		// Refresh the access token
		newTokenPair, err := middleware.RefreshAccessToken(tokenPair.RefreshToken)
		require.NoError(t, err)
		
		assert.NotEmpty(t, newTokenPair.AccessToken)
		assert.NotEqual(t, tokenPair.AccessToken, newTokenPair.AccessToken)
		assert.True(t, newTokenPair.ExpiresAt.After(tokenPair.ExpiresAt))
	})
	
	t.Run("invalid_refresh_token", func(t *testing.T) {
		_, err := middleware.RefreshAccessToken("invalid.refresh.token")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid")
	})
}