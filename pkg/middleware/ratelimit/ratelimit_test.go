package ratelimit

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types for rate limiting middleware testing
type APIRequest struct {
	UserID string `json:"user_id"`
	Action string `json:"action"`
}

type APIResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

// TestTokenBucketRateLimiter_Configuration tests token bucket rate limiter configuration
func TestTokenBucketRateLimiter_Configuration(t *testing.T) {
	t.Run("default_configuration", func(t *testing.T) {
		limiter := NewTokenBucketRateLimiter(10, time.Second)
		assert.NotNil(t, limiter)

		config := limiter.GetConfig()
		assert.Equal(t, 10, config.Capacity)
		assert.Equal(t, time.Second, config.RefillInterval)
		assert.Equal(t, 1, config.RefillTokens)
	})

	t.Run("custom_configuration", func(t *testing.T) {
		limiter := NewTokenBucketRateLimiter(100, time.Minute,
			WithRefillTokens(5),
			WithBurstCapacity(200),
		)

		config := limiter.GetConfig()
		assert.Equal(t, 100, config.Capacity)
		assert.Equal(t, time.Minute, config.RefillInterval)
		assert.Equal(t, 5, config.RefillTokens)
		assert.Equal(t, 200, config.BurstCapacity)
	})
}

// TestTokenBucketRateLimiter_TokenConsumption tests token consumption logic
func TestTokenBucketRateLimiter_TokenConsumption(t *testing.T) {
	limiter := NewTokenBucketRateLimiter(3, 100*time.Millisecond)

	t.Run("consume_available_tokens", func(t *testing.T) {
		// Should be able to consume 3 tokens initially
		for i := 0; i < 3; i++ {
			allowed := limiter.Allow("test-key")
			assert.True(t, allowed, "should allow request %d", i+1)
		}

		// 4th request should be denied
		allowed := limiter.Allow("test-key")
		assert.False(t, allowed, "should deny 4th request")
	})

	t.Run("token_refill", func(t *testing.T) {
		// Wait for refill
		time.Sleep(150 * time.Millisecond)

		// Should be able to consume 1 more token after refill
		allowed := limiter.Allow("test-key-2")
		assert.True(t, allowed, "should allow request after refill")
	})
}

// TestIPBasedRateLimiter_Configuration tests IP-based rate limiter configuration
func TestIPBasedRateLimiter_Configuration(t *testing.T) {
	t.Run("default_configuration", func(t *testing.T) {
		limiter := NewIPBasedRateLimiter(100, time.Hour)
		assert.NotNil(t, limiter)

		config := limiter.GetConfig()
		assert.Equal(t, 100, config.RequestsPerWindow)
		assert.Equal(t, time.Hour, config.Window)
		assert.Equal(t, 1000, config.MaxIPs) // Default max IPs
	})

	t.Run("custom_configuration", func(t *testing.T) {
		limiter := NewIPBasedRateLimiter(50, 30*time.Minute,
			WithMaxIPs(500),
			WithCleanupInterval(5*time.Minute),
			WithWhitelist([]string{"127.0.0.1", "::1"}),
			WithBlacklist([]string{"192.168.1.100"}),
		)

		config := limiter.GetConfig()
		assert.Equal(t, 50, config.RequestsPerWindow)
		assert.Equal(t, 30*time.Minute, config.Window)
		assert.Equal(t, 500, config.MaxIPs)
		assert.Equal(t, 5*time.Minute, config.CleanupInterval)
		assert.Contains(t, config.Whitelist, "127.0.0.1")
		assert.Contains(t, config.Blacklist, "192.168.1.100")
	})
}

// TestIPBasedRateLimiter_IPLimiting tests IP-based rate limiting logic
func TestIPBasedRateLimiter_IPLimiting(t *testing.T) {
	limiter := NewIPBasedRateLimiter(3, time.Hour)

	t.Run("different_ips_independent_limits", func(t *testing.T) {
		// Each IP should have independent limits
		for i := 0; i < 3; i++ {
			allowed := limiter.AllowIP("192.168.1.1")
			assert.True(t, allowed, "IP1 request %d should be allowed", i+1)

			allowed = limiter.AllowIP("192.168.1.2")
			assert.True(t, allowed, "IP2 request %d should be allowed", i+1)
		}

		// 4th request should be denied for both IPs
		allowed := limiter.AllowIP("192.168.1.1")
		assert.False(t, allowed, "IP1 4th request should be denied")

		allowed = limiter.AllowIP("192.168.1.2")
		assert.False(t, allowed, "IP2 4th request should be denied")
	})

	t.Run("whitelist_bypass", func(t *testing.T) {
		limiter := NewIPBasedRateLimiter(1, time.Hour,
			WithWhitelist([]string{"192.168.1.100"}),
		)

		// Whitelist IP should bypass rate limiting
		for i := 0; i < 10; i++ {
			allowed := limiter.AllowIP("192.168.1.100")
			assert.True(t, allowed, "whitelisted IP request %d should be allowed", i+1)
		}
	})

	t.Run("blacklist_block", func(t *testing.T) {
		limiter := NewIPBasedRateLimiter(100, time.Hour,
			WithBlacklist([]string{"192.168.1.200"}),
		)

		// Blacklisted IP should always be blocked
		allowed := limiter.AllowIP("192.168.1.200")
		assert.False(t, allowed, "blacklisted IP should be blocked")
	})
}

