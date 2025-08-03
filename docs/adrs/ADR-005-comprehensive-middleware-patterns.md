# ADR-005: Comprehensive Middleware Patterns and Composability

## Status

**Accepted** - Implemented ✅
**Implementation Date**: July 2025

## Executive Summary

This ADR proposes a comprehensive middleware system for typedhttp that extends beyond standard HTTP middleware to include typed middleware patterns, composition utilities, and a rich ecosystem of common middleware implementations. The solution builds upon the existing middleware support while introducing type-safe middleware that can operate on decoded request/response data, composable middleware chains, and integration patterns that maintain the library's focus on type safety and developer experience.

## Context

The current typedhttp implementation has basic middleware support through the standard Go `func(http.Handler) http.Handler` pattern. While functional, this approach has several limitations as our API ecosystem grows:

1. **HTTP-Only Middleware**: Current middleware operates only at the HTTP transport layer, without access to typed request/response data
2. **Limited Composition**: No built-in utilities for conditional middleware, ordering, or middleware groups
3. **Boilerplate for Common Patterns**: Developers must implement common middleware (auth, logging, rate limiting) from scratch
4. **No Type Safety**: Middleware cannot leverage the type safety benefits of typedhttp's core design
5. **Scattered Configuration**: Middleware configuration is applied per-handler without consistent patterns

### Current Middleware Architecture

```go
// Current basic middleware support
type Middleware func(http.Handler) http.Handler

func WithMiddleware(middleware ...Middleware) HandlerOption {
    return func(cfg *HandlerConfig) {
        cfg.Middleware = append(cfg.Middleware, middleware...)
    }
}

// Applied at HTTPHandler level in router.go
func (h *HTTPHandler[TRequest, TResponse]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    handler := http.HandlerFunc(h.handleRequest)
    
    // Apply middleware in reverse order
    for i := len(h.config.Middleware) - 1; i >= 0; i-- {
        handler = h.config.Middleware[i](handler)
    }
    
    handler.ServeHTTP(w, r)
}
```

### Success Criteria

1. **Backward Compatibility**: Preserve existing middleware interface and patterns
2. **Type Safety**: Enable middleware that operates on typed request/response data
3. **Composability**: Provide utilities for middleware composition, ordering, and conditional application
4. **Rich Ecosystem**: Include common middleware implementations out-of-the-box
5. **Performance**: Maintain zero-overhead abstractions and efficient middleware chains
6. **Developer Experience**: Intuitive APIs for both simple and advanced middleware patterns
7. **Observability**: Comprehensive observability integration for middleware chains

## Decision

We will implement a **Comprehensive Middleware System** that extends the current HTTP middleware pattern with typed middleware interfaces, composition utilities, and a rich library of standard middleware implementations.

### Core Architecture

#### 1. Enhanced Middleware Types

```go
// Standard HTTP middleware (preserved for backward compatibility)
type Middleware func(http.Handler) http.Handler

// Typed middleware interfaces for different phases
type TypedPreMiddleware[TRequest any] interface {
    Before(ctx context.Context, req *TRequest) (context.Context, error)
}

type TypedPostMiddleware[TResponse any] interface {
    After(ctx context.Context, resp *TResponse) (*TResponse, error)
}

// Full typed middleware with access to both request and response
type TypedMiddleware[TRequest, TResponse any] interface {
    Before(ctx context.Context, req *TRequest) (context.Context, error)
    After(ctx context.Context, req *TRequest, resp *TResponse, err error) (*TResponse, error)
}

// Conditional middleware execution
type ConditionalFunc func(*http.Request) bool

// Middleware configuration
type MiddlewareConfig struct {
    Priority    int             // Execution order priority (-100 to 100)
    Conditional ConditionalFunc // Optional condition for execution
    Scope       MiddlewareScope // Application scope
    Name        string          // Middleware identification
    Metadata    map[string]any  // Custom metadata
}

type MiddlewareScope int

const (
    ScopeGlobal MiddlewareScope = iota  // Applied to all handlers
    ScopeGroup                          // Applied to route groups
    ScopeHandler                        // Applied to specific handlers
)
```

#### 2. Middleware Registry and Builder

