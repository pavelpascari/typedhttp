package recovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"runtime/debug"
	"sync"
	"time"
)

// Common errors
var (
	ErrCircuitBreakerOpen = errors.New("circuit breaker open")
	ErrMaxRetriesExceeded = errors.New("maximum retries exceeded")
)

// Panic Recovery Middleware
type PanicRecoveryConfig struct {
	RecoverPanics     bool
	LogPanics         bool
	IncludeStackTrace bool
	StatusCode        int
	PanicHandler      func(interface{}) error
}

type PanicRecoveryMiddleware struct {
	config PanicRecoveryConfig
}

type PanicRecoveryOption func(*PanicRecoveryConfig)

// WithPanicRecovery enables or disables panic recovery
func WithPanicRecovery(enabled bool) PanicRecoveryOption {
	return func(c *PanicRecoveryConfig) {
		c.RecoverPanics = enabled
	}
}

// WithPanicLogging enables or disables panic logging
func WithPanicLogging(enabled bool) PanicRecoveryOption {
	return func(c *PanicRecoveryConfig) {
		c.LogPanics = enabled
	}
}

// WithStackTrace enables or disables stack trace inclusion
func WithStackTrace(enabled bool) PanicRecoveryOption {
	return func(c *PanicRecoveryConfig) {
		c.IncludeStackTrace = enabled
	}
}

// WithRecoveryStatusCode sets the HTTP status code for recovered panics
func WithRecoveryStatusCode(statusCode int) PanicRecoveryOption {
	return func(c *PanicRecoveryConfig) {
		c.StatusCode = statusCode
	}
}

// WithPanicHandler sets a custom panic handler
func WithPanicHandler(handler func(interface{}) error) PanicRecoveryOption {
	return func(c *PanicRecoveryConfig) {
		c.PanicHandler = handler
	}
}

// NewPanicRecoveryMiddleware creates a new panic recovery middleware
func NewPanicRecoveryMiddleware(opts ...PanicRecoveryOption) *PanicRecoveryMiddleware {
	config := PanicRecoveryConfig{
		RecoverPanics:     true,
		LogPanics:         true,
		IncludeStackTrace: true,
		StatusCode:        http.StatusInternalServerError,
		PanicHandler:      nil,
	}

	for _, opt := range opts {
		opt(&config)
	}

	return &PanicRecoveryMiddleware{
		config: config,
	}
}

// GetConfig returns the panic recovery configuration
func (m *PanicRecoveryMiddleware) GetConfig() PanicRecoveryConfig {
	return m.config
}

// HTTPMiddleware returns HTTP middleware function
func (m *PanicRecoveryMiddleware) HTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if m.config.RecoverPanics {
				defer func() {
					if panicValue := recover(); panicValue != nil {
						m.handlePanic(w, r, panicValue)
					}
				}()
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Before implements TypedPreMiddleware interface
func (m *PanicRecoveryMiddleware) Before(ctx context.Context, req interface{}) (context.Context, error) {
	return ctx, nil
}

// After implements TypedPostMiddleware interface
func (m *PanicRecoveryMiddleware) After(ctx context.Context, req interface{}, resp interface{}, err error) (interface{}, error) {
	if err != nil && m.config.LogPanics {
		log.Printf("Recovery middleware: processed error: %v", err)
	}
	return resp, err
}

// handlePanic handles a recovered panic
func (m *PanicRecoveryMiddleware) handlePanic(w http.ResponseWriter, r *http.Request, panicValue interface{}) {
	var err error

	// Use custom panic handler if provided
	if m.config.PanicHandler != nil {
		err = m.config.PanicHandler(panicValue)
	} else {
		err = fmt.Errorf("panic recovered: %v", panicValue)
	}

	// Log the panic if enabled
	if m.config.LogPanics {
		logMessage := fmt.Sprintf("Panic recovered: %v", err)
		if m.config.IncludeStackTrace {
			logMessage += fmt.Sprintf("\nStack trace:\n%s", debug.Stack())
		}
		log.Printf(logMessage)
	}

	// Write error response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(m.config.StatusCode)

	// Use custom error message if custom handler provided, otherwise default message
	errorMessage := "internal server error"
	if m.config.PanicHandler != nil && err != nil {
		errorMessage = err.Error()
	}

	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": errorMessage,
	})
}

