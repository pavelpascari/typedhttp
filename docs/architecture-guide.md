# TypedHTTP Architecture Guide: Middleware & OpenAPI Patterns

This comprehensive guide demonstrates how to architect production-ready applications using TypedHTTP with advanced middleware patterns and automatic OpenAPI generation.

## Table of Contents

1. [Architecture Principles](#architecture-principles)
2. [Middleware Patterns](#middleware-patterns)
3. [Response Schema Modification](#response-schema-modification)
4. [Service Architecture Examples](#service-architecture-examples)
5. [Best Practices](#best-practices)
6. [OpenAPI Documentation Strategies](#openapi-documentation-strategies)

## Architecture Principles

### Core Design Philosophy

TypedHTTP follows these architectural principles:

1. **Type Safety First**: Compile-time guarantees for request/response handling
2. **Composable Middleware**: Mix-and-match middleware for different needs
3. **Accurate Documentation**: OpenAPI specs reflect actual API behavior
4. **Performance Optimization**: Minimal overhead where needed
5. **Developer Experience**: Intuitive APIs with comprehensive tooling

### Layered Architecture

```
┌─────────────────────────────────────────┐
│            HTTP Transport               │
├─────────────────────────────────────────┤
│         Middleware Stack                │
│  ┌─────────────────────────────────┐    │
│  │ Security & Rate Limiting        │    │
│  ├─────────────────────────────────┤    │
│  │ Request/Response Processing     │    │
│  ├─────────────────────────────────┤    │
│  │ Business Logic Handlers         │    │
│  ├─────────────────────────────────┤    │
│  │ Response Transformation         │    │
│  └─────────────────────────────────┘    │
├─────────────────────────────────────────┤
│         OpenAPI Generation              │
├─────────────────────────────────────────┤
│          Type Validation                │
└─────────────────────────────────────────┘
```

## Middleware Patterns

### 1. Standard HTTP Middleware

Traditional middleware that operates at the HTTP transport layer:

```go
type Middleware func(http.Handler) http.Handler

func LoggingMiddleware(logger *slog.Logger) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            next.ServeHTTP(w, r)
            logger.Info("Request completed",
                "method", r.Method,
                "path", r.URL.Path,
                "duration", time.Since(start),
            )
        })
    }
}
```

### 2. Typed Pre-Middleware

Operates on decoded request data before handler execution:

```go
type TypedPreMiddleware[TRequest any] interface {
    Before(ctx context.Context, req *TRequest) (context.Context, error)
}

type ValidationMiddleware[TRequest any] struct {
    validator *validator.Validate
}

func (m *ValidationMiddleware[TRequest]) Before(ctx context.Context, req *TRequest) (context.Context, error) {
    if err := m.validator.Struct(req); err != nil {
        return ctx, &ValidationError{Message: "Request validation failed"}
    }
    return ctx, nil
}
```

### 3. Typed Post-Middleware

Operates on response data after handler execution:

```go
type TypedPostMiddleware[TResponse any] interface {
    After(ctx context.Context, resp *TResponse) (*TResponse, error)
}

type CacheMiddleware[TResponse any] struct {
    TTL time.Duration
}

func (m *CacheMiddleware[TResponse]) After(ctx context.Context, resp *TResponse) (*CachedResponse[TResponse], error) {
    return &CachedResponse[TResponse]{
        Data:      *resp,
        CachedAt:  time.Now(),
        ExpiresAt: time.Now().Add(m.TTL),
    }, nil
}
```

### 4. Full Lifecycle Middleware

Provides hooks for both request and response processing:

```go
type TypedMiddleware[TRequest, TResponse any] interface {
    Before(ctx context.Context, req *TRequest) (context.Context, error)
    After(ctx context.Context, req *TRequest, resp *TResponse, err error) (*TResponse, error)
}

type AuditMiddleware[TRequest, TResponse any] struct {
    auditService AuditService
}

func (m *AuditMiddleware[TRequest, TResponse]) Before(ctx context.Context, req *TRequest) (context.Context, error) {
    m.auditService.LogRequest(ctx, req)
    return ctx, nil
}

func (m *AuditMiddleware[TRequest, TResponse]) After(ctx context.Context, req *TRequest, resp *TResponse, err error) (*TResponse, error) {
    m.auditService.LogResponse(ctx, req, resp, err)
    return resp, err
}
```

## Response Schema Modification

### The Challenge

When middleware transforms response structures, OpenAPI documentation can become inaccurate:

```go
// Handler returns this
type User struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

// But envelope middleware transforms it to this
type APIResponse[T any] struct {
    Data    *T      `json:"data,omitempty"`
    Error   *string `json:"error,omitempty"`
    Success bool    `json:"success"`
}
```

### The Solution: ResponseSchemaModifier

Middleware can implement this interface to declare how they modify OpenAPI schemas:

```go
type ResponseSchemaModifier interface {
    ModifyResponseSchema(ctx context.Context, originalSchema *openapi3.SchemaRef) (*openapi3.SchemaRef, error)
}

type ResponseEnvelopeMiddleware[TResponse any] struct {
    includeRequestID bool
    includeTimestamp bool
}

func (m *ResponseEnvelopeMiddleware[TResponse]) ModifyResponseSchema(ctx context.Context, originalSchema *openapi3.SchemaRef) (*openapi3.SchemaRef, error) {
    return &openapi3.SchemaRef{
        Value: &openapi3.Schema{
            Type: &openapi3.Types{"object"},
            Properties: map[string]*openapi3.SchemaRef{
                "data": originalSchema,
                "error": {Value: &openapi3.Schema{Type: &openapi3.Types{"string"}, Nullable: true}},
                "success": {Value: &openapi3.Schema{Type: &openapi3.Types{"boolean"}}},
            },
            Required: []string{"success"},
        },
    }, nil
}
```

## Service Architecture Examples

### 1. Public API Gateway

**Purpose**: Customer-facing API with full security and observability

**Middleware Stack**:
```go
func createPublicAPIMiddleware() []typedhttp.MiddlewareEntry {
    return []typedhttp.MiddlewareEntry{
        // Security headers (Priority: 100)
        {Middleware: &SecurityHeadersMiddleware{}, Config: typedhttp.MiddlewareConfig{Priority: 100}},
        
        // Rate limiting (Priority: 90)
        {Middleware: &RateLimitMiddleware{Limit: 100}, Config: typedhttp.MiddlewareConfig{Priority: 90}},
        
        // Authentication (Priority: 80)
        {Middleware: &JWTMiddleware{}, Config: typedhttp.MiddlewareConfig{Priority: 80}},
        
        // Request tracking (Priority: 70)
        {Middleware: &RequestTrackingMiddleware{}, Config: typedhttp.MiddlewareConfig{Priority: 70}},
        
        // Response envelope (Priority: 60)
        {Middleware: typedhttp.NewResponseEnvelopeMiddleware[any](), Config: typedhttp.MiddlewareConfig{Priority: 60}},
        
        // Audit logging (Priority: 10)
        {Middleware: &AuditMiddleware[any, any]{}, Config: typedhttp.MiddlewareConfig{Priority: 10}},
    }
}
```

### 2. Internal Service

**Purpose**: High-performance internal communication

**Middleware Stack**:
```go
func createInternalServiceMiddleware() []typedhttp.MiddlewareEntry {
    return []typedhttp.MiddlewareEntry{
        // Minimal request tracking (Priority: 100)
        {Middleware: &RequestTrackingMiddleware{}, Config: typedhttp.MiddlewareConfig{Priority: 100}},
        
        // Simple response format (Priority: 50)
        {Middleware: &SimpleResponseMiddleware[any]{}, Config: typedhttp.MiddlewareConfig{Priority: 50}},
    }
}
```

### 3. Admin API

**Purpose**: Administrative operations with enhanced security

**Middleware Stack**:
```go
func createAdminAPIMiddleware() []typedhttp.MiddlewareEntry {
    return []typedhttp.MiddlewareEntry{
        // Enhanced security (Priority: 100)
        {Middleware: &SecurityHeadersMiddleware{}, Config: typedhttp.MiddlewareConfig{Priority: 100}},
        
        // Admin authentication (Priority: 90)
        {Middleware: &AdminAuthMiddleware{}, Config: typedhttp.MiddlewareConfig{Priority: 90}},
        
        // Comprehensive audit (Priority: 80)
        {Middleware: &AdminAuditMiddleware[any, any]{}, Config: typedhttp.MiddlewareConfig{Priority: 80}},
        
        // Admin response envelope (Priority: 60)
        {Middleware: &AdminResponseEnvelopeMiddleware[any]{}, Config: typedhttp.MiddlewareConfig{Priority: 60}},
    }
}
```

## Best Practices

### 1. Middleware Ordering

Use priority-based ordering for consistent middleware execution:

```go
const (
    PrioritySecurityHeaders = 100  // Highest priority
    PriorityAuthentication  = 90
    PriorityRateLimit      = 80
    PriorityRequestTracking = 70
    PriorityValidation     = 60
    PriorityResponseFormat = 50
    PriorityAudit          = 10   // Lowest priority
)
```

### 2. Error Handling

Implement consistent error handling across middleware:

```go
type MiddlewareError struct {
    Code       string            `json:"code"`
    Message    string            `json:"message"`
    Details    map[string]string `json:"details,omitempty"`
    StatusCode int               `json:"-"`
}

func (e *MiddlewareError) Error() string {
    return e.Message
}
```

### 3. Context Usage

Use context for passing data between middleware layers:

```go
type contextKey string

const (
    RequestIDKey  contextKey = "request_id"
    UserIDKey     contextKey = "user_id"
    TraceIDKey    contextKey = "trace_id"
)

func (m *AuthMiddleware) Before(ctx context.Context, req *interface{}) (context.Context, error) {
    userID := extractUserID(req)
    return context.WithValue(ctx, UserIDKey, userID), nil
}
```

### 4. Performance Considerations

- Use conditional middleware for optional features
- Implement middleware pooling for high-traffic scenarios
- Cache expensive computations
- Profile middleware chains under load

```go
func ConditionalMiddleware(condition func(*http.Request) bool, middleware Middleware) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if condition(r) {
                middleware(next).ServeHTTP(w, r)
            } else {
                next.ServeHTTP(w, r)
            }
        })
    }
}
```

## OpenAPI Documentation Strategies

### 1. Comment-Based Documentation

Use OpenAPI comments for rich field documentation:

```go
type CreateUserRequest struct {
    //openapi:description=User's full name,example=John Doe
    Name string `json:"name" validate:"required,min=2,max=50"`
    
    //openapi:description=User's email address,example=john@example.com  
    Email string `json:"email" validate:"required,email"`
    
    //openapi:description=User's role in the system,example=user
    Role string `json:"role" validate:"required,oneof=admin user"`
}
```

### 2. Comprehensive Service Documentation

Configure detailed OpenAPI metadata:

```go
generator := openapi.NewGenerator(&openapi.Config{
    Info: openapi.Info{
        Title:       "E-commerce API",
        Version:     "1.0.0",
        Description: "Production-ready e-commerce API with comprehensive middleware",
        Contact: &openapi.Contact{
            Name:  "API Team",
            Email: "api@example.com",
        },
        License: &openapi.License{
            Name: "MIT",
            URL:  "https://opensource.org/licenses/MIT", 
        },
    },
    Servers: []openapi.Server{
        {URL: "https://api.example.com/v1", Description: "Production"},
        {URL: "https://staging.example.com/v1", Description: "Staging"},
        {URL: "http://localhost:8080", Description: "Development"},
    },
    Security: map[string]openapi.SecurityScheme{
        "bearerAuth": {
            Type:         "http",
            Scheme:       "bearer", 
            BearerFormat: "JWT",
        },
    },
})
```

### 3. Environment-Specific Documentation

Generate different specs for different environments:

```go
func createEnvironmentConfig(env string) *openapi.Config {
    config := &openapi.Config{
        Info: openapi.Info{
            Title:   fmt.Sprintf("API (%s)", env),
            Version: "1.0.0",
        },
    }
    
    switch env {
    case "production":
        config.Servers = []openapi.Server{
            {URL: "https://api.example.com/v1", Description: "Production"},
        }
    case "staging":
        config.Servers = []openapi.Server{
            {URL: "https://staging.example.com/v1", Description: "Staging"},
        }
    case "development":
        config.Servers = []openapi.Server{
            {URL: "http://localhost:8080", Description: "Development"},
        }
    }
    
    return config
}
```

## Example Implementation

See the complete examples in:

- [`examples/comprehensive-architecture/`](../examples/comprehensive-architecture/) - Full e-commerce API with layered middleware
- [`examples/microservice-patterns/`](../examples/microservice-patterns/) - Different patterns for different service types
- [`examples/envelope-middleware/`](../examples/envelope-middleware/) - Response envelope demonstration

## Performance Benchmarks

| Middleware Stack | Latency Overhead | Memory Overhead | Recommended Use Case |
|------------------|------------------|-----------------|---------------------|
| Minimal (1-2 middleware) | < 1ms | < 1KB | Internal services |
| Standard (3-5 middleware) | 1-3ms | 2-5KB | Public APIs |
| Comprehensive (5+ middleware) | 3-8ms | 5-10KB | Admin/audit APIs |

## Conclusion

TypedHTTP's middleware system provides the flexibility to build APIs that match your exact requirements while maintaining type safety and generating accurate documentation. The key is choosing the right middleware strategy for each service type and implementing it consistently across your architecture.

The response schema modification system ensures that your OpenAPI documentation always reflects the actual API responses your clients receive, regardless of how many middleware layers transform the data.