```go
// Middleware registry for managing middleware chains
type MiddlewareRegistry struct {
    global   []MiddlewareEntry
    groups   map[string][]MiddlewareEntry
    handlers map[string][]MiddlewareEntry
    mu       sync.RWMutex
}

type MiddlewareEntry struct {
    Middleware interface{} // Middleware, TypedPreMiddleware, TypedPostMiddleware, or TypedMiddleware
    Config     MiddlewareConfig
}

// Fluent builder for middleware composition
type MiddlewareBuilder struct {
    entries []MiddlewareEntry
}

func NewMiddlewareBuilder() *MiddlewareBuilder {
    return &MiddlewareBuilder{entries: make([]MiddlewareEntry, 0)}
}

func (b *MiddlewareBuilder) Add(mw interface{}, opts ...MiddlewareOption) *MiddlewareBuilder {
    config := MiddlewareConfig{Priority: 0}
    for _, opt := range opts {
        opt(&config)
    }
    
    b.entries = append(b.entries, MiddlewareEntry{
        Middleware: mw,
        Config:     config,
    })
    return b
}

func (b *MiddlewareBuilder) OnlyFor(condition ConditionalFunc) *MiddlewareBuilder {
    if len(b.entries) > 0 {
        b.entries[len(b.entries)-1].Config.Conditional = condition
    }
    return b
}

func (b *MiddlewareBuilder) WithPriority(priority int) *MiddlewareBuilder {
    if len(b.entries) > 0 {
        b.entries[len(b.entries)-1].Config.Priority = priority
    }
    return b
}

func (b *MiddlewareBuilder) Build() []MiddlewareEntry {
    // Sort by priority (higher priority executes first)
    sort.Slice(b.entries, func(i, j int) bool {
        return b.entries[i].Config.Priority > b.entries[j].Config.Priority
    })
    return b.entries
}
```

#### 3. Enhanced TypedRouter with Middleware Support

```go
type TypedRouter struct {
    *http.ServeMux
    registry    *MiddlewareRegistry
    globalMW    []MiddlewareEntry
    groups      map[string]*MiddlewareGroup
    schemas     map[reflect.Type]Schema
    handlers    []HandlerRegistration
}

// Global middleware registration
func (r *TypedRouter) Use(middleware ...interface{}) {
    for _, mw := range middleware {
        r.globalMW = append(r.globalMW, MiddlewareEntry{
            Middleware: mw,
            Config:     MiddlewareConfig{Scope: ScopeGlobal},
        })
    }
}

// Middleware group for route organization
func (r *TypedRouter) Group(pattern string) *MiddlewareGroup {
    group := &MiddlewareGroup{
        router:  r,
        pattern: pattern,
        middleware: make([]MiddlewareEntry, 0),
    }
    r.groups[pattern] = group
    return group
}

// Enhanced handler registration with middleware
func (r *TypedRouter) RegisterHandler[TReq, TResp any](
    method, path string,
    handler Handler[TReq, TResp],
    opts ...HandlerOption,
) {
    // Apply global middleware, group middleware, and handler-specific middleware
    allMiddleware := r.compileMiddlewareChain(path, opts...)
    
    // Create enhanced HTTP handler with middleware support
    httpHandler := &EnhancedHTTPHandler[TReq, TResp]{
        handler:          handler,
        middleware:       allMiddleware,
        typedMiddleware:  extractTypedMiddleware[TReq, TResp](allMiddleware),
        config:          buildHandlerConfig(opts...),
    }
    
    r.Handle(fmt.Sprintf("%s %s", method, path), httpHandler)
}

type MiddlewareGroup struct {
    router     *TypedRouter
    pattern    string
    middleware []MiddlewareEntry
}

func (g *MiddlewareGroup) Use(middleware ...interface{}) *MiddlewareGroup {
    for _, mw := range middleware {
        g.middleware = append(g.middleware, MiddlewareEntry{
            Middleware: mw,
            Config:     MiddlewareConfig{Scope: ScopeGroup},
        })
    }
    return g
}

func (g *MiddlewareGroup) GET[TReq, TResp any](path string, handler Handler[TReq, TResp], opts ...HandlerOption) {
    g.router.RegisterHandler("GET", g.pattern+path, handler, opts...)
}
```

