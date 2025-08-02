# TypedHTTP Examples & Architecture Guide

This directory contains comprehensive examples demonstrating how to architect production-ready applications using TypedHTTP with advanced middleware patterns and automatic OpenAPI generation.

## 📚 Documentation

### Core Guides
- [**Architecture Guide**](../docs/architecture-guide.md) - Comprehensive guide to TypedHTTP architecture patterns
- [**Middleware Best Practices**](../docs/middleware-best-practices.md) - Best practices for middleware composition and design
- [**ADR Index**](../docs/ADRs-index.md) - All architectural decision records

### ADRs (Architecture Decision Records)
- [ADR-001: Typed HTTP Handlers](../docs/adrs/ADR-001-typed-http-handlers.md)
- [ADR-002: Request Data Source Annotations](../docs/adrs/ADR-002-request-data-source-annotations.md)
- [ADR-003: Automatic OpenAPI Generation](../docs/adrs/ADR-003-automatic-openapi-generation.md)
- [ADR-004: Test Utility Package](../docs/adrs/ADR-004-test-utility-package.md)
- [ADR-005: Comprehensive Middleware Patterns](../docs/adrs/ADR-005-comprehensive-middleware-patterns.md)
- [ADR-006: Middleware Response Schema Modification](../docs/adrs/ADR-006-middleware-response-schema-modification.md)

## 🏗️ Architecture Examples

### 1. [Comprehensive Architecture](./comprehensive-architecture/)
**Purpose**: Full-featured e-commerce API demonstrating enterprise-grade patterns

**Features**:
- ✅ Layered middleware architecture (Security → Auth → Validation → Business Logic → Response → Observability)
- ✅ Multiple response transformations (Cache metadata + Response envelope)
- ✅ Comprehensive OpenAPI generation with accurate schema modification
- ✅ Real-world domain models (Users, Products, Orders)
- ✅ Advanced validation with custom error types
- ✅ Production-ready patterns

**Run Example**:
```bash
cd comprehensive-architecture
go mod tidy
go run main.go
# Visit http://localhost:8080 for interactive demo
```

### 2. [Microservice Patterns](./microservice-patterns/)
**Purpose**: Different middleware strategies for different service types

**Features**:
- ✅ Public API Gateway pattern (Full security stack)
- ✅ Internal Service pattern (Minimal overhead)
- ✅ Admin API pattern (Enhanced security + audit)
- ✅ Health Check pattern (Ultra-minimal)
- ✅ Environment-specific configurations

**Run Example**:
```bash
cd microservice-patterns
go mod tidy
go run main.go
# Visit http://localhost:8080 for patterns demo
```

### 3. [Envelope Middleware](./envelope-middleware/)
**Purpose**: Response envelope middleware with OpenAPI schema modification

**Features**:
- ✅ Response envelope wrapping all API responses
- ✅ Automatic OpenAPI schema transformation
- ✅ Request ID and timestamp injection
- ✅ Error response standardization

**Run Example**:
```bash
cd envelope-middleware
go mod tidy
go run main.go
# Visit http://localhost:8080 for envelope demo
```

## 🎯 Quick Start Guide

### 1. Basic TypedHTTP Setup

```go
package main

import (
    "context"
    "github.com/pavelpascari/typedhttp/pkg/typedhttp"
    "github.com/pavelpascari/typedhttp/pkg/openapi"
)

// Define request/response types
type GetUserRequest struct {
    ID string `path:"id" validate:"required,uuid"`
}

type GetUserResponse struct {
    User User `json:"user"`
}

// Implement handler
func GetUser(ctx context.Context, req GetUserRequest) (GetUserResponse, error) {
    // Your business logic here
    return GetUserResponse{User: user}, nil
}

func main() {
    router := typedhttp.NewRouter()
    
    // Register handler
    typedhttp.GET(router, "/users/{id}", GetUser)
    
    // Generate OpenAPI docs
    generator := openapi.NewGenerator(&openapi.Config{
        Info: openapi.Info{Title: "My API", Version: "1.0.0"},
    })
    spec, _ := generator.Generate(router)
    
    // Serve
    http.ListenAndServe(":8080", router)
}
```

### 2. Adding Middleware

```go
// Create middleware stack
middleware := []typedhttp.MiddlewareEntry{
    {
        Middleware: typedhttp.NewResponseEnvelopeMiddleware[any](),
        Config: typedhttp.MiddlewareConfig{
            Name:     "envelope",
            Priority: 90,
        },
    },
}

// Apply to all handlers
handlers := router.GetHandlers()
for i := range handlers {
    handlers[i].MiddlewareEntries = middleware
}
```

### 3. Rich OpenAPI Documentation

```go
type CreateUserRequest struct {
    //openapi:description=User's full name,example=John Doe
    Name string `json:"name" validate:"required,min=2,max=50"`
    
    //openapi:description=User's email address,example=john@example.com
    Email string `json:"email" validate:"required,email"`
}
```

## 🏛️ Architecture Patterns

### Layered Middleware Architecture

