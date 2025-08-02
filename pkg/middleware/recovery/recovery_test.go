package recovery

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types for recovery middleware testing
type RecoveryRequest struct {
	Action    string `json:"action"`
	ShouldFail bool   `json:"should_fail"`
}

type RecoveryResponse struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

// TestPanicRecoveryMiddleware_Configuration tests panic recovery middleware configuration
func TestPanicRecoveryMiddleware_Configuration(t *testing.T) {
	t.Run("default_configuration", func(t *testing.T) {
		middleware := NewPanicRecoveryMiddleware()
		assert.NotNil(t, middleware)
		
		config := middleware.GetConfig()
		assert.True(t, config.RecoverPanics)
		assert.True(t, config.LogPanics)
		assert.True(t, config.IncludeStackTrace)
		assert.Equal(t, http.StatusInternalServerError, config.StatusCode)
	})
	
	t.Run("custom_configuration", func(t *testing.T) {
		middleware := NewPanicRecoveryMiddleware(
			WithPanicRecovery(true),
			WithPanicLogging(false),
			WithStackTrace(false),
			WithRecoveryStatusCode(http.StatusServiceUnavailable),
			WithPanicHandler(func(interface{}) error {
				return errors.New("custom panic handler")
			}),
		)
		
		config := middleware.GetConfig()
		assert.True(t, config.RecoverPanics)
		assert.False(t, config.LogPanics)
		assert.False(t, config.IncludeStackTrace)
		assert.Equal(t, http.StatusServiceUnavailable, config.StatusCode)
		assert.NotNil(t, config.PanicHandler)
	})
}

// TestPanicRecoveryMiddleware_HTTPMiddleware tests panic recovery as HTTP middleware
func TestPanicRecoveryMiddleware_HTTPMiddleware(t *testing.T) {
	middleware := NewPanicRecoveryMiddleware()
	
	t.Run("normal_request_processing", func(t *testing.T) {
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message": "success"}`))
		})
		
		handler := middleware.HTTPMiddleware()(testHandler)
		
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()
		
		handler.ServeHTTP(rr, req)
		
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "success")
	})
	
	t.Run("panic_recovery", func(t *testing.T) {
		panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		})
		
		handler := middleware.HTTPMiddleware()(panicHandler)
		
		req := httptest.NewRequest(http.MethodGet, "/panic", nil)
		rr := httptest.NewRecorder()
		
		// This should not panic and should be recovered
		assert.NotPanics(t, func() {
			handler.ServeHTTP(rr, req)
		})
		
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "internal server error")
	})
	
	t.Run("panic_with_custom_handler", func(t *testing.T) {
		customMiddleware := NewPanicRecoveryMiddleware(
			WithPanicHandler(func(panicValue interface{}) error {
				return fmt.Errorf("custom panic: %v", panicValue)
			}),
		)
		
		panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("custom test panic")
		})
		
		handler := customMiddleware.HTTPMiddleware()(panicHandler)
		
		req := httptest.NewRequest(http.MethodGet, "/custom-panic", nil)
		rr := httptest.NewRecorder()
		
		assert.NotPanics(t, func() {
			handler.ServeHTTP(rr, req)
		})
		
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "custom panic")
	})
}

// TestPanicRecoveryMiddleware_TypedMiddleware tests panic recovery as typed middleware
func TestPanicRecoveryMiddleware_TypedMiddleware(t *testing.T) {
	middleware := NewPanicRecoveryMiddleware()
	
	t.Run("normal_typed_processing", func(t *testing.T) {
		req := &RecoveryRequest{
			Action:    "normal",
			ShouldFail: false,
		}
		
		ctx := context.Background()
		newCtx, err := middleware.Before(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, newCtx)
		
		resp := &RecoveryResponse{
			Message: "success",
			Success: true,
		}
		
		finalResp, err := middleware.After(newCtx, req, resp, nil)
		require.NoError(t, err)
		assert.Equal(t, resp, finalResp)
	})
	
	t.Run("error_handling", func(t *testing.T) {
		req := &RecoveryRequest{
			Action:    "error",
			ShouldFail: true,
		}
		
		ctx := context.Background()
		newCtx, _ := middleware.Before(ctx, req)
		
		resp := &RecoveryResponse{
			Message: "failed",
			Success: false,
		}
		
		// Simulate an error
		testErr := errors.New("processing error")
		finalResp, err := middleware.After(newCtx, req, resp, testErr)
		
		// Recovery middleware should not modify the error, just log it
		assert.Error(t, err)
		assert.Equal(t, resp, finalResp)
	})
}

