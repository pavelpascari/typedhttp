package observability

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// LoggingConfig holds logging middleware configuration
type LoggingConfig struct {
	LogRequests       bool
	LogResponses      bool
	LogRequestBody    bool
	LogResponseBody   bool
	Level             slog.Level
	Fields            map[string]interface{}
}

// LoggingMiddleware provides structured logging for HTTP requests and typed operations
type LoggingMiddleware struct {
	logger *slog.Logger
	config LoggingConfig
}

// LoggingOption configures logging middleware
type LoggingOption func(*LoggingConfig)

// WithLogLevel sets the logging level
func WithLogLevel(level slog.Level) LoggingOption {
	return func(c *LoggingConfig) {
		c.Level = level
	}
}

// WithRequestBodyLogging enables request body logging
func WithRequestBodyLogging(enabled bool) LoggingOption {
	return func(c *LoggingConfig) {
		c.LogRequestBody = enabled
	}
}

// WithResponseBodyLogging enables response body logging
func WithResponseBodyLogging(enabled bool) LoggingOption {
	return func(c *LoggingConfig) {
		c.LogResponseBody = enabled
	}
}

// WithLogFields sets additional fields to include in all log entries
func WithLogFields(fields map[string]interface{}) LoggingOption {
	return func(c *LoggingConfig) {
		c.Fields = fields
	}
}

// NewLoggingMiddleware creates a new logging middleware
func NewLoggingMiddleware(logger *slog.Logger, opts ...LoggingOption) *LoggingMiddleware {
	config := LoggingConfig{
		LogRequests:     true,
		LogResponses:    true,
		LogRequestBody:  false,
		LogResponseBody: false,
		Level:           slog.LevelInfo,
		Fields:          make(map[string]interface{}),
	}

	for _, opt := range opts {
		opt(&config)
	}

	return &LoggingMiddleware{
		logger: logger,
		config: config,
	}
}

// GetConfig returns the logging configuration
func (m *LoggingMiddleware) GetConfig() LoggingConfig {
	return m.config
}

// HTTPMiddleware returns HTTP middleware function
func (m *LoggingMiddleware) HTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Log request
			if m.config.LogRequests {
				fields := []interface{}{
					"event", "request_received",
					"method", r.Method,
					"path", r.URL.Path,
					"query", r.URL.RawQuery,
					"remote_addr", r.RemoteAddr,
					"user_agent", r.UserAgent(),
				}

				// Add custom fields
				for k, v := range m.config.Fields {
					fields = append(fields, k, v)
				}

				m.logger.LogAttrs(context.Background(), m.config.Level, "HTTP request received", m.fieldsToAttrs(fields)...)
			}

			// Wrap response writer to capture status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Process request
			next.ServeHTTP(rw, r)

			// Log response
			if m.config.LogResponses {
				duration := time.Since(start)
				fields := []interface{}{
					"event", "request_completed",
					"method", r.Method,
					"path", r.URL.Path,
					"status_code", rw.statusCode,
					"duration_ms", duration.Milliseconds(),
				}

				// Add custom fields
				for k, v := range m.config.Fields {
					fields = append(fields, k, v)
				}

				m.logger.LogAttrs(context.Background(), m.config.Level, "HTTP request completed", m.fieldsToAttrs(fields)...)
			}
		})
	}
}

// Before implements TypedPreMiddleware interface
func (m *LoggingMiddleware) Before(ctx context.Context, req interface{}) (context.Context, error) {
	if m.config.LogRequests {
		fields := []interface{}{
			"event", "typed_request_received",
			"request_type", fmt.Sprintf("%T", req),
		}

		// Add custom fields
		for k, v := range m.config.Fields {
			fields = append(fields, k, v)
		}

		// Try to extract user ID if it's in the request
		if userReq, ok := req.(interface{ GetUserID() string }); ok {
			fields = append(fields, "user_id", userReq.GetUserID())
		}

		// Log request body if enabled (simplified for demo)
		if m.config.LogRequestBody {
			fields = append(fields, "request", fmt.Sprintf("%+v", req))
		}

		m.logger.LogAttrs(ctx, m.config.Level, "Typed request received", m.fieldsToAttrs(fields)...)
	}

	// Add start time to context for duration calculation
	return context.WithValue(ctx, "start_time", time.Now()), nil
}