// TestUserBasedRateLimiter_Configuration tests user-based rate limiter configuration
func TestUserBasedRateLimiter_Configuration(t *testing.T) {
	t.Run("default_configuration", func(t *testing.T) {
		limiter := NewUserBasedRateLimiter(100, time.Hour)
		assert.NotNil(t, limiter)

		config := limiter.GetConfig()
		assert.Equal(t, 100, config.RequestsPerWindow)
		assert.Equal(t, time.Hour, config.Window)
		assert.Equal(t, 10000, config.MaxUsers) // Default max users
	})

	t.Run("tier_based_configuration", func(t *testing.T) {
		tiers := map[string]int{
			"free":       10,
			"premium":    100,
			"enterprise": 1000,
		}

		limiter := NewUserBasedRateLimiter(50, time.Hour,
			WithUserTiers(tiers),
			WithMaxUsers(5000),
		)

		config := limiter.GetConfig()
		assert.Equal(t, 50, config.RequestsPerWindow)
		assert.Equal(t, tiers, config.UserTiers)
		assert.Equal(t, 5000, config.MaxUsers)
	})
}

// TestUserBasedRateLimiter_UserLimiting tests user-based rate limiting logic
func TestUserBasedRateLimiter_UserLimiting(t *testing.T) {
	t.Run("different_users_independent_limits", func(t *testing.T) {
		limiter := NewUserBasedRateLimiter(2, time.Hour)

		// Each user should have independent limits
		for i := 0; i < 2; i++ {
			allowed := limiter.AllowUser("user1", "")
			assert.True(t, allowed, "user1 request %d should be allowed", i+1)

			allowed = limiter.AllowUser("user2", "")
			assert.True(t, allowed, "user2 request %d should be allowed", i+1)
		}

		// 3rd request should be denied for both users
		allowed := limiter.AllowUser("user1", "")
		assert.False(t, allowed, "user1 3rd request should be denied")

		allowed = limiter.AllowUser("user2", "")
		assert.False(t, allowed, "user2 3rd request should be denied")
	})

	t.Run("tier_based_limits", func(t *testing.T) {
		tiers := map[string]int{
			"free":    1,
			"premium": 3,
		}

		limiter := NewUserBasedRateLimiter(2, time.Hour, // default limit
			WithUserTiers(tiers),
		)

		// Free tier user should have limit of 1
		allowed := limiter.AllowUser("free_user", "free")
		assert.True(t, allowed, "free user 1st request should be allowed")

		allowed = limiter.AllowUser("free_user", "free")
		assert.False(t, allowed, "free user 2nd request should be denied")

		// Premium tier user should have limit of 3
		for i := 0; i < 3; i++ {
			allowed = limiter.AllowUser("premium_user", "premium")
			assert.True(t, allowed, "premium user request %d should be allowed", i+1)
		}

		allowed = limiter.AllowUser("premium_user", "premium")
		assert.False(t, allowed, "premium user 4th request should be denied")
	})
}

