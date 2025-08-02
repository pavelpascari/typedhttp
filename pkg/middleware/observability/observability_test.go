package observability

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types for observability middleware testing
type TestRequest struct {
	UserID string `json:"user_id"`
	Action string `json:"action"`
}

type TestResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

// TestLoggingMiddleware_Configuration tests logging middleware configuration
func TestLoggingMiddleware_Configuration(t *testing.T) {
	var buf bytes.Buffer
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	logger := slog.New(slog.NewJSONHandler(&buf, opts))
	
	t.Run("default_configuration", func(t *testing.T) {
		middleware := NewLoggingMiddleware(logger)
		assert.NotNil(t, middleware)
		
		config := middleware.GetConfig()
		assert.True(t, config.LogRequests)
		assert.True(t, config.LogResponses)
		assert.Equal(t, slog.LevelInfo, config.Level)
		assert.False(t, config.LogRequestBody)
		assert.False(t, config.LogResponseBody)
	})
	
	t.Run("custom_configuration", func(t *testing.T) {
		middleware := NewLoggingMiddleware(logger,
			WithLogLevel(slog.LevelDebug),
			WithRequestBodyLogging(true),
			WithResponseBodyLogging(true),
			WithLogFields(map[string]interface{}{
				"service": "test-api",
				"version": "1.0.0",
			}),
		)
		
		config := middleware.GetConfig()
		assert.Equal(t, slog.LevelDebug, config.Level)
		assert.True(t, config.LogRequestBody)
		assert.True(t, config.LogResponseBody)
		assert.Equal(t, "test-api", config.Fields["service"])
		assert.Equal(t, "1.0.0", config.Fields["version"])
	})
}