// TestCircuitBreakerMiddleware_Configuration tests circuit breaker middleware configuration
func TestCircuitBreakerMiddleware_Configuration(t *testing.T) {
	t.Run("default_configuration", func(t *testing.T) {
		middleware := NewCircuitBreakerMiddleware("test_service")
		assert.NotNil(t, middleware)
		
		config := middleware.GetConfig()
		assert.Equal(t, "test_service", config.ServiceName)
		assert.Equal(t, 5, config.FailureThreshold)
		assert.Equal(t, 60*time.Second, config.RecoveryTimeout)
		assert.Equal(t, 3, config.MaxRequests)
	})
	
	t.Run("custom_configuration", func(t *testing.T) {
		middleware := NewCircuitBreakerMiddleware("custom_service",
			WithFailureThreshold(10),
			WithRecoveryTimeout(2*time.Minute),
			WithMaxRequests(5),
			WithReadyToTripFunc(func(counts CircuitBreakerCounts) bool {
				return counts.ConsecutiveFailures >= 3
			}),
		)
		
		config := middleware.GetConfig()
		assert.Equal(t, "custom_service", config.ServiceName)
		assert.Equal(t, 10, config.FailureThreshold)
		assert.Equal(t, 2*time.Minute, config.RecoveryTimeout)
		assert.Equal(t, 5, config.MaxRequests)
		assert.NotNil(t, config.ReadyToTrip)
	})
}

// TestCircuitBreakerMiddleware_States tests circuit breaker state transitions
func TestCircuitBreakerMiddleware_States(t *testing.T) {
	middleware := NewCircuitBreakerMiddleware("test_cb",
		WithFailureThreshold(3),
		WithRecoveryTimeout(100*time.Millisecond),
		WithMaxRequests(2),
	)
	
	t.Run("closed_state_normal_operation", func(t *testing.T) {
		// Circuit should start in closed state
		assert.Equal(t, StateClosed, middleware.GetState())
		
		// Successful requests should keep it closed
		for i := 0; i < 5; i++ {
			allowed := middleware.Allow()
			assert.True(t, allowed)
			middleware.RecordSuccess()
		}
		
		assert.Equal(t, StateClosed, middleware.GetState())
	})
	
	t.Run("closed_to_open_state_transition", func(t *testing.T) {
		middleware := NewCircuitBreakerMiddleware("test_cb2",
			WithFailureThreshold(3),
		)
		
		// Record failures to trip the circuit
		for i := 0; i < 3; i++ {
			allowed := middleware.Allow()
			assert.True(t, allowed)
			middleware.RecordFailure()
		}
		
		// Circuit should now be open
		assert.Equal(t, StateOpen, middleware.GetState())
		
		// Requests should be rejected
		allowed := middleware.Allow()
		assert.False(t, allowed)
	})
	
	t.Run("open_to_half_open_state_transition", func(t *testing.T) {
		middleware := NewCircuitBreakerMiddleware("test_cb3",
			WithFailureThreshold(2),
			WithRecoveryTimeout(50*time.Millisecond),
		)
		
		// Trip the circuit
		middleware.Allow()
		middleware.RecordFailure()
		middleware.Allow()
		middleware.RecordFailure()
		
		assert.Equal(t, StateOpen, middleware.GetState())
		
		// Wait for recovery timeout
		time.Sleep(60 * time.Millisecond)
		
		// First request after timeout should transition to half-open
		allowed := middleware.Allow()
		assert.True(t, allowed)
		assert.Equal(t, StateHalfOpen, middleware.GetState())
	})
	
	t.Run("half_open_to_closed_recovery", func(t *testing.T) {
		middleware := NewCircuitBreakerMiddleware("test_cb4",
			WithFailureThreshold(2),
			WithRecoveryTimeout(50*time.Millisecond),
			WithMaxRequests(2),
		)
		
		// Trip the circuit
		middleware.Allow()
		middleware.RecordFailure()
		middleware.Allow()
		middleware.RecordFailure()
		
		// Wait for recovery
		time.Sleep(60 * time.Millisecond)
		
		// Transition to half-open
		middleware.Allow()
		middleware.RecordSuccess()
		
		// Another successful request should close the circuit
		allowed := middleware.Allow()
		assert.True(t, allowed)
		middleware.RecordSuccess()
		
		assert.Equal(t, StateClosed, middleware.GetState())
	})
}

