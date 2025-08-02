# Architecture Decision Records Index

## Active ADRs

| ADR | Title | Status | Date | Summary |
|-----|-------|--------|------|---------|
| [ADR-001](./adrs/ADR-001-typed-http-handlers.md) | Typed HTTP Handlers | ✅ **Implemented** | 2024 | Core typed handler architecture with Go generics |
| [ADR-002](./adrs/ADR-002-request-data-source-annotations.md) | Request Data Source Annotations | ✅ **Implemented** | Jan 2025 | Multi-source request data extraction with precedence rules |
| [ADR-003](./adrs/ADR-003-automatic-openapi-generation.md) | Automatic OpenAPI Generation | ✅ **Implemented** | Jan 2025 | Automatic OpenAPI 3.0+ spec generation with comment-based documentation |
| [ADR-004](./adrs/ADR-004-test-utility-package.md) | TypedHTTP Test Utility Package | ✅ **Implemented** | Jan 2025 | Comprehensive test utilities for end-to-end handler testing with fluent APIs |
| [ADR-005](./adrs/ADR-005-comprehensive-middleware-patterns.md) | Comprehensive Middleware Patterns | 📋 **Proposed** | Jan 2025 | Advanced middleware system with typed middleware, composition utilities, and standard implementations |
| [ADR-006](./adrs/ADR-006-middleware-response-schema-modification.md) | Middleware Response Schema Modification | ✅ **Implemented** | Jan 2025 | Enable middleware to modify OpenAPI response schemas for accurate documentation |
| [ADR-007](./adrs/ADR-007-boilerplate-reduction-router-composition.md) | Boilerplate Reduction and Router Composition | ✅ **Implemented** | Aug 2025 | 52% code reduction via Resource pattern + team-based router composition for 50+ engineer organizations |

## Implementation Status

### ✅ Completed Features
- **Type-safe HTTP handlers** with Go generics
- **Multi-source data extraction** from path, query, headers, cookies, forms, JSON
- **Precedence rules** for intelligent data source fallback
- **Data transformations** and custom format parsing
- **File upload support** with multipart forms
- **Validation integration** with go-playground/validator
- **Comprehensive test coverage** and working examples
- **OpenAPI automatic generation** with comment-based documentation
- **JSON and YAML spec output** with HTTP server endpoints
- **5/5 Go-idiomatic test utilities** with context support, explicit error handling, and zero boilerplate
- **Middleware response schema modification** for accurate OpenAPI documentation with envelope middleware
- **Resource pattern for boilerplate reduction** - 52% less code with single Resource() calls replacing multiple handler wrappers
- **Router composition system** for large teams - ComposableRouter with mounting, middleware inheritance, and team isolation

### 🚧 Future Enhancements
- **Comprehensive middleware system** (ADR-005 proposed) - Advanced middleware patterns with type safety
- Client code generation from OpenAPI specs

## Quick Reference

### Basic Usage
```go
type GetUserRequest struct {
    ID   string `path:"id" validate:"required"`
    Page int    `query:"page" default:"1"`
}
```

### Multi-Source with Precedence
```go
type APIRequest struct {
    UserID string `header:"X-User-ID" cookie:"user_id" precedence:"header,cookie"`
    Token  string `header:"Authorization" validate:"required"`
}
```
