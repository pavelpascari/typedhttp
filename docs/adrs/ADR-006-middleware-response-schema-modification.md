# ADR-006: Middleware Response Schema Modification for OpenAPI Generation

## Status

**Accepted** - Implemented ✅
**Implementation Date**: July 2025

## Executive Summary

This ADR addresses the challenge of generating accurate OpenAPI specifications when middleware transforms response structures. Specifically, it provides a solution for middleware that wraps or modifies handler responses (such as response envelope middleware) while maintaining type safety and ensuring OpenAPI specs reflect the actual response structure clients receive.

## Context

### Current Problem

The existing OpenAPI generation system (ADR-003) assumes that the handler's response type is the final response structure. However, middleware can transform responses, leading to inaccurate OpenAPI specifications.

**Example Scenario:**
```go
// Handler returns User
func GetUser(ctx context.Context, req GetUserRequest) (User, error) {
    return User{ID: "123", Name: "John"}, nil
}

// Envelope middleware wraps response
type APIResponse[T any] struct {
    Data    *T      `json:"data,omitempty"`
    Error   *string `json:"error,omitempty"`
    Success bool    `json:"success"`
}

// Client actually receives APIResponse[User], not User
```

**Current OpenAPI Generation (Incorrect):**
```yaml
responses:
  "200":
    content:
      application/json:
        schema:
          $ref: "#/components/schemas/User"
```

**Desired OpenAPI Generation (Correct):**
```yaml
responses:
  "200":
    content:
      application/json:
        schema:
          $ref: "#/components/schemas/APIResponse_User"
```

### Impact

1. **API Documentation Drift**: OpenAPI specs don't match actual API responses
2. **Client Code Generation**: Generated clients expect wrong response structure
3. **Developer Confusion**: Mismatch between documentation and reality
4. **Testing Issues**: API contract tests fail due to schema mismatches

## Decision

We will implement a **Middleware Response Schema Modification** system that allows middleware to declare how they transform response schemas for OpenAPI generation.

### Core Design Principles

1. **Schema Composition**: Middleware can declare schema transformations
2. **Type Safety**: Maintain compile-time type safety for middleware chains
3. **Composability**: Multiple schema-modifying middleware can be chained
4. **Backward Compatibility**: Existing middleware continues to work unchanged
5. **Opt-in**: Only middleware that implement the interface participate in schema modification

## Detailed Design

### 1. Response Schema Modifier Interface

```go
// ResponseSchemaModifier allows middleware to modify response schemas for OpenAPI generation
type ResponseSchemaModifier interface {
    ModifyResponseSchema(ctx context.Context, originalSchema *openapi3.SchemaRef) (*openapi3.SchemaRef, error)
}

// Combined interface for middleware that both modifies responses and their schemas
type SchemaAwareMiddleware[TResponse any] interface {
    TypedPostMiddleware[TResponse]
    ResponseSchemaModifier
}
```

### 2. Envelope Middleware Implementation

```go
// ResponseEnvelopeMiddleware wraps responses in a standard envelope structure
type ResponseEnvelopeMiddleware[TResponse any] struct {
    includeRequestID bool
    includeTimestamp bool
    includeMeta      bool
}

// Runtime response transformation
func (m *ResponseEnvelopeMiddleware[TResponse]) After(ctx context.Context, resp *TResponse) (*APIResponse[TResponse], error) {
    envelope := &APIResponse[TResponse]{
        Data:    resp,
        Success: true,
    }
    
    if m.includeMeta {
        meta := &ResponseMeta{}
        if m.includeRequestID {
            if requestID := ctx.Value("request_id"); requestID != nil {
                meta.RequestID = requestID.(string)
            }
        }
        if m.includeTimestamp {
            meta.Timestamp = time.Now().Format(time.RFC3339)
        }
        envelope.Meta = meta
    }
    
    return envelope, nil
}

// OpenAPI schema transformation
func (m *ResponseEnvelopeMiddleware[TResponse]) ModifyResponseSchema(ctx context.Context, originalSchema *openapi3.SchemaRef) (*openapi3.SchemaRef, error) {
    envelopeSchema := &openapi3.SchemaRef{
        Value: &openapi3.Schema{
            Type: "object",
            Properties: map[string]*openapi3.SchemaRef{
                "data": originalSchema,
                "error": {
                    Value: &openapi3.Schema{
                        Type:     "string",
                        Nullable: true,
                    },
                },
                "success": {
                    Value: &openapi3.Schema{Type: "boolean"},
                },
            },
            Required: []string{"success"},
        },
    }
    
    if m.includeMeta {
        envelopeSchema.Value.Properties["meta"] = &openapi3.SchemaRef{
            Value: &openapi3.Schema{
                Type: "object",
                Properties: map[string]*openapi3.SchemaRef{
                    "request_id": {
                        Value: &openapi3.Schema{
                            Type: "string",
                        },
                    },
                    "timestamp": {
                        Value: &openapi3.Schema{
                            Type:   "string",
                            Format: "date-time",
                        },
                    },
                },
            },
        }
    }
    
    return envelopeSchema, nil
}

// Standard envelope types
type APIResponse[T any] struct {
    Data    *T            `json:"data,omitempty"`
    Error   *string       `json:"error,omitempty"`
    Success bool          `json:"success"`
    Meta    *ResponseMeta `json:"meta,omitempty"`
}

type ResponseMeta struct {
    RequestID string `json:"request_id,omitempty"`
    Timestamp string `json:"timestamp,omitempty"`
}
```