// TestCircuitBreakerMiddleware_HTTPMiddleware tests circuit breaker as HTTP middleware
func TestCircuitBreakerMiddleware_HTTPMiddleware(t *testing.T) {
	middleware := NewCircuitBreakerMiddleware("http_cb",
		WithFailureThreshold(2),
		WithRecoveryTimeout(100*time.Millisecond),
		WithMaxRequests(1),
	)
	
	var requestCount int32
	
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		if count <= 2 {
			// First two requests fail
			http.Error(w, "service error", http.StatusInternalServerError)
			return
		}
		// Subsequent requests succeed
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	})
	
	handler := middleware.HTTPMiddleware()(testHandler)
	
	t.Run("circuit_breaker_http_flow", func(t *testing.T) {
		// First request - should fail but circuit stays closed
		req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr1 := httptest.NewRecorder()
		handler.ServeHTTP(rr1, req1)
		assert.Equal(t, http.StatusInternalServerError, rr1.Code)
		assert.Equal(t, StateClosed, middleware.GetState())
		
		// Second request - should fail and trip circuit
		req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr2 := httptest.NewRecorder()
		handler.ServeHTTP(rr2, req2)
		assert.Equal(t, http.StatusInternalServerError, rr2.Code)
		assert.Equal(t, StateOpen, middleware.GetState())
		
		// Third request - should be rejected by circuit breaker
		req3 := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr3 := httptest.NewRecorder()
		handler.ServeHTTP(rr3, req3)
		assert.Equal(t, http.StatusServiceUnavailable, rr3.Code)
		assert.Contains(t, rr3.Body.String(), "circuit breaker")
		
		// Wait for recovery timeout
		time.Sleep(110 * time.Millisecond)
		
		// Fourth request - should be allowed (half-open) and succeed
		req4 := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr4 := httptest.NewRecorder()
		handler.ServeHTTP(rr4, req4)
		assert.Equal(t, http.StatusOK, rr4.Code)
		assert.Equal(t, StateClosed, middleware.GetState())
	})
}

// TestRetryMiddleware_Configuration tests retry middleware configuration
func TestRetryMiddleware_Configuration(t *testing.T) {
	t.Run("default_configuration", func(t *testing.T) {
		middleware := NewRetryMiddleware()
		assert.NotNil(t, middleware)
		
		config := middleware.GetConfig()
		assert.Equal(t, 3, config.MaxRetries)
		assert.Equal(t, 100*time.Millisecond, config.InitialDelay)
		assert.Equal(t, 2.0, config.BackoffMultiplier)
		assert.Equal(t, 5*time.Second, config.MaxDelay)
	})
	
	t.Run("custom_configuration", func(t *testing.T) {
		middleware := NewRetryMiddleware(
			WithMaxRetries(5),
			WithInitialDelay(200*time.Millisecond),
			WithBackoffMultiplier(1.5),
			WithMaxDelay(10*time.Second),
			WithRetryableErrors([]error{errors.New("custom error")}),
			WithRetryCondition(func(error) bool {
				return true
			}),
		)
		
		config := middleware.GetConfig()
		assert.Equal(t, 5, config.MaxRetries)
		assert.Equal(t, 200*time.Millisecond, config.InitialDelay)
		assert.Equal(t, 1.5, config.BackoffMultiplier)
		assert.Equal(t, 10*time.Second, config.MaxDelay)
		assert.Len(t, config.RetryableErrors, 1)
		assert.NotNil(t, config.RetryCondition)
	})
}