#### 4. Enhanced HTTPHandler with Typed Middleware

```go
type EnhancedHTTPHandler[TRequest, TResponse any] struct {
    handler         Handler[TRequest, TResponse]
    middleware      []MiddlewareEntry
    typedMiddleware TypedMiddlewareChain[TRequest, TResponse]
    config          HandlerConfig
}

type TypedMiddlewareChain[TRequest, TResponse any] struct {
    preMiddleware  []TypedPreMiddleware[TRequest]
    postMiddleware []TypedPostMiddleware[TResponse]
    fullMiddleware []TypedMiddleware[TRequest, TResponse]
}

func (h *EnhancedHTTPHandler[TRequest, TResponse]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Create middleware chain starting with typed middleware wrapper
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        h.handleRequestWithTypedMiddleware(w, r)
    })
    
    // Apply standard HTTP middleware in reverse order
    for i := len(h.middleware) - 1; i >= 0; i-- {
        entry := h.middleware[i]
        
        // Check conditional execution
        if entry.Config.Conditional != nil && !entry.Config.Conditional(r) {
            continue
        }
        
        if httpMW, ok := entry.Middleware.(Middleware); ok {
            handler = httpMW(handler)
        }
    }
    
    handler.ServeHTTP(w, r)
}

func (h *EnhancedHTTPHandler[TRequest, TResponse]) handleRequestWithTypedMiddleware(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    // Decode request
    var req TRequest
    if err := h.config.Decoder.Decode(r, &req); err != nil {
        h.writeError(w, err)
        return
    }
    
    // Execute pre-middleware
    for _, mw := range h.typedMiddleware.preMiddleware {
        var err error
        ctx, err = mw.Before(ctx, &req)
        if err != nil {
            h.writeError(w, err)
            return
        }
    }
    
    // Execute full typed middleware Before hooks
    for _, mw := range h.typedMiddleware.fullMiddleware {
        var err error
        ctx, err = mw.Before(ctx, &req)
        if err != nil {
            h.writeError(w, err)
            return
        }
    }
    
    // Execute handler
    resp, err := h.handler.Handle(ctx, req)
    
    // Execute full typed middleware After hooks (in reverse order)
    for i := len(h.typedMiddleware.fullMiddleware) - 1; i >= 0; i-- {
        mw := h.typedMiddleware.fullMiddleware[i]
        resp, err = mw.After(ctx, &req, &resp, err)
    }
    
    // Execute post-middleware
    for _, mw := range h.typedMiddleware.postMiddleware {
        if err == nil {
            resp, err = mw.After(ctx, &resp)
        }
    }
    
    // Handle response
    if err != nil {
        h.writeError(w, err)
        return
    }
    
    h.writeResponse(w, resp)
}
```

### Standard Middleware Library

#### 1. Authentication Middleware

```go
// JWT Authentication
type JWTMiddleware struct {
    secret    []byte
    algorithm string
    claims    func() jwt.Claims
}

func NewJWTMiddleware(secret []byte, opts ...JWTOption) Middleware {
    config := &JWTConfig{
        Algorithm: "HS256",
        Claims:    func() jwt.Claims { return jwt.MapClaims{} },
    }
    
    for _, opt := range opts {
        opt(config)
    }
    
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := extractTokenFromRequest(r)
            if token == "" {
                http.Error(w, "Missing authentication token", http.StatusUnauthorized)
                return
            }
            
            claims := config.Claims()
            parsedToken, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
                return config.Secret, nil
            })
            
            if err != nil || !parsedToken.Valid {
                http.Error(w, "Invalid authentication token", http.StatusUnauthorized)
                return
            }
            
            ctx := context.WithValue(r.Context(), "jwt_claims", claims)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// API Key Authentication
func APIKeyMiddleware(store APIKeyStore) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            apiKey := r.Header.Get("X-API-Key")
            if apiKey == "" {
                http.Error(w, "Missing API key", http.StatusUnauthorized)
                return
            }
            
            user, err := store.ValidateAPIKey(r.Context(), apiKey)
            if err != nil {
                http.Error(w, "Invalid API key", http.StatusUnauthorized)
                return
            }
            
            ctx := context.WithValue(r.Context(), "user", user)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// Typed authentication middleware that adds user context
type AuthenticationMiddleware[TRequest any] struct {
    validator TokenValidator
}

func (m *AuthenticationMiddleware[TRequest]) Before(ctx context.Context, req *TRequest) (context.Context, error) {
    // Extract user from JWT claims or API key validation
    user, ok := ctx.Value("user").(*User)
    if !ok {
        return ctx, errors.New("authentication required")
    }
    
    // Add user to context for handler access
    return context.WithValue(ctx, "authenticated_user", user), nil
}
```