### 3. Enhanced OpenAPI Generator

```go
// Enhanced generator that applies middleware schema transformations
func (g *DefaultGenerator) processHandler(reg HandlerRegistration) (*openapi3.PathItem, error) {
    // 1. Extract base response schema from handler
    baseResponseSchema, err := g.schemaAnalyzer.AnalyzeResponse(reg.ResponseType)
    if err != nil {
        return nil, fmt.Errorf("failed to analyze response type: %w", err)
    }
    
    // 2. Apply middleware schema transformations
    finalResponseSchema, err := g.applyMiddlewareSchemaTransformations(
        context.Background(),
        reg.MiddlewareEntries,
        baseResponseSchema,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to apply middleware schema transformations: %w", err)
    }
    
    // 3. Build operation with transformed schema
    operation := &openapi3.Operation{
        Parameters: parameters,
        Responses: map[string]*openapi3.ResponseRef{
            "200": {
                Value: &openapi3.Response{
                    Description: "Success",
                    Content: map[string]*openapi3.MediaType{
                        "application/json": {
                            Schema: finalResponseSchema,
                        },
                    },
                },
            },
        },
    }
    
    // 4. Add error responses if envelope middleware is present
    if g.hasEnvelopeMiddleware(reg.MiddlewareEntries) {
        g.addEnvelopeErrorResponses(operation)
    }
    
    return g.buildPathItem(reg.Method, operation), nil
}

func (g *DefaultGenerator) applyMiddlewareSchemaTransformations(
    ctx context.Context,
    entries []MiddlewareEntry,
    baseSchema *openapi3.SchemaRef,
) (*openapi3.SchemaRef, error) {
    currentSchema := baseSchema
    
    // Apply transformations in middleware execution order (post-middleware runs in reverse)
    for i := len(entries) - 1; i >= 0; i-- {
        entry := entries[i]
        if modifier, ok := entry.Middleware.(ResponseSchemaModifier); ok {
            transformedSchema, err := modifier.ModifyResponseSchema(ctx, currentSchema)
            if err != nil {
                return nil, fmt.Errorf("middleware %q failed to modify schema: %w", 
                    entry.Config.Name, err)
            }
            currentSchema = transformedSchema
        }
    }
    
    return currentSchema, nil
}

func (g *DefaultGenerator) hasEnvelopeMiddleware(entries []MiddlewareEntry) bool {
    for _, entry := range entries {
        if _, ok := entry.Middleware.(*ResponseEnvelopeMiddleware[any]); ok {
            return true
        }
    }
    return false
}

func (g *DefaultGenerator) addEnvelopeErrorResponses(operation *openapi3.Operation) {
    // Add standard envelope error responses
    operation.Responses["400"] = &openapi3.ResponseRef{
        Value: &openapi3.Response{
            Description: "Bad Request",
            Content: map[string]*openapi3.MediaType{
                "application/json": {
                    Schema: &openapi3.SchemaRef{
                        Value: &openapi3.Schema{
                            Type: "object",
                            Properties: map[string]*openapi3.SchemaRef{
                                "data": {
                                    Value: &openapi3.Schema{Type: "null"},
                                },
                                "error": {
                                    Value: &openapi3.Schema{Type: "string"},
                                },
                                "success": {
                                    Value: &openapi3.Schema{Type: "boolean", Enum: []interface{}{false}},
                                },
                            },
                            Required: []string{"success", "error"},
                        },
                    },
                },
            },
        },
    }
    
    operation.Responses["500"] = &openapi3.ResponseRef{
        Value: &openapi3.Response{
            Description: "Internal Server Error",
            Content: map[string]*openapi3.MediaType{
                "application/json": {
                    Schema: operation.Responses["400"].Value.Content["application/json"].Schema,
                },
            },
        },
    }
}
```