// TestLoggingMiddleware_HTTPMiddleware tests logging as HTTP middleware
func TestLoggingMiddleware_HTTPMiddleware(t *testing.T) {
	var buf bytes.Buffer
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	logger := slog.New(slog.NewJSONHandler(&buf, opts))
	middleware := NewLoggingMiddleware(logger, WithLogLevel(slog.LevelDebug))
	
	// Test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success"}`))
	})
	
	handler := middleware.HTTPMiddleware()(testHandler)
	
	t.Run("successful_request_logging", func(t *testing.T) {
		buf.Reset()
		
		req := httptest.NewRequest(http.MethodPost, "/api/test?param=value", strings.NewReader(`{"user_id": "123"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "test-client")
		rr := httptest.NewRecorder()
		
		handler.ServeHTTP(rr, req)
		
		assert.Equal(t, http.StatusOK, rr.Code)
		
		// Verify logs were written
		logOutput := buf.String()
		assert.NotEmpty(t, logOutput)
		
		// Should contain request and response logs
		assert.Contains(t, logOutput, "request_received")
		assert.Contains(t, logOutput, "request_completed")
		assert.Contains(t, logOutput, "POST")
		assert.Contains(t, logOutput, "/api/test")
		assert.Contains(t, logOutput, "200")
	})
	
	t.Run("error_request_logging", func(t *testing.T) {
		buf.Reset()
		
		errorHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "internal error", http.StatusInternalServerError)
		})
		
		handler := middleware.HTTPMiddleware()(errorHandler)
		
		req := httptest.NewRequest(http.MethodGet, "/api/error", nil)
		rr := httptest.NewRecorder()
		
		handler.ServeHTTP(rr, req)
		
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		
		logOutput := buf.String()
		assert.Contains(t, logOutput, "500")
		assert.Contains(t, logOutput, "request_completed")
	})
}

// TestLoggingMiddleware_TypedMiddleware tests logging as typed middleware
func TestLoggingMiddleware_TypedMiddleware(t *testing.T) {
	var buf bytes.Buffer
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	logger := slog.New(slog.NewJSONHandler(&buf, opts))
	middleware := NewLoggingMiddleware(logger, WithRequestBodyLogging(true))
	
	t.Run("successful_typed_middleware", func(t *testing.T) {
		buf.Reset()
		
		req := &TestRequest{
			UserID: "user123",
			Action: "test_action",
		}
		
		ctx := context.Background()
		
		// Before middleware
		newCtx, err := middleware.Before(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, newCtx)
		
		// After middleware
		resp := &TestResponse{
			Message: "success",
			Status:  "ok",
		}
		
		finalResp, err := middleware.After(newCtx, req, resp, nil)
		require.NoError(t, err)
		assert.Equal(t, resp, finalResp)
		
		// Verify logs
		logOutput := buf.String()
		assert.Contains(t, logOutput, "typed_request_received")
		assert.Contains(t, logOutput, "typed_request_completed")
		assert.Contains(t, logOutput, "user123")
		assert.Contains(t, logOutput, "test_action")
	})
	
	t.Run("error_handling", func(t *testing.T) {
		buf.Reset()
		
		req := &TestRequest{UserID: "user456", Action: "error_action"}
		resp := &TestResponse{Message: "error", Status: "failed"}
		
		ctx := context.Background()
		newCtx, _ := middleware.Before(ctx, req)
		
		// Simulate error
		testErr := fmt.Errorf("processing error")
		finalResp, err := middleware.After(newCtx, req, resp, testErr)
		
		require.NoError(t, err) // After middleware shouldn't return error
		assert.Equal(t, resp, finalResp)
		
		logOutput := buf.String()
		assert.Contains(t, logOutput, "processing error")
		assert.Contains(t, logOutput, "typed_request_completed")
	})
}

// TestMetricsMiddleware_Configuration tests metrics middleware configuration
func TestMetricsMiddleware_Configuration(t *testing.T) {
	t.Run("default_configuration", func(t *testing.T) {
		middleware := NewMetricsMiddleware("test_service")
		assert.NotNil(t, middleware)
		
		config := middleware.GetConfig()
		assert.Equal(t, "test_service", config.ServiceName)
		assert.True(t, config.CollectRequestMetrics)
		assert.True(t, config.CollectResponseMetrics)
		assert.False(t, config.CollectCustomMetrics)
	})
	
	t.Run("custom_configuration", func(t *testing.T) {
		middleware := NewMetricsMiddleware("test_service",
			WithCustomMetrics(true),
			WithMetricLabels(map[string]string{
				"version": "1.0.0",
				"env":     "test",
			}),
			WithHistogramBuckets([]float64{0.1, 0.5, 1.0, 5.0}),
		)
		
		config := middleware.GetConfig()
		assert.True(t, config.CollectCustomMetrics)
		assert.Equal(t, "1.0.0", config.Labels["version"])
		assert.Equal(t, "test", config.Labels["env"])
		assert.Equal(t, []float64{0.1, 0.5, 1.0, 5.0}, config.HistogramBuckets)
	})
}

// TestMetricsMiddleware_HTTPMiddleware tests metrics collection in HTTP middleware
func TestMetricsMiddleware_HTTPMiddleware(t *testing.T) {
	middleware := NewMetricsMiddleware("test_api")
	
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate some processing time
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	})
	
	handler := middleware.HTTPMiddleware()(testHandler)
	
	t.Run("request_metrics_collection", func(t *testing.T) {
		// Make multiple requests
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/test/%d", i), nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusOK, rr.Code)
		}
		
		// Verify metrics were collected
		metrics := middleware.GetMetrics()
		assert.NotNil(t, metrics)
		assert.Equal(t, int64(5), metrics.RequestCount)
		assert.Greater(t, metrics.TotalDuration.Nanoseconds(), int64(0))
		assert.Greater(t, len(metrics.ResponseCodes), 0)
		assert.Equal(t, int64(5), metrics.ResponseCodes[200])
	})
	
	t.Run("error_metrics_collection", func(t *testing.T) {
		errorHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not found", http.StatusNotFound)
		})
		
		handler := middleware.HTTPMiddleware()(errorHandler)
		
		req := httptest.NewRequest(http.MethodGet, "/api/notfound", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		
		metrics := middleware.GetMetrics()
		assert.Greater(t, metrics.ResponseCodes[404], int64(0))
	})
}

// TestMetricsMiddleware_TypedMiddleware tests metrics collection in typed middleware
func TestMetricsMiddleware_TypedMiddleware(t *testing.T) {
	middleware := NewMetricsMiddleware("typed_api", WithCustomMetrics(true))
	
	t.Run("typed_metrics_collection", func(t *testing.T) {
		req := &TestRequest{UserID: "metrics_user", Action: "test"}
		resp := &TestResponse{Message: "success", Status: "ok"}
		
		ctx := context.Background()
		
		// Simulate processing
		newCtx, err := middleware.Before(ctx, req)
		require.NoError(t, err)
		
		time.Sleep(5 * time.Millisecond)
		
		finalResp, err := middleware.After(newCtx, req, resp, nil)
		require.NoError(t, err)
		assert.Equal(t, resp, finalResp)
		
		// Verify metrics
		metrics := middleware.GetMetrics()
		assert.Greater(t, metrics.TypedRequestCount, int64(0))
		assert.Greater(t, metrics.TotalTypedDuration.Nanoseconds(), int64(0))
	})
}

// TestTracingMiddleware_Configuration tests tracing middleware configuration
func TestTracingMiddleware_Configuration(t *testing.T) {
	t.Run("default_configuration", func(t *testing.T) {
		middleware := NewTracingMiddleware("test_service")
		assert.NotNil(t, middleware)
		
		config := middleware.GetConfig()
		assert.Equal(t, "test_service", config.ServiceName)
		assert.True(t, config.TraceRequests)
		assert.True(t, config.TraceResponses)
		assert.False(t, config.TraceRequestBody)
		assert.False(t, config.TraceResponseBody)
	})
	
	t.Run("custom_configuration", func(t *testing.T) {
		middleware := NewTracingMiddleware("test_service",
			WithBodyTracing(true),
			WithTraceAttributes(map[string]interface{}{
				"version": "1.0.0",
				"region":  "us-east-1",
			}),
		)
		
		config := middleware.GetConfig()
		assert.True(t, config.TraceRequestBody)
		assert.True(t, config.TraceResponseBody)
		assert.Equal(t, "1.0.0", config.Attributes["version"])
		assert.Equal(t, "us-east-1", config.Attributes["region"])
	})
}

// TestTracingMiddleware_HTTPMiddleware tests tracing in HTTP middleware
func TestTracingMiddleware_HTTPMiddleware(t *testing.T) {
	middleware := NewTracingMiddleware("trace_api")
	
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify trace context is available
		ctx := r.Context()
		span := GetSpanFromContext(ctx)
		assert.NotNil(t, span, "span should be available in context")
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"traced": true}`))
	})
	
	handler := middleware.HTTPMiddleware()(testHandler)
	
	t.Run("trace_creation_and_context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/traced", nil)
		req.Header.Set("X-Trace-Id", "test-trace-123")
		rr := httptest.NewRecorder()
		
		handler.ServeHTTP(rr, req)
		
		assert.Equal(t, http.StatusOK, rr.Code)
		
		// Verify trace headers are set
		traceID := rr.Header().Get("X-Trace-Id")
		assert.NotEmpty(t, traceID)
	})
}