#### 2. Rate Limiting Middleware

```go
// IP-based rate limiting
func IPRateLimitMiddleware(requests int, window time.Duration) Middleware {
    limiter := rate.NewLimiter(rate.Every(window/time.Duration(requests)), requests)
    limiters := make(map[string]*rate.Limiter)
    mu := sync.RWMutex{}
    
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ip := getClientIP(r)
            
            mu.RLock()
            ipLimiter, exists := limiters[ip]
            mu.RUnlock()
            
            if !exists {
                mu.Lock()
                ipLimiter = rate.NewLimiter(limiter.Limit(), limiter.Burst())
                limiters[ip] = ipLimiter
                mu.Unlock()
            }
            
            if !ipLimiter.Allow() {
                http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}

// User-based rate limiting with typed middleware
type UserRateLimitMiddleware[TRequest any] struct {
    limiter map[string]*rate.Limiter
    mu      sync.RWMutex
    rate    rate.Limit
    burst   int
}

func NewUserRateLimitMiddleware[TRequest any](requests int, window time.Duration) *UserRateLimitMiddleware[TRequest] {
    return &UserRateLimitMiddleware[TRequest]{
        limiter: make(map[string]*rate.Limiter),
        rate:    rate.Every(window / time.Duration(requests)),
        burst:   requests,
    }
}

func (m *UserRateLimitMiddleware[TRequest]) Before(ctx context.Context, req *TRequest) (context.Context, error) {
    user, ok := ctx.Value("authenticated_user").(*User)
    if !ok {
        return ctx, nil // Skip rate limiting for unauthenticated requests
    }
    
    m.mu.RLock()
    userLimiter, exists := m.limiter[user.ID]
    m.mu.RUnlock()
    
    if !exists {
        m.mu.Lock()
        userLimiter = rate.NewLimiter(m.rate, m.burst)
        m.limiter[user.ID] = userLimiter
        m.mu.Unlock()
    }
    
    if !userLimiter.Allow() {
        return ctx, &RateLimitError{
            UserID: user.ID,
            Limit:  m.burst,
            Window: time.Duration(1) / time.Duration(m.rate),
        }
    }
    
    return ctx, nil
}
```

#### 3. Observability Middleware

```go
// Structured logging middleware
func LoggingMiddleware(logger *slog.Logger) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            
            wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
            
            logger.InfoContext(r.Context(), "request_started",
                "method", r.Method,
                "path", r.URL.Path,
                "remote_addr", r.RemoteAddr,
                "user_agent", r.UserAgent(),
            )
            
            next.ServeHTTP(wrapped, r)
            
            duration := time.Since(start)
            logger.InfoContext(r.Context(), "request_completed",
                "method", r.Method,
                "path", r.URL.Path,
                "status_code", wrapped.statusCode,
                "duration_ms", duration.Milliseconds(),
                "response_size", wrapped.size,
            )
        })
    }
}

// Metrics collection middleware
func MetricsMiddleware(collector MetricsCollector) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            
            wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
            next.ServeHTTP(wrapped, r)
            
            duration := time.Since(start)
            
            collector.RecordHTTPRequest(
                r.Method,
                r.URL.Path,
                wrapped.statusCode,
                duration,
            )
        })
    }
}

// Request ID middleware
func RequestIDMiddleware() Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            requestID := r.Header.Get("X-Request-ID")
            if requestID == "" {
                requestID = generateRequestID()
            }
            
            w.Header().Set("X-Request-ID", requestID)
            ctx := context.WithValue(r.Context(), "request_id", requestID)
            
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// Typed observability middleware that adds handler metadata
type ObservabilityMiddleware[TRequest, TResponse any] struct {
    logger    *slog.Logger
    metrics   MetricsCollector
    tracer    trace.Tracer
}

func (m *ObservabilityMiddleware[TRequest, TResponse]) Before(ctx context.Context, req *TRequest) (context.Context, error) {
    // Start tracing span
    ctx, span := m.tracer.Start(ctx, "handler_execution")
    span.SetAttributes(
        attribute.String("request_type", reflect.TypeOf(*req).Name()),
        attribute.String("response_type", reflect.TypeOf(*new(TResponse)).Name()),
    )
    
    return ctx, nil
}

func (m *ObservabilityMiddleware[TRequest, TResponse]) After(ctx context.Context, req *TRequest, resp *TResponse, err error) (*TResponse, error) {
    // End tracing span
    span := trace.SpanFromContext(ctx)
    if err != nil {
        span.SetStatus(codes.Error, err.Error())
    } else {
        span.SetStatus(codes.Ok, "")
    }
    span.End()
    
    // Record metrics
    m.metrics.RecordHandlerExecution(
        reflect.TypeOf(*req).Name(),
        reflect.TypeOf(*resp).Name(),
        err == nil,
    )
    
    return resp, err
}
```