// TestRetryMiddleware_HTTPMiddleware tests retry as HTTP middleware
func TestRetryMiddleware_HTTPMiddleware(t *testing.T) {
	var requestCount int32
	
	middleware := NewRetryMiddleware(
		WithMaxRetries(3),
		WithInitialDelay(10*time.Millisecond),
	)
	
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		if count < 3 {
			// First two attempts fail
			http.Error(w, "temporary failure", http.StatusServiceUnavailable)
			return
		}
		// Third attempt succeeds
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	})
	
	handler := middleware.HTTPMiddleware()(testHandler)
	
	t.Run("retry_until_success", func(t *testing.T) {
		atomic.StoreInt32(&requestCount, 0)
		
		req := httptest.NewRequest(http.MethodGet, "/retry", nil)
		rr := httptest.NewRecorder()
		
		handler.ServeHTTP(rr, req)
		
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "success")
		assert.Equal(t, int32(3), atomic.LoadInt32(&requestCount))
	})
	
	t.Run("retry_exhausted", func(t *testing.T) {
		atomic.StoreInt32(&requestCount, 0)
		
		// Handler that always fails
		failingHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&requestCount, 1)
			http.Error(w, "persistent failure", http.StatusInternalServerError)
		})
		
		retryHandler := middleware.HTTPMiddleware()(failingHandler)
		
		req := httptest.NewRequest(http.MethodGet, "/failing", nil)
		rr := httptest.NewRecorder()
		
		retryHandler.ServeHTTP(rr, req)
		
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		// Should try initial + 3 retries = 4 total attempts
		assert.Equal(t, int32(4), atomic.LoadInt32(&requestCount))
	})
}

// TestRetryMiddleware_TypedMiddleware tests retry as typed middleware
func TestRetryMiddleware_TypedMiddleware(t *testing.T) {
	var attemptCount int32
	
	middleware := NewRetryMiddleware(
		WithMaxRetries(2),
		WithInitialDelay(10*time.Millisecond),
	)
	
	t.Run("typed_retry_success", func(t *testing.T) {
		atomic.StoreInt32(&attemptCount, 0)
		
		ctx := context.Background()
		
		// Simulate a function that fails twice then succeeds
		retryFunc := func() error {
			count := atomic.AddInt32(&attemptCount, 1)
			if count < 3 {
				return errors.New("temporary failure")
			}
			return nil
		}
		
		err := middleware.ExecuteWithRetry(ctx, retryFunc)
		assert.NoError(t, err)
		assert.Equal(t, int32(3), atomic.LoadInt32(&attemptCount))
	})
	
	t.Run("typed_retry_exhausted", func(t *testing.T) {
		atomic.StoreInt32(&attemptCount, 0)
		
		// Function that always fails
		failingFunc := func() error {
			atomic.AddInt32(&attemptCount, 1)
			return errors.New("persistent failure")
		}
		
		ctx := context.Background()
		err := middleware.ExecuteWithRetry(ctx, failingFunc)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "persistent failure")
		// Should try initial + 2 retries = 3 total attempts
		assert.Equal(t, int32(3), atomic.LoadInt32(&attemptCount))
	})
}