### 4. Error Handling Middleware Integration

```go
// ErrorHandlingMiddleware that works with envelope middleware
type ErrorHandlingMiddleware[TRequest, TResponse any] struct{}

func (m *ErrorHandlingMiddleware[TRequest, TResponse]) After(
    ctx context.Context, 
    req *TRequest, 
    resp *TResponse, 
    err error,
) (*TResponse, error) {
    if err != nil {
        // If there's envelope middleware, it will wrap this error
        return nil, &APIError{
            Message: err.Error(),
            Code:    "INTERNAL_ERROR",
        }
    }
    return resp, nil
}

type APIError struct {
    Message string `json:"message"`
    Code    string `json:"code"`
}

func (e *APIError) Error() string {
    return e.Message
}
```

## Usage Examples

### 1. Basic Envelope Middleware

```go
func main() {
    router := typedhttp.NewRouter()
    
    // Apply envelope middleware globally
    router.Use(&ResponseEnvelopeMiddleware[any]{
        includeRequestID: true,
        includeTimestamp: true,
        includeMeta:      true,
    })
    
    // Handlers return their natural types
    router.GET("/users/{id}", getUserHandler)
    router.POST("/users", createUserHandler)
    
    // Generate OpenAPI spec with envelope schemas
    generator := openapi.NewGenerator(openapi.Config{
        Info: openapi.Info{
            Title:   "Enveloped API",
            Version: "1.0.0",
        },
    })
    
    spec, _ := generator.Generate(router)
    
    // Serve documentation
    http.Handle("/openapi.json", openapi.JSONHandler(spec))
    http.ListenAndServe(":8080", router)
}
```

### 2. Conditional Envelope Middleware

```go
// Apply envelope only to API routes
apiGroup := router.Group("/api/v1")
apiGroup.Use(&ResponseEnvelopeMiddleware[any]{
    includeRequestID: true,
    includeTimestamp: false,
    includeMeta:      true,
})

// Public routes without envelope
router.GET("/health", healthHandler)         // Returns: {"status": "ok"}
apiGroup.GET("/users", listUsersHandler)     // Returns: {"data": [...], "success": true}
```

### 3. Multiple Schema Transformations

```go
// Custom caching middleware that adds cache metadata
type CacheMetadataMiddleware[TResponse any] struct{}

func (m *CacheMetadataMiddleware[TResponse]) After(ctx context.Context, resp *TResponse) (*CachedResponse[TResponse], error) {
    return &CachedResponse[TResponse]{
        Data:      resp,
        CachedAt:  time.Now(),
        ExpiresAt: time.Now().Add(time.Hour),
    }, nil
}

func (m *CacheMetadataMiddleware[TResponse]) ModifyResponseSchema(ctx context.Context, originalSchema *openapi3.SchemaRef) (*openapi3.SchemaRef, error) {
    return &openapi3.SchemaRef{
        Value: &openapi3.Schema{
            Type: "object",
            Properties: map[string]*openapi3.SchemaRef{
                "data":       originalSchema,
                "cached_at":  {Value: &openapi3.Schema{Type: "string", Format: "date-time"}},
                "expires_at": {Value: &openapi3.Schema{Type: "string", Format: "date-time"}},
            },
            Required: []string{"data", "cached_at", "expires_at"},
        },
    }, nil
}

// Chain multiple transformations
router.GET("/cached-data", dataHandler,
    WithTypedPostMiddleware(&CacheMetadataMiddleware[DataResponse]{}),
    WithTypedPostMiddleware(&ResponseEnvelopeMiddleware[CachedResponse[DataResponse]]{}),
)

// Final response: APIResponse[CachedResponse[DataResponse]]
```