#### 4. Security Middleware

```go
// CORS middleware
func CORSMiddleware(config CORSConfig) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            origin := r.Header.Get("Origin")
            
            if config.AllowAllOrigins || contains(config.AllowedOrigins, origin) {
                w.Header().Set("Access-Control-Allow-Origin", origin)
                w.Header().Set("Access-Control-Allow-Credentials", "true")
            }
            
            w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
            w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
            
            if r.Method == "OPTIONS" {
                w.WriteHeader(http.StatusNoContent)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}

// Security headers middleware
func SecurityHeadersMiddleware(config SecurityConfig) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if config.CSP != "" {
                w.Header().Set("Content-Security-Policy", config.CSP)
            }
            
            if config.HSTS {
                w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
            }
            
            w.Header().Set("X-Content-Type-Options", "nosniff")
            w.Header().Set("X-Frame-Options", "DENY")
            w.Header().Set("X-XSS-Protection", "1; mode=block")
            
            next.ServeHTTP(w, r)
        })
    }
}

// Input validation middleware (typed)
type ValidationMiddleware[TRequest any] struct {
    validator *validator.Validate
}

func (m *ValidationMiddleware[TRequest]) Before(ctx context.Context, req *TRequest) (context.Context, error) {
    if err := m.validator.Struct(req); err != nil {
        return ctx, &ValidationError{
            Message: "Request validation failed",
            Fields:  extractValidationErrors(err),
        }
    }
    
    return ctx, nil
}
```

### Usage Examples

#### 1. Simple Middleware Usage

```go
// Basic router setup with global middleware
router := typedhttp.NewRouter()

// Apply global middleware
router.Use(
    RequestIDMiddleware(),
    LoggingMiddleware(logger),
    MetricsMiddleware(metrics),
    CORSMiddleware(CORSConfig{
        AllowedOrigins: []string{"https://app.example.com"},
        AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
    }),
)

// Simple handler registration inherits global middleware
router.GET("/api/v1/users/{id}", getUserHandler)
```

#### 2. Route Groups with Middleware

```go
// API route group with authentication
apiGroup := router.Group("/api/v1")
apiGroup.Use(
    JWTMiddleware(jwtSecret),
    UserRateLimitMiddleware[any](100, time.Hour),
)

// Admin route group with additional security
adminGroup := apiGroup.Group("/admin")
adminGroup.Use(
    RequireRoleMiddleware("admin"),
    SecurityHeadersMiddleware(SecurityConfig{
        CSP:  "default-src 'self'",
        HSTS: true,
    }),
)

adminGroup.GET("/users", listUsersHandler)
adminGroup.POST("/users", createUserHandler)
```

#### 3. Handler-Specific Middleware

```go
// Handler with specific middleware requirements
router.POST("/api/v1/transactions", createTransactionHandler,
    WithMiddleware(
        IPRateLimitMiddleware(10, time.Minute), // Specific rate limiting
        CompressionMiddleware("gzip"),          // Response compression
    ),
    WithTypedPreMiddleware(&ValidationMiddleware[CreateTransactionRequest]{
        validator: customValidator,
    }),
    WithTypedPostMiddleware(&AuditMiddleware[CreateTransactionResponse]{
        auditService: auditService,
    }),
)
```