// TestTracingMiddleware_TypedMiddleware tests tracing in typed middleware
func TestTracingMiddleware_TypedMiddleware(t *testing.T) {
	middleware := NewTracingMiddleware("typed_trace", WithBodyTracing(true))
	
	t.Run("typed_trace_creation", func(t *testing.T) {
		req := &TestRequest{UserID: "trace_user", Action: "trace_action"}
		resp := &TestResponse{Message: "traced", Status: "ok"}
		
		ctx := context.Background()
		
		// Before middleware - should create span
		newCtx, err := middleware.Before(ctx, req)
		require.NoError(t, err)
		
		// Verify span is in context
		span := GetSpanFromContext(newCtx)
		assert.NotNil(t, span)
		
		// After middleware - should complete span
		finalResp, err := middleware.After(newCtx, req, resp, nil)
		require.NoError(t, err)
		assert.Equal(t, resp, finalResp)
	})
	
	t.Run("error_tracing", func(t *testing.T) {
		req := &TestRequest{UserID: "error_user", Action: "error_action"}
		resp := &TestResponse{Message: "error", Status: "failed"}
		
		ctx := context.Background()
		newCtx, _ := middleware.Before(ctx, req)
		
		// Simulate error
		testErr := fmt.Errorf("trace error")
		finalResp, err := middleware.After(newCtx, req, resp, testErr)
		
		require.NoError(t, err)
		assert.Equal(t, resp, finalResp)
		
		// Verify error was recorded in span
		span := GetSpanFromContext(newCtx)
		assert.NotNil(t, span)
		assert.True(t, span.HasError(), "span should record error")
	})
}