// TestRecoveryMiddleware_CombinedUsage tests using all recovery middleware together
func TestRecoveryMiddleware_CombinedUsage(t *testing.T) {
	// Create all recovery middleware
	panicMW := NewPanicRecoveryMiddleware()
	circuitMW := NewCircuitBreakerMiddleware("combined_service",
		WithFailureThreshold(2),
		WithRecoveryTimeout(100*time.Millisecond),
	)
	retryMW := NewRetryMiddleware(
		WithMaxRetries(2),
		WithInitialDelay(10*time.Millisecond),
	)
	
	// Combine middleware
	combined := NewRecoveryMiddleware(panicMW, circuitMW, retryMW)
	
	var requestCount int32
	
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		
		switch r.URL.Path {
		case "/panic":
			panic("test panic")
		case "/failing":
			if count < 2 {
				http.Error(w, "temporary failure", http.StatusServiceUnavailable)
				return
			}
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"combined": "success"}`))
	})
	
	handler := combined.HTTPMiddleware()(testHandler)
	
	t.Run("combined_recovery_features", func(t *testing.T) {
		// Test panic recovery
		panicReq := httptest.NewRequest(http.MethodGet, "/panic", nil)
		panicRR := httptest.NewRecorder()
		
		assert.NotPanics(t, func() {
			handler.ServeHTTP(panicRR, panicReq)
		})
		assert.Equal(t, http.StatusInternalServerError, panicRR.Code)
		
		// Test retry with eventual success
		atomic.StoreInt32(&requestCount, 0)
		retryReq := httptest.NewRequest(http.MethodGet, "/failing", nil)
		retryRR := httptest.NewRecorder()
		
		handler.ServeHTTP(retryRR, retryReq)
		assert.Equal(t, http.StatusOK, retryRR.Code)
		assert.Contains(t, retryRR.Body.String(), "success")
	})
}

// TestRecoveryMiddleware_ConcurrentAccess tests concurrent access to recovery middleware
func TestRecoveryMiddleware_ConcurrentAccess(t *testing.T) {
	circuitMW := NewCircuitBreakerMiddleware("concurrent_test",
		WithFailureThreshold(5),
	)
	
	const numGoroutines = 20
	const requestsPerGoroutine = 10
	
	var successCount int32
	var failureCount int32
	
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Randomly succeed or fail
		if atomic.LoadInt32(&successCount) < 50 {
			atomic.AddInt32(&successCount, 1)
			w.WriteHeader(http.StatusOK)
		} else {
			atomic.AddInt32(&failureCount, 1)
			http.Error(w, "failure", http.StatusInternalServerError)
		}
	})
	
	handler := circuitMW.HTTPMiddleware()(testHandler)
	
	var wg sync.WaitGroup
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			for j := 0; j < requestsPerGoroutine; j++ {
				req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/test/%d/%d", goroutineID, j), nil)
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)
				// Don't assert specific status codes as circuit breaker state may vary
			}
		}(i)
	}
	
	wg.Wait()
	
	// Verify circuit breaker handled concurrent access without panicking
	t.Logf("Concurrent test completed - Success: %d, Failures: %d", 
		atomic.LoadInt32(&successCount), atomic.LoadInt32(&failureCount))
}

// TestRecoveryMiddleware_Performance tests performance impact of recovery middleware
func TestRecoveryMiddleware_Performance(t *testing.T) {
	panicMW := NewPanicRecoveryMiddleware()
	circuitMW := NewCircuitBreakerMiddleware("perf_test")
	retryMW := NewRetryMiddleware(WithMaxRetries(1))
	
	combined := NewRecoveryMiddleware(panicMW, circuitMW, retryMW)
	
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"performance": "test"}`))
	})
	
	handler := combined.HTTPMiddleware()(testHandler)
	
	// Benchmark the middleware
	start := time.Now()
	iterations := 1000
	
	for i := 0; i < iterations; i++ {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/perf/%d", i), nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}
	
	duration := time.Since(start)
	avgDuration := duration / time.Duration(iterations)
	
	t.Logf("Performance test: %d requests in %v (avg: %v per request)", 
		iterations, duration, avgDuration)
	
	// Verify reasonable performance (should be under 100µs per request for simple operations)
	assert.Less(t, avgDuration.Nanoseconds(), int64(100*time.Microsecond), 
		"recovery middleware should have minimal performance impact")
}