#### 4. Conditional Middleware

```go
// Middleware builder with conditions
middleware := NewMiddlewareBuilder().
    Add(RequestIDMiddleware()).
    Add(LoggingMiddleware(logger)).
    Add(AuthenticationMiddleware(), OnlyFor(func(r *http.Request) bool {
        return !strings.HasPrefix(r.URL.Path, "/public/")
    })).
    Add(RateLimitMiddleware(100, time.Hour), OnlyFor(func(r *http.Request) bool {
        return r.Header.Get("X-Client-Type") == "mobile"
    })).
    WithPriority(10). // High priority for security middleware
    Build()

router.RegisterHandler("GET", "/api/v1/data", dataHandler, WithMiddleware(middleware...))
```

#### 5. Typed Middleware with Business Logic Integration

```go
type TransactionValidationMiddleware struct {
    budgetService BudgetService
}

func (m *TransactionValidationMiddleware) Before(ctx context.Context, req *CreateTransactionRequest) (context.Context, error) {
    // Access to typed request data
    budget, err := m.budgetService.GetBudget(ctx, req.FamilyID)
    if err != nil {
        return ctx, err
    }
    
    if req.Amount > budget.GetRemainingAmount(req.Category) {
        return ctx, &BudgetExceededError{
            Category: req.Category,
            Amount:   req.Amount,
            Remaining: budget.GetRemainingAmount(req.Category),
        }
    }
    
    // Add budget context for handler use
    ctx = context.WithValue(ctx, "budget", budget)
    return ctx, nil
}

// Registration with typed middleware
router.POST("/api/v1/transactions", createTransactionHandler,
    WithTypedPreMiddleware(&TransactionValidationMiddleware{
        budgetService: budgetService,
    }),
)
```

### Advanced Patterns

#### 1. Middleware Inheritance and Override

```go
// Base configuration for all API endpoints
baseConfig := []HandlerOption{
    WithMiddleware(
        RequestIDMiddleware(),
        LoggingMiddleware(logger),
        MetricsMiddleware(metrics),
    ),
}

// Specific handler overrides base config
router.POST("/api/v1/sensitive-operation", sensitiveHandler,
    append(baseConfig, 
        WithMiddleware(
            AdditionalSecurityMiddleware(),
            AuditMiddleware(),
        ),
        WithTypedPreMiddleware(&InputSanitizationMiddleware[SensitiveRequest]{}),
    )...,
)
```

#### 2. Middleware Factories and Configuration

```go
// Middleware factory for consistent configuration
func NewAPIMiddlewareStack(config APIConfig) []MiddlewareEntry {
    builder := NewMiddlewareBuilder()
    
    // Core middleware (always applied)
    builder.Add(RequestIDMiddleware(), WithPriority(100)).
           Add(LoggingMiddleware(config.Logger), WithPriority(90)).
           Add(MetricsMiddleware(config.Metrics), WithPriority(80))
    
    // Conditional middleware based on configuration
    if config.EnableAuth {
        builder.Add(JWTMiddleware(config.JWTSecret), WithPriority(70))
    }
    
    if config.EnableRateLimit {
        builder.Add(IPRateLimitMiddleware(config.RateLimit.Requests, config.RateLimit.Window), WithPriority(60))
    }
    
    if config.EnableCORS {
        builder.Add(CORSMiddleware(config.CORS), WithPriority(50))
    }
    
    return builder.Build()
}

// Usage
apiMiddleware := NewAPIMiddlewareStack(APIConfig{
    EnableAuth:      true,
    EnableRateLimit: true,
    EnableCORS:      true,
    JWTSecret:       jwtSecret,
    RateLimit: RateLimitConfig{
        Requests: 100,
        Window:   time.Hour,
    },
    CORS: CORSConfig{
        AllowedOrigins: []string{"https://app.example.com"},
    },
})

router.Use(apiMiddleware...)
```

#### 3. Middleware Testing