// TestObservabilityMiddleware_CombinedUsage tests using all observability middleware together
func TestObservabilityMiddleware_CombinedUsage(t *testing.T) {
	var buf bytes.Buffer
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	logger := slog.New(slog.NewJSONHandler(&buf, opts))
	
	// Create all observability middleware
	loggingMW := NewLoggingMiddleware(logger, WithLogLevel(slog.LevelDebug))
	metricsMW := NewMetricsMiddleware("combined_api")
	tracingMW := NewTracingMiddleware("combined_api")
	
	// Combine middleware
	combined := NewObservabilityMiddleware(loggingMW, metricsMW, tracingMW)
	
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify all contexts are available
		span := GetSpanFromContext(r.Context())
		assert.NotNil(t, span, "tracing span should be available")
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"combined": true}`))
	})
	
	handler := combined.HTTPMiddleware()(testHandler)
	
	t.Run("combined_observability", func(t *testing.T) {
		buf.Reset()
		
		req := httptest.NewRequest(http.MethodPost, "/api/combined", strings.NewReader(`{"test": true}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		
		handler.ServeHTTP(rr, req)
		
		assert.Equal(t, http.StatusOK, rr.Code)
		
		// Verify logging occurred
		logOutput := buf.String()
		assert.NotEmpty(t, logOutput)
		assert.Contains(t, logOutput, "request_received")
		
		// Verify metrics were collected
		metrics := metricsMW.GetMetrics()
		assert.Greater(t, metrics.RequestCount, int64(0))
		
		// Verify tracing headers
		traceID := rr.Header().Get("X-Trace-Id")
		assert.NotEmpty(t, traceID)
	})
}

// TestObservabilityMiddleware_Performance tests performance impact of observability
func TestObservabilityMiddleware_Performance(t *testing.T) {
	var buf bytes.Buffer
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	logger := slog.New(slog.NewJSONHandler(&buf, opts))
	
	loggingMW := NewLoggingMiddleware(logger)
	metricsMW := NewMetricsMiddleware("perf_test")
	tracingMW := NewTracingMiddleware("perf_test")
	
	combined := NewObservabilityMiddleware(loggingMW, metricsMW, tracingMW)
	
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"performance": "test"}`))
	})
	
	handler := combined.HTTPMiddleware()(testHandler)
	
	// Benchmark the middleware
	start := time.Now()
	iterations := 100
	
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
	
	// Verify reasonable performance (should be under 1ms per request)
	assert.Less(t, avgDuration.Nanoseconds(), int64(time.Millisecond), 
		"observability middleware should have minimal performance impact")
	
	// Verify all metrics were collected
	metrics := metricsMW.GetMetrics()
	assert.Equal(t, int64(iterations), metrics.RequestCount)
}