## Implementation Plan

### Phase 1: Core Interface and Basic Implementation (Week 1)
- [ ] Add `ResponseSchemaModifier` interface to middleware package
- [ ] Implement basic envelope middleware with schema modification
- [ ] Add schema transformation support to OpenAPI generator

### Phase 2: Enhanced Features (Week 2)
- [ ] Add error response handling for envelope middleware
- [ ] Implement multiple schema transformation support
- [ ] Add comprehensive error handling and validation

### Phase 3: Testing and Documentation (Week 3)
- [ ] Create comprehensive test suite for schema transformations
- [ ] Add integration tests with complex middleware chains
- [ ] Update documentation and examples

### Phase 4: Advanced Patterns (Week 4)
- [ ] Add middleware composition utilities for common patterns
- [ ] Implement conditional schema transformation
- [ ] Add performance optimization for schema transformation chains

## Benefits

### 1. Accurate API Documentation
- OpenAPI specs reflect actual response structures clients receive
- Generated client code matches real API behavior
- API contract tests work correctly

### 2. Type Safety Maintained
- Compile-time type checking for middleware chains
- Clear interfaces for schema-modifying middleware
- No runtime surprises or type mismatches

### 3. Composable Transformations
- Multiple middleware can transform schemas in sequence
- Clean separation between runtime behavior and documentation
- Reusable middleware components

### 4. Backward Compatibility
- Existing middleware continues to work unchanged
- Opt-in system for schema modification
- No breaking changes to existing APIs

### 5. Developer Experience
- Clear mental model for how middleware affects documentation
- Easy to test and debug schema transformations
- Consistent patterns for common use cases

## Risks and Mitigations

### Risk 1: Schema Transformation Complexity
**Impact**: MEDIUM - Complex middleware chains may produce confusing schemas
**Mitigation**:
- Provide clear documentation and examples
- Add debug utilities to visualize schema transformations
- Limit transformation depth or complexity

### Risk 2: Performance Impact
**Impact**: LOW - Schema transformation during OpenAPI generation may be slow
**Mitigation**:
- Cache transformed schemas
- Optimize schema transformation algorithms
- Make schema generation optional in production

### Risk 3: Type Safety Edge Cases
**Impact**: MEDIUM - Generic middleware may not properly transform schemas
**Mitigation**:
- Comprehensive testing of generic middleware patterns
- Runtime validation of schema transformations
- Clear error messages for invalid transformations

## Alternatives Considered

### 1. Response Type Annotations
**Rejected**: Would require manual annotation of actual response types, defeating the purpose of automatic generation.

### 2. Runtime Schema Detection
**Rejected**: Would require complex runtime analysis and might miss conditional transformations.

### 3. Code Generation Approach
**Rejected**: Would require build-time tooling and break the runtime-only approach of the current system.

## Success Criteria

1. **Accurate Documentation**: OpenAPI specs match actual API responses for all middleware patterns
2. **Type Safety**: No compile-time or runtime type errors in middleware chains
3. **Performance**: Schema transformation adds less than 10ms to OpenAPI generation
4. **Adoption**: Envelope middleware pattern is easily adoptable by developers
5. **Composability**: Multiple schema-transforming middleware work correctly together

## Conclusion

This ADR provides a clean, type-safe solution for handling middleware that transforms response structures. The `ResponseSchemaModifier` interface allows middleware to declare how they modify schemas for OpenAPI generation, ensuring accurate documentation while maintaining type safety and composability.

The envelope middleware pattern serves as an excellent proving ground for this system, addressing a common real-world need while demonstrating the flexibility and power of the schema transformation approach.

## References

- [ADR-003: Automatic OpenAPI Generation](./ADR-003-automatic-openapi-generation.md)
- [ADR-005: Comprehensive Middleware Patterns](./ADR-005-comprehensive-middleware-patterns.md)
- [OpenAPI 3.0 Specification](https://swagger.io/specification/)
- [Go Middleware Patterns](https://www.alexedwards.net/blog/making-and-using-middleware)