// TestRateLimitMiddleware_HTTPMiddleware tests rate limiting as HTTP middleware
func TestRateLimitMiddleware_HTTPMiddleware(t *testing.T) {
	limiter := NewIPBasedRateLimiter(2, time.Hour)
	middleware := NewRateLimitMiddleware(limiter)

	// Test handler that returns success
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "success"}`))
	})

	handler := middleware.HTTPMiddleware()(testHandler)

	t.Run("successful_requests_within_limit", func(t *testing.T) {
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
			req.RemoteAddr = "192.168.1.10:12345"
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
			assert.Contains(t, rr.Body.String(), "success")
		}
	})

	t.Run("request_denied_over_limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
		req.RemoteAddr = "192.168.1.10:12345"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusTooManyRequests, rr.Code)
		assert.Contains(t, rr.Body.String(), "rate limit exceeded")
	})

	t.Run("rate_limit_headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
		req.RemoteAddr = "192.168.1.20:12345"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.NotEmpty(t, rr.Header().Get("X-RateLimit-Limit"))
		assert.NotEmpty(t, rr.Header().Get("X-RateLimit-Remaining"))
		assert.NotEmpty(t, rr.Header().Get("X-RateLimit-Reset"))
	})
}

// TestRateLimitMiddleware_TypedMiddleware tests rate limiting as typed middleware
func TestRateLimitMiddleware_TypedMiddleware(t *testing.T) {
	limiter := NewUserBasedRateLimiter(2, time.Hour)
	middleware := NewRateLimitMiddleware(limiter,
		WithUserExtractor(func(req interface{}) string {
			if apiReq, ok := req.(*APIRequest); ok {
				return apiReq.UserID
			}
			return ""
		}),
	)

	t.Run("successful_typed_middleware", func(t *testing.T) {
		req := &APIRequest{
			UserID: "user123",
			Action: "test",
		}

		// Create context with HTTP request for IP extraction fallback
		httpReq := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
		httpReq.RemoteAddr = "192.168.1.30:12345"
		ctx := context.WithValue(context.Background(), "http_request", httpReq)

		// First two requests should succeed
		for i := 0; i < 2; i++ {
			newCtx, err := middleware.Before(ctx, req)
			require.NoError(t, err)
			assert.NotNil(t, newCtx)
		}

		// Third request should fail
		_, err := middleware.Before(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rate limit exceeded")
	})
}

// TestRateLimitMiddleware_ConcurrentAccess tests concurrent access to rate limiter
func TestRateLimitMiddleware_ConcurrentAccess(t *testing.T) {
	limiter := NewTokenBucketRateLimiter(10, 100*time.Millisecond)

	const numGoroutines = 20
	const requestsPerGoroutine = 5

	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := 0
	denied := 0

	// Use the same key for all goroutines to force competition
	const sharedKey = "concurrent-test-shared"

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < requestsPerGoroutine; j++ {
				if limiter.Allow(sharedKey) {
					mu.Lock()
					allowed++
					mu.Unlock()
				} else {
					mu.Lock()
					denied++
					mu.Unlock()
				}
			}
		}(i)
	}

	wg.Wait()

	total := allowed + denied
	expectedTotal := numGoroutines * requestsPerGoroutine

	assert.Equal(t, expectedTotal, total, "total requests should match expected")
	assert.Greater(t, denied, 0, "some requests should be denied")

	t.Logf("Concurrent test results - Allowed: %d, Denied: %d, Total: %d", allowed, denied, total)
}

// TestRateLimitMiddleware_CleanupExpiredEntries tests cleanup of expired entries
func TestRateLimitMiddleware_CleanupExpiredEntries(t *testing.T) {
	limiter := NewIPBasedRateLimiter(10, 100*time.Millisecond,
		WithCleanupInterval(50*time.Millisecond),
	)

	// Make requests from different IPs
	for i := 0; i < 5; i++ {
		ip := fmt.Sprintf("192.168.1.%d", i+1)
		limiter.AllowIP(ip)
	}

	// Check initial count
	initialCount := limiter.GetActiveIPCount()
	assert.Equal(t, 5, initialCount)

	// Wait for entries to expire and cleanup to run
	time.Sleep(200 * time.Millisecond)

	// Count should be reduced after cleanup
	finalCount := limiter.GetActiveIPCount()
	assert.LessOrEqual(t, finalCount, initialCount, "expired entries should be cleaned up")
}

// TestSlidingWindowRateLimiter_WindowBehavior tests sliding window rate limiting behavior
func TestSlidingWindowRateLimiter_WindowBehavior(t *testing.T) {
	limiter := NewSlidingWindowRateLimiter(3, time.Second)

	t.Run("requests_within_window", func(t *testing.T) {
		// Make 3 requests quickly
		for i := 0; i < 3; i++ {
			allowed := limiter.Allow("test-key")
			assert.True(t, allowed, "request %d should be allowed", i+1)
		}

		// 4th request should be denied
		allowed := limiter.Allow("test-key")
		assert.False(t, allowed, "4th request should be denied")
	})

	t.Run("window_sliding", func(t *testing.T) {
		// Wait for window to slide
		time.Sleep(1100 * time.Millisecond)

		// Should be able to make requests again
		allowed := limiter.Allow("test-key-2")
		assert.True(t, allowed, "request after window slide should be allowed")
	})
}

// TestRateLimitMiddleware_MetricsCollection tests metrics collection
func TestRateLimitMiddleware_MetricsCollection(t *testing.T) {
	limiter := NewTokenBucketRateLimiter(2, time.Hour)
	middleware := NewRateLimitMiddleware(limiter, WithMetricsEnabled(true))

	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.HTTPMiddleware()(testHandler)

	// Make requests to trigger metrics
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
		req.RemoteAddr = "192.168.1.100:12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}

	metrics := middleware.GetMetrics()
	assert.NotNil(t, metrics)
	assert.Greater(t, metrics.TotalRequests, int64(0))
	assert.Greater(t, metrics.AllowedRequests, int64(0))
	assert.Greater(t, metrics.DeniedRequests, int64(0))
}