// Circuit Breaker Middleware
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateOpen
	StateHalfOpen
)

type CircuitBreakerCounts struct {
	Requests             uint32
	TotalSuccesses       uint32
	TotalFailures        uint32
	ConsecutiveSuccesses uint32
	ConsecutiveFailures  uint32
}

type CircuitBreakerConfig struct {
	ServiceName      string
	FailureThreshold int
	RecoveryTimeout  time.Duration
	MaxRequests      int
	ReadyToTrip      func(CircuitBreakerCounts) bool
}

type CircuitBreakerMiddleware struct {
	config CircuitBreakerConfig
	state  CircuitBreakerState
	counts CircuitBreakerCounts
	expiry time.Time
	mu     sync.RWMutex
}

type CircuitBreakerOption func(*CircuitBreakerConfig)

// WithFailureThreshold sets the failure threshold
func WithFailureThreshold(threshold int) CircuitBreakerOption {
	return func(c *CircuitBreakerConfig) {
		c.FailureThreshold = threshold
	}
}

// WithRecoveryTimeout sets the recovery timeout
func WithRecoveryTimeout(timeout time.Duration) CircuitBreakerOption {
	return func(c *CircuitBreakerConfig) {
		c.RecoveryTimeout = timeout
	}
}

// WithMaxRequests sets the maximum requests in half-open state
func WithMaxRequests(maxRequests int) CircuitBreakerOption {
	return func(c *CircuitBreakerConfig) {
		c.MaxRequests = maxRequests
	}
}

// WithReadyToTripFunc sets a custom function to determine when to trip
func WithReadyToTripFunc(fn func(CircuitBreakerCounts) bool) CircuitBreakerOption {
	return func(c *CircuitBreakerConfig) {
		c.ReadyToTrip = fn
	}
}

// NewCircuitBreakerMiddleware creates a new circuit breaker middleware
func NewCircuitBreakerMiddleware(serviceName string, opts ...CircuitBreakerOption) *CircuitBreakerMiddleware {
	config := CircuitBreakerConfig{
		ServiceName:      serviceName,
		FailureThreshold: 5,
		RecoveryTimeout:  60 * time.Second,
		MaxRequests:      3,
		ReadyToTrip:      nil,
	}

	for _, opt := range opts {
		opt(&config)
	}

	// Default ready to trip function
	if config.ReadyToTrip == nil {
		config.ReadyToTrip = func(counts CircuitBreakerCounts) bool {
			return counts.ConsecutiveFailures >= uint32(config.FailureThreshold)
		}
	}

	return &CircuitBreakerMiddleware{
		config: config,
		state:  StateClosed,
	}
}

// GetConfig returns the circuit breaker configuration
func (cb *CircuitBreakerMiddleware) GetConfig() CircuitBreakerConfig {
	return cb.config
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreakerMiddleware) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Allow checks if a request should be allowed
func (cb *CircuitBreakerMiddleware) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if now.After(cb.expiry) {
			cb.state = StateHalfOpen
			cb.counts = CircuitBreakerCounts{}
			return true
		}
		return false
	case StateHalfOpen:
		return cb.counts.Requests < uint32(cb.config.MaxRequests)
	}

	return false
}

// RecordSuccess records a successful request
func (cb *CircuitBreakerMiddleware) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.counts.Requests++
	cb.counts.TotalSuccesses++
	cb.counts.ConsecutiveSuccesses++
	cb.counts.ConsecutiveFailures = 0

	if cb.state == StateHalfOpen && cb.counts.ConsecutiveSuccesses >= uint32(cb.config.MaxRequests) {
		cb.state = StateClosed
		cb.counts = CircuitBreakerCounts{}
	}
}

