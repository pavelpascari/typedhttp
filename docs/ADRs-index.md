# Architecture Decision Records Index

## Active ADRs

| ADR | Title | Status | Date | Summary |
|-----|-------|--------|------|---------|
| [ADR-001](./adrs/ADR-001-typed-http-handlers.md) | Typed HTTP Handlers | ✅ **Implemented** | 2024 | Core typed handler architecture with Go generics |
| [ADR-002](./adrs/ADR-002-request-data-source-annotations.md) | Request Data Source Annotations | ✅ **Implemented** | Jan 2025 | Multi-source request data extraction with precedence rules |
| [ADR-003](./adrs/ADR-003-automatic-openapi-generation.md) | Automatic OpenAPI Generation | ✅ **Implemented** | Jan 2025 | Automatic OpenAPI 3.0+ spec generation with comment-based documentation |

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

### 🚧 Future Enhancements
- Advanced middleware system (planned future ADR)
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