```
┌─────────────────────────────────────────┐
│            HTTP Transport               │
├─────────────────────────────────────────┤
│ Layer 1: Security & Rate Limiting      │ Priority: 100-90
│   • CORS, Security Headers             │
│   • Rate Limiting, DDoS Protection     │
├─────────────────────────────────────────┤
│ Layer 2: Authentication & Authorization│ Priority: 89-80
│   • JWT Validation, API Keys           │
│   • Role-based Access Control          │
├─────────────────────────────────────────┤
│ Layer 3: Request Processing            │ Priority: 79-70
│   • Validation, Transformation         │
│   • Request Enrichment                 │
├─────────────────────────────────────────┤
│ Layer 4: Business Logic                │ Priority: N/A
│   • Your Handler Functions             │
├─────────────────────────────────────────┤
│ Layer 5: Response Processing           │ Priority: 69-50
│   • Response Transformation            │
│   • Caching, Envelope Wrapping         │
├─────────────────────────────────────────┤
│ Layer 6: Observability                 │ Priority: 49-10
│   • Metrics, Logging, Tracing          │
│   • Audit Trails                       │
└─────────────────────────────────────────┘
```

### Response Schema Modification Flow

```
Handler Response → Middleware Chain → Final Client Response
     ↓                    ↓                    ↓
  User{...}         Cache Wrapper         Envelope Wrapper
                         ↓                    ↓
               CachedResponse{        APIResponse{
                 Data: User{...}       Data: CachedResponse{...}
                 CachedAt: ...          Success: true
                 ExpiresAt: ...         Meta: {...}
               }                      }
```

### Service Type Patterns

| Service Type | Middleware Stack | Use Case |
|-------------|------------------|----------|
| **Public API** | Security + Auth + Validation + Envelope + Audit | Customer-facing APIs |
| **Internal Service** | Minimal tracking + Simple response | High-performance internal communication |
| **Admin API** | Enhanced security + Admin auth + Comprehensive audit | Administrative operations |
| **Health Check** | Minimal tracking only | Service health monitoring |

## 🔧 Middleware Types

### 1. Standard HTTP Middleware
```go
func LoggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler
```
- Operates at HTTP transport layer
- Cross-cutting concerns (CORS, compression, etc.)

### 2. Typed Pre-Middleware  
```go
type TypedPreMiddleware[TRequest any] interface {
    Before(ctx context.Context, req *TRequest) (context.Context, error)
}
```
- Operates on decoded request data
- Validation, authentication, request enrichment

### 3. Typed Post-Middleware
```go
type TypedPostMiddleware[TResponse any] interface {
    After(ctx context.Context, resp *TResponse) (*TResponse, error)
}
```
- Operates on response data
- Response transformation, caching, formatting

### 4. Full Lifecycle Middleware
```go
type TypedMiddleware[TRequest, TResponse any] interface {
    Before(ctx context.Context, req *TRequest) (context.Context, error)
    After(ctx context.Context, req *TRequest, resp *TResponse, err error) (*TResponse, error)
}
```
- Full request/response lifecycle
- Auditing, metrics, complex correlation

### 5. Schema-Aware Middleware
```go
type ResponseSchemaModifier interface {
    ModifyResponseSchema(ctx context.Context, originalSchema *openapi3.SchemaRef) (*openapi3.SchemaRef, error)
}
```
- Modifies OpenAPI response schemas
- Ensures documentation accuracy

## 📊 Performance Characteristics

| Pattern | Latency Overhead | Memory Overhead | Best For |
|---------|------------------|-----------------|----------|
| No Middleware | 0ms | 0KB | Testing only |
| Minimal (1-2) | < 1ms | < 1KB | Internal services |
| Standard (3-5) | 1-3ms | 2-5KB | Public APIs |
| Comprehensive (6+) | 3-8ms | 5-10KB | Admin/audit APIs |

## 🎨 Design Principles

1. **Type Safety First** - Compile-time guarantees for all request/response handling
2. **Composable Architecture** - Mix and match middleware based on requirements  
3. **Accurate Documentation** - OpenAPI specs always reflect actual API behavior
4. **Performance Optimization** - Minimal overhead where needed, full features where required
5. **Developer Experience** - Intuitive APIs with comprehensive tooling

## 🚀 Getting Started

1. **Clone and explore examples**:
   ```bash
   git clone <repo>
   cd typedhttp/examples
   ```

2. **Run comprehensive example**:
   ```bash
   cd comprehensive-architecture
   go mod tidy
   go run main.go
   open http://localhost:8080
   ```

3. **Review architecture guides**:
   - Start with [Architecture Guide](../docs/architecture-guide.md)
   - Read [Middleware Best Practices](../docs/middleware-best-practices.md)
   - Explore specific ADRs for detailed decisions

4. **Adapt patterns for your needs**:
   - Copy middleware patterns that fit your requirements
   - Modify service architectures for your domain
   - Extend OpenAPI documentation for your APIs

## 📈 Next Steps

After exploring these examples, you'll be able to:

- ✅ Design layered middleware architectures for any application
- ✅ Implement response schema modification for accurate documentation
- ✅ Choose the right middleware patterns for different service types
- ✅ Generate comprehensive OpenAPI documentation automatically
- ✅ Build production-ready APIs with type safety and performance

For questions or contributions, see the main [TypedHTTP repository](../).

---

**Happy Building! 🎉**