// RecordFailure records a failed request
func (cb *CircuitBreakerMiddleware) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.counts.Requests++
	cb.counts.TotalFailures++
	cb.counts.ConsecutiveFailures++
	cb.counts.ConsecutiveSuccesses = 0

	if cb.config.ReadyToTrip(cb.counts) {
		cb.state = StateOpen
		cb.expiry = time.Now().Add(cb.config.RecoveryTimeout)
	}
}

// HTTPMiddleware returns HTTP middleware function
func (cb *CircuitBreakerMiddleware) HTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cb.Allow() {
				cb.writeCircuitBreakerError(w)
				return
			}

			// Use a custom response writer to capture the status code
			rw := &circuitBreakerResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)

			// Record success or failure based on status code
			if rw.statusCode >= 200 && rw.statusCode < 400 {
				cb.RecordSuccess()
			} else {
				cb.RecordFailure()
			}
		})
	}
}

// writeCircuitBreakerError writes a circuit breaker error response
func (cb *CircuitBreakerMiddleware) writeCircuitBreakerError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusServiceUnavailable)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": "circuit breaker open",
	})
}

// circuitBreakerResponseWriter wraps http.ResponseWriter to capture status code
type circuitBreakerResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *circuitBreakerResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Retry Middleware
type RetryConfig struct {
	MaxRetries        int
	InitialDelay      time.Duration
	BackoffMultiplier float64
	MaxDelay          time.Duration
	RetryableErrors   []error
	RetryCondition    func(error) bool
}

type RetryMiddleware struct {
	config RetryConfig
}

type RetryOption func(*RetryConfig)

// WithMaxRetries sets the maximum number of retries
func WithMaxRetries(maxRetries int) RetryOption {
	return func(c *RetryConfig) {
		c.MaxRetries = maxRetries
	}
}

// WithInitialDelay sets the initial delay between retries
func WithInitialDelay(delay time.Duration) RetryOption {
	return func(c *RetryConfig) {
		c.InitialDelay = delay
	}
}

// WithBackoffMultiplier sets the backoff multiplier
func WithBackoffMultiplier(multiplier float64) RetryOption {
	return func(c *RetryConfig) {
		c.BackoffMultiplier = multiplier
	}
}

// WithMaxDelay sets the maximum delay between retries
func WithMaxDelay(maxDelay time.Duration) RetryOption {
	return func(c *RetryConfig) {
		c.MaxDelay = maxDelay
	}
}

// WithRetryableErrors sets specific errors that should trigger retries
func WithRetryableErrors(errors []error) RetryOption {
	return func(c *RetryConfig) {
		c.RetryableErrors = errors
	}
}

// WithRetryCondition sets a custom retry condition function
func WithRetryCondition(condition func(error) bool) RetryOption {
	return func(c *RetryConfig) {
		c.RetryCondition = condition
	}
}

// NewRetryMiddleware creates a new retry middleware
func NewRetryMiddleware(opts ...RetryOption) *RetryMiddleware {
	config := RetryConfig{
		MaxRetries:        3,
		InitialDelay:      100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		MaxDelay:          5 * time.Second,
		RetryableErrors:   []error{},
		RetryCondition:    nil,
	}

	for _, opt := range opts {
		opt(&config)
	}

	// Default retry condition
	if config.RetryCondition == nil {
		config.RetryCondition = func(err error) bool {
			// Retry on specific error types or status codes
			return true // Simplified - in practice, this would be more sophisticated
		}
	}

	return &RetryMiddleware{
		config: config,
	}
}

// GetConfig returns the retry configuration
func (m *RetryMiddleware) GetConfig() RetryConfig {
	return m.config
}