// After implements TypedPostMiddleware interface
func (m *LoggingMiddleware) After(ctx context.Context, req interface{}, resp interface{}, err error) (interface{}, error) {
	if m.config.LogResponses {
		startTime, _ := ctx.Value("start_time").(time.Time)
		duration := time.Since(startTime)

		fields := []interface{}{
			"event", "typed_request_completed",
			"request_type", fmt.Sprintf("%T", req),
			"response_type", fmt.Sprintf("%T", resp),
			"duration_ms", duration.Milliseconds(),
		}

		// Add custom fields
		for k, v := range m.config.Fields {
			fields = append(fields, k, v)
		}

		// Log error if present
		if err != nil {
			fields = append(fields, "error", err.Error())
		}

		// Log response body if enabled (simplified for demo)
		if m.config.LogResponseBody {
			fields = append(fields, "response", fmt.Sprintf("%+v", resp))
		}

		m.logger.LogAttrs(ctx, m.config.Level, "Typed request completed", m.fieldsToAttrs(fields)...)
	}

	return resp, nil
}

// fieldsToAttrs converts key-value pairs to slog.Attr slice
func (m *LoggingMiddleware) fieldsToAttrs(fields []interface{}) []slog.Attr {
	attrs := make([]slog.Attr, 0, len(fields)/2)
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			key := fmt.Sprintf("%v", fields[i])
			value := fields[i+1]
			attrs = append(attrs, slog.Any(key, value))
		}
	}
	return attrs
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// MetricsConfig holds metrics middleware configuration
type MetricsConfig struct {
	ServiceName            string
	CollectRequestMetrics  bool
	CollectResponseMetrics bool
	CollectCustomMetrics   bool
	Labels                 map[string]string
	HistogramBuckets       []float64
}

// MetricsData holds collected metrics
type MetricsData struct {
	RequestCount        int64
	TypedRequestCount   int64
	TotalDuration       time.Duration
	TotalTypedDuration  time.Duration
	ResponseCodes       map[int]int64
	mu                  sync.RWMutex
}

// MetricsMiddleware provides metrics collection for HTTP requests and typed operations
type MetricsMiddleware struct {
	config  MetricsConfig
	metrics *MetricsData
}

// MetricsOption configures metrics middleware
type MetricsOption func(*MetricsConfig)

// WithCustomMetrics enables custom metrics collection
func WithCustomMetrics(enabled bool) MetricsOption {
	return func(c *MetricsConfig) {
		c.CollectCustomMetrics = enabled
	}
}

// WithMetricLabels sets additional labels for metrics
func WithMetricLabels(labels map[string]string) MetricsOption {
	return func(c *MetricsConfig) {
		c.Labels = labels
	}
}

// WithHistogramBuckets sets histogram buckets for duration metrics
func WithHistogramBuckets(buckets []float64) MetricsOption {
	return func(c *MetricsConfig) {
		c.HistogramBuckets = buckets
	}
}

// NewMetricsMiddleware creates a new metrics middleware
func NewMetricsMiddleware(serviceName string, opts ...MetricsOption) *MetricsMiddleware {
	config := MetricsConfig{
		ServiceName:            serviceName,
		CollectRequestMetrics:  true,
		CollectResponseMetrics: true,
		CollectCustomMetrics:   false,
		Labels:                 make(map[string]string),
		HistogramBuckets:       []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	}

	for _, opt := range opts {
		opt(&config)
	}

	return &MetricsMiddleware{
		config: config,
		metrics: &MetricsData{
			ResponseCodes: make(map[int]int64),
		},
	}
}