```go
func TestAuthenticationMiddleware(t *testing.T) {
    tests := []struct {
        name           string
        token          string
        expectedStatus int
        expectedUser   *User
    }{
        {
            name:           "valid_token",
            token:          "Bearer " + validJWT,
            expectedStatus: http.StatusOK,
            expectedUser:   &User{ID: "user123"},
        },
        {
            name:           "invalid_token",
            token:          "Bearer invalid",
            expectedStatus: http.StatusUnauthorized,
            expectedUser:   nil,
        },
        {
            name:           "missing_token",
            token:          "",
            expectedStatus: http.StatusUnauthorized,
            expectedUser:   nil,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test handler that requires authentication
            handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                user := r.Context().Value("user")
                if user != nil {
                    w.WriteHeader(http.StatusOK)
                    json.NewEncoder(w).Encode(user)
                } else {
                    w.WriteHeader(http.StatusUnauthorized)
                }
            })
            
            // Apply authentication middleware
            mw := JWTMiddleware(jwtSecret)
            wrappedHandler := mw(handler)
            
            // Create test request
            req := httptest.NewRequest("GET", "/test", nil)
            if tt.token != "" {
                req.Header.Set("Authorization", tt.token)
            }
            
            rr := httptest.NewRecorder()
            wrappedHandler.ServeHTTP(rr, req)
            
            assert.Equal(t, tt.expectedStatus, rr.Code)
            
            if tt.expectedUser != nil {
                var user User
                err := json.NewDecoder(rr.Body).Decode(&user)
                require.NoError(t, err)
                assert.Equal(t, tt.expectedUser.ID, user.ID)
            }
        })
    }
}

func TestTypedValidationMiddleware(t *testing.T) {
    middleware := &ValidationMiddleware[CreateUserRequest]{
        validator: validator.New(),
    }
    
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
                Age:   25,
            },
            expectError: false,
        },
        {
            name: "invalid_email",
            request: CreateUserRequest{
                Name:  "John Doe",
                Email: "invalid-email",
                Age:   25,
            },
            expectError: true,
            errorType:   "validation",
        },
        {
            name: "missing_name",
            request: CreateUserRequest{
                Email: "john@example.com",
                Age:   25,
            },
            expectError: true,
            errorType:   "validation",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctx := context.Background()
            
            _, err := middleware.Before(ctx, &tt.request)
            
            if tt.expectError {
                require.Error(t, err)
                if tt.errorType == "validation" {
                    var valErr *ValidationError
                    assert.ErrorAs(t, err, &valErr)
                }
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

## Implementation Plan

### Phase 1: Core Infrastructure (Weeks 1-2)
- **Week 1**: Implement middleware registry, builder, and configuration types
- **Week 2**: Enhance TypedRouter with middleware group support and Enhanced HTTPHandler

### Phase 2: Typed Middleware System (Weeks 3-4)
- **Week 3**: Implement typed middleware interfaces and integration with HTTPHandler
- **Week 4**: Add middleware composition utilities and conditional execution

### Phase 3: Standard Middleware Library (Weeks 5-6)
- **Week 5**: Implement authentication, rate limiting, and security middleware
- **Week 6**: Add observability middleware (logging, metrics, tracing, request ID)

### Phase 4: Advanced Features & Testing (Weeks 7-8)
- **Week 7**: Implement middleware factories, inheritance patterns, and performance optimization
- **Week 8**: Comprehensive testing, documentation, and integration with existing examples

## Benefits

### 1. Enhanced Type Safety
- Typed middleware can operate on decoded request/response data
- Compile-time validation of middleware compatibility with handlers
- Type-safe context passing between middleware and handlers

### 2. Rich Middleware Ecosystem
- Comprehensive library of common middleware patterns
- Consistent configuration and behavior across middleware
- Easy integration with external libraries and services

### 3. Flexible Composition
- Middleware builder pattern for complex compositions
- Conditional middleware execution based on request characteristics
- Priority-based ordering and middleware groups

### 4. Backward Compatibility
- Preserves existing middleware interface and patterns
- Gradual migration path for existing middleware
- Standard HTTP middleware continue to work unchanged

### 5. Performance Optimization
- Efficient middleware chain compilation and execution
- Zero-overhead abstractions for typed middleware
- Optional middleware execution based on conditions

### 6. Developer Experience
- Intuitive APIs for both simple and advanced middleware patterns
- Consistent error handling and observability integration
- Rich testing utilities for middleware development

## Risks and Mitigations

### Risk 1: Complexity and Learning Curve
**Impact**: MEDIUM - Multiple middleware types and patterns may overwhelm developers
**Mitigation**:
- Provide clear documentation with progressive complexity examples
- Start with simple HTTP middleware and gradually introduce typed middleware
- Create middleware templates and code generators for common patterns

### Risk 2: Performance Overhead
**Impact**: LOW-MEDIUM - Multiple middleware layers could impact performance
**Mitigation**:
- Benchmark middleware chains during development
- Implement lazy evaluation for conditional middleware
- Provide profiling tools for middleware performance analysis

### Risk 3: Middleware Ordering Conflicts
**Impact**: MEDIUM - Complex priority systems may lead to unexpected behavior
**Mitigation**:
- Clear documentation of middleware execution order
- Debug utilities to visualize middleware chains
- Default priority ranges for different middleware categories

### Risk 4: Type Safety Complexity
**Impact**: MEDIUM - Generic typed middleware may be difficult to debug
**Mitigation**:
- Provide clear error messages for type mismatches
- Comprehensive testing utilities for typed middleware
- Fallback to standard HTTP middleware for complex cases

## Alternatives Considered

### 1. Decorator Pattern Only
**Rejected**: Less flexible than middleware chain composition and doesn't integrate well with existing HTTP middleware ecosystem.

### 2. Aspect-Oriented Programming (AOP)
**Rejected**: Too complex for Go's simplicity philosophy and would require code generation or runtime reflection.

### 3. Plugin Architecture
**Rejected**: Adds unnecessary complexity and doesn't provide the type safety benefits we're seeking.

## Migration Strategy

### Phase 1: Introduction (Backward Compatible)
- Add new middleware types alongside existing patterns
- Provide migration utilities for existing middleware
- Update documentation with new patterns

### Phase 2: Gradual Adoption
- Migrate high-value middleware to new patterns (authentication, rate limiting)
- Add typed middleware to new handlers
- Provide both old and new patterns in examples

### Phase 3: Standardization
- Deprecate old patterns in favor of new ones
- Complete migration of all middleware
- Remove legacy middleware support (optional)

## Conclusion

This comprehensive middleware system extends typedhttp's type safety benefits to the middleware layer while maintaining backward compatibility and providing a rich ecosystem of standard middleware implementations. The design supports both simple use cases through standard HTTP middleware and advanced patterns through typed middleware that can operate on decoded request/response data.

The implementation provides significant value through:
- Enhanced type safety for middleware operations
- Rich ecosystem of standard middleware implementations
- Flexible composition and conditional execution patterns
- Seamless integration with existing typedhttp architecture
- Performance-optimized middleware chain execution

## Decision Recommendation

**PROCEED WITH IMPLEMENTATION** - This ADR presents a well-architected extension to typedhttp that addresses real needs for middleware composition and provides significant value through type safety and ecosystem richness.

### Recommended Implementation Order
1. Start with core infrastructure and standard HTTP middleware enhancements
2. Implement typed middleware system for high-value use cases
3. Build standard middleware library focusing on authentication and observability
4. Add advanced features and comprehensive testing

### Success Criteria
- Backward compatibility with existing middleware patterns
- Performance parity or improvement over current implementation
- Comprehensive test coverage including integration tests
- Rich documentation with real-world examples
- Positive developer feedback on usability and type safety

## References

- [Go HTTP Middleware Patterns](https://www.alexedwards.net/blog/making-and-using-middleware)
- [Chi Router Middleware](https://github.com/go-chi/chi/tree/master/middleware)
- [Gin Framework Middleware](https://gin-gonic.com/docs/examples/using-middleware/)
- [Echo Framework Middleware](https://echo.labstack.com/middleware/)
- [OWASP Security Headers](https://owasp.org/www-project-secure-headers/)
- [Rate Limiting Strategies](https://blog.cloudflare.com/counting-things-a-lot-of-different-things/)