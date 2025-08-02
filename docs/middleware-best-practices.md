# Middleware Composition Best Practices

This guide provides comprehensive best practices for composing middleware in TypedHTTP applications, covering everything from simple single-purpose middleware to complex enterprise-grade middleware stacks.

## Table of Contents

1. [Fundamental Principles](#fundamental-principles)
2. [Middleware Types & When to Use](#middleware-types--when-to-use)
3. [Composition Patterns](#composition-patterns)
4. [Error Handling Strategies](#error-handling-strategies)
5. [Performance Optimization](#performance-optimization)
6. [Testing Strategies](#testing-strategies)
7. [Common Anti-patterns](#common-anti-patterns)

## Fundamental Principles

### 1. Single Responsibility Principle

Each middleware should have one clear responsibility:

```go
// ✅ Good - Single responsibility
type RequestIDMiddleware struct{}

func (m *RequestIDMiddleware) Before(ctx context.Context, req *interface{}) (context.Context, error) {
    requestID := generateRequestID()
    return context.WithValue(ctx, "request_id", requestID), nil
}

// ❌ Bad - Multiple responsibilities
type RequestProcessingMiddleware struct{}

func (m *RequestProcessingMiddleware) Before(ctx context.Context, req *interface{}) (context.Context, error) {
    // Adding request ID
    requestID := generateRequestID()
    ctx = context.WithValue(ctx, "request_id", requestID)
    
    // Logging
    log.Printf("Request started: %s", requestID)
    
    // Authentication
    if !isAuthenticated(req) {
        return ctx, errors.New("authentication required")
    }
    
    // Rate limiting
    if !checkRateLimit(req) {
        return ctx, errors.New("rate limit exceeded")
    }
    
    return ctx, nil
}
```

### 2. Immutability and Side-Effect Management

Keep middleware pure and predictable:

```go
// ✅ Good - Immutable operations
type HeaderMiddleware struct {
    headers map[string]string
}

func (m *HeaderMiddleware) Before(ctx context.Context, req *interface{}) (context.Context, error) {
    // Add headers without modifying the original request
    enrichedCtx := context.WithValue(ctx, "additional_headers", m.headers)
    return enrichedCtx, nil
}

// ❌ Bad - Mutating global state
var globalRequestCount int

type CounterMiddleware struct{}

func (m *CounterMiddleware) Before(ctx context.Context, req *interface{}) (context.Context, error) {
    // Dangerous: mutating global state
    globalRequestCount++
    return ctx, nil
}
```

### 3. Explicit Dependencies

Make middleware dependencies clear and testable:

```go
// ✅ Good - Explicit dependencies
type AuthMiddleware struct {
    tokenValidator TokenValidator
    userService    UserService
    logger         Logger
}

func NewAuthMiddleware(validator TokenValidator, userService UserService, logger Logger) *AuthMiddleware {
    return &AuthMiddleware{
        tokenValidator: validator,
        userService:    userService,
        logger:         logger,
    }
}

// ❌ Bad - Hidden dependencies
type AuthMiddleware struct{}

func (m *AuthMiddleware) Before(ctx context.Context, req *interface{}) (context.Context, error) {
    // Hidden dependencies on global variables
    user, err := globalUserService.ValidateToken(globalTokenValidator.Extract(req))
    if err != nil {
        globalLogger.Error("Auth failed")
        return ctx, err
    }
    return context.WithValue(ctx, "user", user), nil
}
```

## Middleware Types & When to Use

### 1. HTTP Transport Middleware

**When to use**: Cross-cutting concerns that need to operate at the HTTP level

```go
// CORS, security headers, compression
func CORSMiddleware(config CORSConfig) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if config.AllowOrigin != "" {
                w.Header().Set("Access-Control-Allow-Origin", config.AllowOrigin)
            }
            
            if r.Method == "OPTIONS" {
                w.WriteHeader(http.StatusNoContent)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}
```

### 2. Typed Pre-Middleware

**When to use**: Request validation, enrichment, or authentication

```go
// Request validation
type ValidationMiddleware[TRequest any] struct {
    validator *validator.Validate
}

func (m *ValidationMiddleware[TRequest]) Before(ctx context.Context, req *TRequest) (context.Context, error) {
    if err := m.validator.Struct(req); err != nil {
        return ctx, &ValidationError{
            Message: "Request validation failed",
            Details: extractValidationDetails(err),
        }
    }
    return ctx, nil
}

// Request enrichment
type EnrichmentMiddleware[TRequest any] struct {
    enricher DataEnricher
}

func (m *EnrichmentMiddleware[TRequest]) Before(ctx context.Context, req *TRequest) (context.Context, error) {
    enrichedData, err := m.enricher.Enrich(ctx, req)
    if err != nil {
        return ctx, err
    }
    return context.WithValue(ctx, "enriched_data", enrichedData), nil
}
```

### 3. Typed Post-Middleware

**When to use**: Response transformation, caching, or formatting

```go
// Response caching
type CacheMiddleware[TResponse any] struct {
    cache Cache
    ttl   time.Duration
}

func (m *CacheMiddleware[TResponse]) After(ctx context.Context, resp *TResponse) (*CachedResponse[TResponse], error) {
    cacheKey := generateCacheKey(ctx)
    
    // Store in cache
    go m.cache.Set(cacheKey, *resp, m.ttl)
    
    return &CachedResponse[TResponse]{
        Data:      *resp,
        CachedAt:  time.Now(),
        ExpiresAt: time.Now().Add(m.ttl),
        CacheHit:  false,
    }, nil
}

// Response formatting
type FormattingMiddleware[TResponse any] struct {
    formatter ResponseFormatter
}

func (m *FormattingMiddleware[TResponse]) After(ctx context.Context, resp *TResponse) (*FormattedResponse[TResponse], error) {
    formatted, err := m.formatter.Format(*resp)
    if err != nil {
        return nil, err
    }
    
    return &FormattedResponse[TResponse]{
        Data:      formatted,
        Format:    m.formatter.GetFormat(),
        Timestamp: time.Now(),
    }, nil
}
```

### 4. Full Lifecycle Middleware

**When to use**: Auditing, metrics collection, or complex request/response correlation

```go
// Comprehensive audit middleware
type AuditMiddleware[TRequest, TResponse any] struct {
    auditService AuditService
    sensitiveFields []string
}

func (m *AuditMiddleware[TRequest, TResponse]) Before(ctx context.Context, req *TRequest) (context.Context, error) {
    auditCtx := &AuditContext{
        RequestID:   getRequestID(ctx),
        UserID:      getUserID(ctx),
        StartTime:   time.Now(),
        RequestType: reflect.TypeOf(*req).Name(),
    }
    
    // Log sanitized request
    sanitizedReq := m.sanitizeRequest(*req)
    m.auditService.LogRequestStart(auditCtx, sanitizedReq)
    
    return context.WithValue(ctx, "audit_context", auditCtx), nil
}

func (m *AuditMiddleware[TRequest, TResponse]) After(ctx context.Context, req *TRequest, resp *TResponse, err error) (*TResponse, error) {
    auditCtx, _ := ctx.Value("audit_context").(*AuditContext)
    if auditCtx != nil {
        auditCtx.EndTime = time.Now()
        auditCtx.Duration = auditCtx.EndTime.Sub(auditCtx.StartTime)
        auditCtx.Success = (err == nil)
        
        if err != nil {
            m.auditService.LogRequestError(auditCtx, err)
        } else {
            sanitizedResp := m.sanitizeResponse(*resp)
            m.auditService.LogRequestSuccess(auditCtx, sanitizedResp)
        }
    }
    
    return resp, err
}
```

## Composition Patterns

### 1. Layer-Based Composition

Organize middleware into logical layers with clear priorities:

```go
type MiddlewareLayer int

const (
    SecurityLayer MiddlewareLayer = iota
    AuthenticationLayer
    ValidationLayer
    BusinessLogicLayer
    ResponseFormattingLayer
    ObservabilityLayer
)

func CreateLayeredMiddleware() []typedhttp.MiddlewareEntry {
    return []typedhttp.MiddlewareEntry{
        // Security Layer (100-90)
        {
            Middleware: &SecurityHeadersMiddleware{},
            Config: typedhttp.MiddlewareConfig{
                Name:     "security_headers",
                Priority: 100,
                Metadata: map[string]interface{}{"layer": SecurityLayer},
            },
        },
        {
            Middleware: &RateLimitMiddleware{Limit: 100},
            Config: typedhttp.MiddlewareConfig{
                Name:     "rate_limit",
                Priority: 95,
                Metadata: map[string]interface{}{"layer": SecurityLayer},
            },
        },
        
        // Authentication Layer (89-80)
        {
            Middleware: &JWTMiddleware{},
            Config: typedhttp.MiddlewareConfig{
                Name:     "jwt_auth",
                Priority: 85,
                Metadata: map[string]interface{}{"layer": AuthenticationLayer},
            },
        },
        
        // Validation Layer (79-70)
        {
            Middleware: &ValidationMiddleware[any]{},
            Config: typedhttp.MiddlewareConfig{
                Name:     "validation",
                Priority: 75,
                Metadata: map[string]interface{}{"layer": ValidationLayer},
            },
        },
        
        // Response Formatting Layer (59-50)
        {
            Middleware: typedhttp.NewResponseEnvelopeMiddleware[any](),
            Config: typedhttp.MiddlewareConfig{
                Name:     "envelope",
                Priority: 55,
                Metadata: map[string]interface{}{"layer": ResponseFormattingLayer},
            },
        },
        
        // Observability Layer (49-40)
        {
            Middleware: &MetricsMiddleware{},
            Config: typedhttp.MiddlewareConfig{
                Name:     "metrics",
                Priority: 45,
                Metadata: map[string]interface{}{"layer": ObservabilityLayer},
            },
        },
        {
            Middleware: &AuditMiddleware[any, any]{},
            Config: typedhttp.MiddlewareConfig{
                Name:     "audit",
                Priority: 40,
                Metadata: map[string]interface{}{"layer": ObservabilityLayer},
            },
        },
    }
}
```

### 2. Conditional Composition

Apply middleware based on runtime conditions:

```go
type ConditionalMiddleware struct {
    condition func(*http.Request) bool
    middleware interface{}
}

func NewConditionalMiddleware(condition func(*http.Request) bool, middleware interface{}) *ConditionalMiddleware {
    return &ConditionalMiddleware{
        condition: condition,
        middleware: middleware,
    }
}

// Usage examples
func CreateConditionalStack() []typedhttp.MiddlewareEntry {
    return []typedhttp.MiddlewareEntry{
        // Always applied
        {
            Middleware: &RequestIDMiddleware{},
            Config: typedhttp.MiddlewareConfig{Priority: 100},
        },
        
        // Only for authenticated endpoints
        {
            Middleware: NewConditionalMiddleware(
                func(r *http.Request) bool {
                    return !strings.HasPrefix(r.URL.Path, "/public/")
                },
                &AuthenticationMiddleware{},
            ),
            Config: typedhttp.MiddlewareConfig{Priority: 90},
        },
        
        // Only for admin endpoints
        {
            Middleware: NewConditionalMiddleware(
                func(r *http.Request) bool {
                    return strings.HasPrefix(r.URL.Path, "/admin/")
                },
                &AdminAuditMiddleware[any, any]{},
            ),
            Config: typedhttp.MiddlewareConfig{Priority: 80},
        },
        
        // Only during business hours
        {
            Middleware: NewConditionalMiddleware(
                func(r *http.Request) bool {
                    hour := time.Now().Hour()
                    return hour >= 9 && hour <= 17
                },
                &BusinessHoursMiddleware{},
            ),
            Config: typedhttp.MiddlewareConfig{Priority: 70},
        },
    }
}
```

### 3. Environment-Based Composition

Different middleware stacks for different environments:

```go
type Environment string

const (
    Development Environment = "development"
    Staging     Environment = "staging"
    Production  Environment = "production"
)

func CreateEnvironmentMiddleware(env Environment) []typedhttp.MiddlewareEntry {
    base := []typedhttp.MiddlewareEntry{
        {Middleware: &RequestIDMiddleware{}, Config: typedhttp.MiddlewareConfig{Priority: 100}},
        {Middleware: &ValidationMiddleware[any]{}, Config: typedhttp.MiddlewareConfig{Priority: 90}},
    }
    
    switch env {
    case Development:
        return append(base, []typedhttp.MiddlewareEntry{
            {Middleware: &DebugMiddleware{}, Config: typedhttp.MiddlewareConfig{Priority: 80}},
            {Middleware: &VerboseLoggingMiddleware{}, Config: typedhttp.MiddlewareConfig{Priority: 70}},
        }...)
        
    case Staging:
        return append(base, []typedhttp.MiddlewareEntry{
            {Middleware: &RateLimitMiddleware{Limit: 1000}, Config: typedhttp.MiddlewareConfig{Priority: 80}},
            {Middleware: &AuditMiddleware[any, any]{}, Config: typedhttp.MiddlewareConfig{Priority: 70}},
        }...)
        
    case Production:
        return append(base, []typedhttp.MiddlewareEntry{
            {Middleware: &SecurityHeadersMiddleware{}, Config: typedhttp.MiddlewareConfig{Priority: 85}},
            {Middleware: &RateLimitMiddleware{Limit: 100}, Config: typedhttp.MiddlewareConfig{Priority: 80}},
            {Middleware: &MetricsMiddleware{}, Config: typedhttp.MiddlewareConfig{Priority: 75}},
            {Middleware: &AuditMiddleware[any, any]{}, Config: typedhttp.MiddlewareConfig{Priority: 70}},
        }...)
        
    default:
        return base
    }
}
```

## Error Handling Strategies

### 1. Standardized Error Types

Define a clear error hierarchy:

```go
type MiddlewareError struct {
    Type       ErrorType         `json:"type"`
    Code       string           `json:"code"`
    Message    string           `json:"message"`
    Details    map[string]any   `json:"details,omitempty"`
    StatusCode int              `json:"-"`
    Cause      error            `json:"-"`
}

type ErrorType string

const (
    ValidationError    ErrorType = "validation_error"
    AuthenticationError ErrorType = "authentication_error"
    AuthorizationError ErrorType = "authorization_error"
    RateLimitError     ErrorType = "rate_limit_error"
    InternalError      ErrorType = "internal_error"
)

func (e *MiddlewareError) Error() string {
    return e.Message
}

func (e *MiddlewareError) Unwrap() error {
    return e.Cause
}

// Factory functions for common errors
func NewValidationError(message string, details map[string]any) *MiddlewareError {
    return &MiddlewareError{
        Type:       ValidationError,
        Code:       "VALIDATION_FAILED",
        Message:    message,
        Details:    details,
        StatusCode: http.StatusBadRequest,
    }
}

func NewAuthenticationError(message string) *MiddlewareError {
    return &MiddlewareError{
        Type:       AuthenticationError,
        Code:       "AUTHENTICATION_REQUIRED",
        Message:    message,
        StatusCode: http.StatusUnauthorized,
    }
}
```

### 2. Error Recovery and Fallbacks

Implement graceful degradation:

```go
type ResilientMiddleware[TRequest any] struct {
    primary   TypedPreMiddleware[TRequest]
    fallback  TypedPreMiddleware[TRequest]
    logger    Logger
}

func (m *ResilientMiddleware[TRequest]) Before(ctx context.Context, req *TRequest) (context.Context, error) {
    // Try primary middleware
    newCtx, err := m.primary.Before(ctx, req)
    if err == nil {
        return newCtx, nil
    }
    
    // Log error and try fallback
    m.logger.Warn("Primary middleware failed, trying fallback", "error", err)
    
    if m.fallback != nil {
        fallbackCtx, fallbackErr := m.fallback.Before(ctx, req)
        if fallbackErr == nil {
            // Add metadata indicating fallback was used
            return context.WithValue(fallbackCtx, "fallback_used", true), nil
        }
        m.logger.Error("Fallback middleware also failed", "fallback_error", fallbackErr)
    }
    
    return ctx, err
}
```

### 3. Circuit Breaker Pattern

Prevent cascading failures:

```go
type CircuitBreakerMiddleware[TRequest any] struct {
    breaker       *CircuitBreaker
    fallbackValue interface{}
}

func (m *CircuitBreakerMiddleware[TRequest]) Before(ctx context.Context, req *TRequest) (context.Context, error) {
    return m.breaker.Execute(func() (context.Context, error) {
        // Execute the actual middleware logic
        return m.executeLogic(ctx, req)
    })
}

type CircuitBreaker struct {
    state         CircuitState
    failureCount  int
    successCount  int
    threshold     int
    timeout       time.Duration
    lastFailTime  time.Time
    mu           sync.RWMutex
}

type CircuitState int

const (
    Closed CircuitState = iota
    Open
    HalfOpen
)
```

## Performance Optimization

### 1. Middleware Pooling

Reuse middleware instances for high-traffic scenarios:

```go
type MiddlewarePool[T any] struct {
    pool sync.Pool
    factory func() T
}

func NewMiddlewarePool[T any](factory func() T) *MiddlewarePool[T] {
    return &MiddlewarePool[T]{
        pool: sync.Pool{
            New: func() interface{} {
                return factory()
            },
        },
        factory: factory,
    }
}

func (p *MiddlewarePool[T]) Get() T {
    return p.pool.Get().(T)
}

func (p *MiddlewarePool[T]) Put(middleware T) {
    // Reset middleware state if needed
    p.pool.Put(middleware)
}

// Usage
var validationPool = NewMiddlewarePool(func() *ValidationMiddleware[any] {
    return &ValidationMiddleware[any]{
        validator: validator.New(),
    }
})

func GetValidationMiddleware() *ValidationMiddleware[any] {
    return validationPool.Get()
}
```

### 2. Lazy Initialization

Defer expensive operations until needed:

```go
type LazyMiddleware struct {
    once     sync.Once
    initFunc func() (interface{}, error)
    instance interface{}
    err      error
}

func NewLazyMiddleware(initFunc func() (interface{}, error)) *LazyMiddleware {
    return &LazyMiddleware{
        initFunc: initFunc,
    }
}

func (m *LazyMiddleware) getInstance() (interface{}, error) {
    m.once.Do(func() {
        m.instance, m.err = m.initFunc()
    })
    return m.instance, m.err
}

func (m *LazyMiddleware) Before(ctx context.Context, req *interface{}) (context.Context, error) {
    instance, err := m.getInstance()
    if err != nil {
        return ctx, err
    }
    
    if middleware, ok := instance.(TypedPreMiddleware[interface{}]); ok {
        return middleware.Before(ctx, req)
    }
    
    return ctx, errors.New("invalid middleware type")
}
```

### 3. Caching Strategies

Cache expensive computations:

```go
type CachedMiddleware[TRequest any] struct {
    cache      Cache
    ttl        time.Duration
    keyFunc    func(*TRequest) string
    middleware TypedPreMiddleware[TRequest]
}

func (m *CachedMiddleware[TRequest]) Before(ctx context.Context, req *TRequest) (context.Context, error) {
    cacheKey := m.keyFunc(req)
    
    // Check cache first
    if cached, found := m.cache.Get(cacheKey); found {
        if result, ok := cached.(CachedResult); ok {
            if result.Error != nil {
                return ctx, result.Error
            }
            return result.Context, nil
        }
    }
    
    // Execute middleware
    newCtx, err := m.middleware.Before(ctx, req)
    
    // Cache result
    result := CachedResult{
        Context: newCtx,
        Error:   err,
    }
    m.cache.Set(cacheKey, result, m.ttl)
    
    return newCtx, err
}

type CachedResult struct {
    Context context.Context
    Error   error
}
```

## Testing Strategies

### 1. Unit Testing Individual Middleware

Test middleware in isolation:

```go
func TestValidationMiddleware(t *testing.T) {
    tests := []struct {
        name        string
        request     CreateUserRequest
        expectError bool
        errorType   string
    }{
        {
            name: "valid_request",
            request: CreateUserRequest{
                Name:  "John Doe",
                Email: "john@example.com",
            },
            expectError: false,
        },
        {
            name: "invalid_email",
            request: CreateUserRequest{
                Name:  "John Doe",
                Email: "invalid-email",
            },
            expectError: true,
            errorType:   "validation_error",
        },
    }
    
    middleware := &ValidationMiddleware[CreateUserRequest]{
        validator: validator.New(),
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctx := context.Background()
            _, err := middleware.Before(ctx, &tt.request)
            
            if tt.expectError {
                require.Error(t, err)
                var middlewareErr *MiddlewareError
                assert.ErrorAs(t, err, &middlewareErr)
                assert.Equal(t, tt.errorType, string(middlewareErr.Type))
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

### 2. Integration Testing Middleware Chains

Test middleware interactions:

```go
func TestMiddlewareChain(t *testing.T) {
    // Create test router with middleware
    router := typedhttp.NewRouter()
    
    middleware := []typedhttp.MiddlewareEntry{
        {Middleware: &RequestIDMiddleware{}, Config: typedhttp.MiddlewareConfig{Priority: 100}},
        {Middleware: &ValidationMiddleware[TestRequest]{}, Config: typedhttp.MiddlewareConfig{Priority: 90}},
        {Middleware: typedhttp.NewResponseEnvelopeMiddleware[TestResponse](), Config: typedhttp.MiddlewareConfig{Priority: 80}},
    }
    
    // Register test handler
    typedhttp.POST(router, "/test", testHandler)
    
    // Apply middleware
    handlers := router.GetHandlers()
    for i := range handlers {
        handlers[i].MiddlewareEntries = middleware
    }
    
    // Test request
    reqBody := `{"name": "John", "email": "john@example.com"}`
    req := httptest.NewRequest("POST", "/test", strings.NewReader(reqBody))
    req.Header.Set("Content-Type", "application/json")
    
    rr := httptest.NewRecorder()
    router.ServeHTTP(rr, req)
    
    // Verify response
    assert.Equal(t, http.StatusOK, rr.Code)
    
    var response map[string]interface{}
    err := json.Unmarshal(rr.Body.Bytes(), &response)
    require.NoError(t, err)
    
    // Verify envelope structure
    assert.Contains(t, response, "data")
    assert.Contains(t, response, "success")
    assert.True(t, response["success"].(bool))
}
```

### 3. Performance Testing

Benchmark middleware overhead:

```go
func BenchmarkMiddlewareChain(b *testing.B) {
    middleware := []typedhttp.MiddlewareEntry{
        {Middleware: &RequestIDMiddleware{}, Config: typedhttp.MiddlewareConfig{Priority: 100}},
        {Middleware: &ValidationMiddleware[TestRequest]{}, Config: typedhttp.MiddlewareConfig{Priority: 90}},
        {Middleware: &MetricsMiddleware{}, Config: typedhttp.MiddlewareConfig{Priority: 80}},
    }
    
    req := &TestRequest{Name: "John", Email: "john@example.com"}
    ctx := context.Background()
    
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            // Simulate middleware chain execution
            currentCtx := ctx
            for _, entry := range middleware {
                if preMiddleware, ok := entry.Middleware.(typedhttp.TypedPreMiddleware[TestRequest]); ok {
                    var err error
                    currentCtx, err = preMiddleware.Before(currentCtx, req)
                    if err != nil {
                        b.Fatal(err)
                    }
                }
            }
        }
    })
}
```

## Common Anti-patterns

### 1. ❌ God Middleware

Don't create middleware that does everything:

```go
// ❌ Bad - Too many responsibilities
type GodMiddleware struct{}

func (m *GodMiddleware) Before(ctx context.Context, req *interface{}) (context.Context, error) {
    // Authentication
    if !authenticate(req) {
        return ctx, errors.New("auth failed")
    }
    
    // Authorization
    if !authorize(req) {
        return ctx, errors.New("authorization failed")
    }
    
    // Validation
    if !validate(req) {
        return ctx, errors.New("validation failed")
    }
    
    // Rate limiting
    if !checkRateLimit(req) {
        return ctx, errors.New("rate limit exceeded")
    }
    
    // Logging
    log.Printf("Request processed")
    
    return ctx, nil
}
```

### 2. ❌ Middleware Order Dependencies

Avoid implicit dependencies between middleware:

```go
// ❌ Bad - Hidden dependency on order
type UserMiddleware struct{}

func (m *UserMiddleware) Before(ctx context.Context, req *interface{}) (context.Context, error) {
    // Assumes AuthMiddleware ran first and set user_id
    userID := ctx.Value("user_id").(string) // Panic if auth middleware didn't run
    user := getUserFromDB(userID)
    return context.WithValue(ctx, "user", user), nil
}

// ✅ Good - Explicit dependency checking
func (m *UserMiddleware) Before(ctx context.Context, req *interface{}) (context.Context, error) {
    userID, ok := ctx.Value("user_id").(string)
    if !ok {
        return ctx, errors.New("user_id not found in context - ensure auth middleware runs first")
    }
    user := getUserFromDB(userID)
    return context.WithValue(ctx, "user", user), nil
}
```

### 3. ❌ Ignoring Context Cancellation

Always respect context cancellation:

```go
// ❌ Bad - Ignoring context
func (m *SlowMiddleware) Before(ctx context.Context, req *interface{}) (context.Context, error) {
    time.Sleep(5 * time.Second) // Blocks regardless of context cancellation
    return ctx, nil
}

// ✅ Good - Respecting context
func (m *SlowMiddleware) Before(ctx context.Context, req *interface{}) (context.Context, error) {
    select {
    case <-time.After(5 * time.Second):
        return ctx, nil
    case <-ctx.Done():
        return ctx, ctx.Err()
    }
}
```

### 4. ❌ Middleware State Mutation

Avoid mutable state in middleware:

```go
// ❌ Bad - Mutable state
type CounterMiddleware struct {
    count int // Race condition!
}

func (m *CounterMiddleware) Before(ctx context.Context, req *interface{}) (context.Context, error) {
    m.count++ // Not thread-safe
    return context.WithValue(ctx, "count", m.count), nil
}

// ✅ Good - Thread-safe state
type CounterMiddleware struct {
    count int64
}

func (m *CounterMiddleware) Before(ctx context.Context, req *interface{}) (context.Context, error) {
    count := atomic.AddInt64(&m.count, 1)
    return context.WithValue(ctx, "count", count), nil
}
```

## Summary

Effective middleware composition in TypedHTTP requires:

1. **Clear separation of concerns** - Each middleware should have a single responsibility
2. **Explicit dependencies** - Make dependencies clear and testable
3. **Proper error handling** - Use standardized error types and graceful degradation
4. **Performance awareness** - Use pooling, caching, and lazy initialization where appropriate
5. **Comprehensive testing** - Test both individual middleware and their interactions
6. **Avoiding anti-patterns** - Don't create god middleware or hidden dependencies

By following these practices, you can build robust, maintainable, and performant middleware stacks that scale with your application's needs.