// GetConfig returns the metrics configuration
func (m *MetricsMiddleware) GetConfig() MetricsConfig {
	return m.config
}

// GetMetrics returns current metrics data
func (m *MetricsMiddleware) GetMetrics() *MetricsData {
	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()

	// Return a copy to avoid race conditions
	responseCodesCopy := make(map[int]int64)
	for k, v := range m.metrics.ResponseCodes {
		responseCodesCopy[k] = v
	}

	return &MetricsData{
		RequestCount:       m.metrics.RequestCount,
		TypedRequestCount:  m.metrics.TypedRequestCount,
		TotalDuration:      m.metrics.TotalDuration,
		TotalTypedDuration: m.metrics.TotalTypedDuration,
		ResponseCodes:      responseCodesCopy,
	}
}

// HTTPMiddleware returns HTTP middleware function
func (m *MetricsMiddleware) HTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Process request
			next.ServeHTTP(rw, r)

			// Collect metrics
			duration := time.Since(start)
			m.recordHTTPMetrics(rw.statusCode, duration)
		})
	}
}

// Before implements TypedPreMiddleware interface
func (m *MetricsMiddleware) Before(ctx context.Context, req interface{}) (context.Context, error) {
	// Add start time to context for duration calculation
	return context.WithValue(ctx, "metrics_start_time", time.Now()), nil
}

// After implements TypedPostMiddleware interface
func (m *MetricsMiddleware) After(ctx context.Context, req interface{}, resp interface{}, err error) (interface{}, error) {
	startTime, _ := ctx.Value("metrics_start_time").(time.Time)
	duration := time.Since(startTime)
	m.recordTypedMetrics(duration)

	return resp, nil
}

// recordHTTPMetrics records HTTP request metrics
func (m *MetricsMiddleware) recordHTTPMetrics(statusCode int, duration time.Duration) {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()

	m.metrics.RequestCount++
	m.metrics.TotalDuration += duration
	m.metrics.ResponseCodes[statusCode]++
}

// recordTypedMetrics records typed request metrics
func (m *MetricsMiddleware) recordTypedMetrics(duration time.Duration) {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()

	m.metrics.TypedRequestCount++
	m.metrics.TotalTypedDuration += duration
}

// TracingConfig holds tracing middleware configuration
type TracingConfig struct {
	ServiceName        string
	TraceRequests      bool
	TraceResponses     bool
	TraceRequestBody   bool
	TraceResponseBody  bool
	Attributes         map[string]interface{}
}

// Span represents a simple tracing span
type Span struct {
	TraceID   string
	SpanID    string
	Operation string
	StartTime time.Time
	EndTime   time.Time
	Tags      map[string]interface{}
	Error     error
	mu        sync.Mutex
}

// HasError returns true if the span has an error
func (s *Span) HasError() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Error != nil
}

// SetError sets an error on the span
func (s *Span) SetError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Error = err
}

// SetTag sets a tag on the span
func (s *Span) SetTag(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Tags == nil {
		s.Tags = make(map[string]interface{})
	}
	s.Tags[key] = value
}

// Finish finishes the span
func (s *Span) Finish() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.EndTime = time.Now()
}

// TracingMiddleware provides distributed tracing for HTTP requests and typed operations
type TracingMiddleware struct {
	config TracingConfig
}

// TracingOption configures tracing middleware
type TracingOption func(*TracingConfig)

// WithBodyTracing enables body tracing for requests and responses
func WithBodyTracing(enabled bool) TracingOption {
	return func(c *TracingConfig) {
		c.TraceRequestBody = enabled
		c.TraceResponseBody = enabled
	}
}

// WithTraceAttributes sets additional attributes for traces
func WithTraceAttributes(attributes map[string]interface{}) TracingOption {
	return func(c *TracingConfig) {
		c.Attributes = attributes
	}
}