// HTTPMiddleware returns HTTP middleware function
func (m *RetryMiddleware) HTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var lastErr error

			for attempt := 0; attempt <= m.config.MaxRetries; attempt++ {
				if attempt > 0 {
					// Wait before retry
					delay := m.calculateDelay(attempt - 1)
					time.Sleep(delay)
				}

				// Create a response recorder to capture the response
				rr := &retryResponseRecorder{
					ResponseWriter: w,
					statusCode:     http.StatusOK,
					body:           make([]byte, 0),
				}

				next.ServeHTTP(rr, r)

				// Check if the response indicates success
				if rr.statusCode >= 200 && rr.statusCode < 400 {
					// Success - write the response and return
					for key, values := range rr.Header() {
						for _, value := range values {
							w.Header().Add(key, value)
						}
					}
					w.WriteHeader(rr.statusCode)
					w.Write(rr.body)
					return
				}

				// Failure - check if we should retry
				lastErr = fmt.Errorf("HTTP %d", rr.statusCode)
				if !m.shouldRetry(lastErr) || attempt == m.config.MaxRetries {
					// Don't retry or max retries reached - write the response
					for key, values := range rr.Header() {
						for _, value := range values {
							w.Header().Add(key, value)
						}
					}
					w.WriteHeader(rr.statusCode)
					w.Write(rr.body)
					return
				}
			}
		})
	}
}

// ExecuteWithRetry executes a function with retry logic
func (m *RetryMiddleware) ExecuteWithRetry(ctx context.Context, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= m.config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := m.calculateDelay(attempt - 1)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err
		if !m.shouldRetry(err) || attempt == m.config.MaxRetries {
			break
		}
	}

	return lastErr
}

// calculateDelay calculates the delay for a given attempt
func (m *RetryMiddleware) calculateDelay(attempt int) time.Duration {
	delay := float64(m.config.InitialDelay) * math.Pow(m.config.BackoffMultiplier, float64(attempt))
	if delay > float64(m.config.MaxDelay) {
		delay = float64(m.config.MaxDelay)
	}
	return time.Duration(delay)
}

// shouldRetry determines if an error should trigger a retry
func (m *RetryMiddleware) shouldRetry(err error) bool {
	if m.config.RetryCondition != nil {
		return m.config.RetryCondition(err)
	}

	// Check against specific retryable errors
	for _, retryableErr := range m.config.RetryableErrors {
		if errors.Is(err, retryableErr) {
			return true
		}
	}

	return true // Default to retrying
}

// retryResponseRecorder captures HTTP responses for retry logic
type retryResponseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       []byte
	written    bool
}

func (rr *retryResponseRecorder) WriteHeader(code int) {
	if !rr.written {
		rr.statusCode = code
		rr.written = true
	}
}

func (rr *retryResponseRecorder) Write(data []byte) (int, error) {
	rr.body = append(rr.body, data...)
	return len(data), nil
}

// Combined Recovery Middleware
type RecoveryMiddleware struct {
	panicRecovery  *PanicRecoveryMiddleware
	circuitBreaker *CircuitBreakerMiddleware
	retry          *RetryMiddleware
}

// NewRecoveryMiddleware creates a combined recovery middleware
func NewRecoveryMiddleware(panicRecovery *PanicRecoveryMiddleware, circuitBreaker *CircuitBreakerMiddleware, retry *RetryMiddleware) *RecoveryMiddleware {
	return &RecoveryMiddleware{
		panicRecovery:  panicRecovery,
		circuitBreaker: circuitBreaker,
		retry:          retry,
	}
}

// HTTPMiddleware returns HTTP middleware function that combines all recovery features
func (m *RecoveryMiddleware) HTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		handler := next

		// Apply middlewares in order: retry -> circuit breaker -> panic recovery
		if m.retry != nil {
			handler = m.retry.HTTPMiddleware()(handler)
		}
		if m.circuitBreaker != nil {
			handler = m.circuitBreaker.HTTPMiddleware()(handler)
		}
		if m.panicRecovery != nil {
			handler = m.panicRecovery.HTTPMiddleware()(handler)
		}

		return handler
	}
}