// NewTracingMiddleware creates a new tracing middleware
func NewTracingMiddleware(serviceName string, opts ...TracingOption) *TracingMiddleware {
	config := TracingConfig{
		ServiceName:       serviceName,
		TraceRequests:     true,
		TraceResponses:    true,
		TraceRequestBody:  false,
		TraceResponseBody: false,
		Attributes:        make(map[string]interface{}),
	}

	for _, opt := range opts {
		opt(&config)
	}

	return &TracingMiddleware{
		config: config,
	}
}

// GetConfig returns the tracing configuration
func (m *TracingMiddleware) GetConfig() TracingConfig {
	return m.config
}

// HTTPMiddleware returns HTTP middleware function
func (m *TracingMiddleware) HTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create or extract trace
			traceID := r.Header.Get("X-Trace-Id")
			if traceID == "" {
				traceID = generateTraceID()
			}

			spanID := generateSpanID()

			// Create span
			span := &Span{
				TraceID:   traceID,
				SpanID:    spanID,
				Operation: fmt.Sprintf("%s %s", r.Method, r.URL.Path),
				StartTime: time.Now(),
				Tags:      make(map[string]interface{}),
			}

			// Add span to context
			ctx := context.WithValue(r.Context(), "span", span)

			// Set trace headers
			w.Header().Set("X-Trace-Id", traceID)
			w.Header().Set("X-Span-Id", spanID)

			// Process request
			next.ServeHTTP(w, r.WithContext(ctx))

			// Finish span
			span.Finish()
		})
	}
}

// Before implements TypedPreMiddleware interface
func (m *TracingMiddleware) Before(ctx context.Context, req interface{}) (context.Context, error) {
	// Create span for typed operation
	span := &Span{
		TraceID:   generateTraceID(),
		SpanID:    generateSpanID(),
		Operation: fmt.Sprintf("typed_operation_%T", req),
		StartTime: time.Now(),
		Tags:      make(map[string]interface{}),
	}

	// Add attributes
	for k, v := range m.config.Attributes {
		span.SetTag(k, v)
	}

	// Add span to context
	return context.WithValue(ctx, "span", span), nil
}

// After implements TypedPostMiddleware interface
func (m *TracingMiddleware) After(ctx context.Context, req interface{}, resp interface{}, err error) (interface{}, error) {
	span := GetSpanFromContext(ctx)
	if span != nil {
		if err != nil {
			span.SetError(err)
		}
		span.Finish()
	}

	return resp, nil
}

// GetSpanFromContext extracts span from context
func GetSpanFromContext(ctx context.Context) *Span {
	span, _ := ctx.Value("span").(*Span)
	return span
}

// ObservabilityMiddleware combines logging, metrics, and tracing
type ObservabilityMiddleware struct {
	logging *LoggingMiddleware
	metrics *MetricsMiddleware
	tracing *TracingMiddleware
}

// NewObservabilityMiddleware creates a combined observability middleware
func NewObservabilityMiddleware(logging *LoggingMiddleware, metrics *MetricsMiddleware, tracing *TracingMiddleware) *ObservabilityMiddleware {
	return &ObservabilityMiddleware{
		logging: logging,
		metrics: metrics,
		tracing: tracing,
	}
}

// HTTPMiddleware returns HTTP middleware function that combines all observability features
func (m *ObservabilityMiddleware) HTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		handler := next

		// Apply middlewares in reverse order (tracing -> metrics -> logging)
		if m.tracing != nil {
			handler = m.tracing.HTTPMiddleware()(handler)
		}
		if m.metrics != nil {
			handler = m.metrics.HTTPMiddleware()(handler)
		}
		if m.logging != nil {
			handler = m.logging.HTTPMiddleware()(handler)
		}

		return handler
	}
}

// generateTraceID generates a simple trace ID (in production, use proper distributed tracing)
func generateTraceID() string {
	return fmt.Sprintf("trace-%d", time.Now().UnixNano())
}

// generateSpanID generates a simple span ID (in production, use proper distributed tracing)
func generateSpanID() string {
	return fmt.Sprintf("span-%d", time.Now().UnixNano())
}