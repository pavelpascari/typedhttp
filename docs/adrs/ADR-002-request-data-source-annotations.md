# ADR-002: Request Data Source Annotations

## Status

**Proposed** - Under Review

## Executive Summary

This ADR proposes a comprehensive annotation system for TypedHTTP request types to declaratively specify where request data should be extracted from (path parameters, query parameters, headers, cookies, body, etc.). The goal is to provide an ergonomic, type-safe way to collect and populate request data from multiple HTTP sources into a single strongly-typed Go struct.

## Context

Our current `CombinedDecoder` implementation attempts to merge data from multiple sources (path, query, JSON body) but lacks a clear, standardized way to specify which fields should come from which sources. The current approach has several limitations:

### Current State
```go
type GetUserRequest struct {
    ID   string `path:"id" validate:"required"`           // Path parameter
    Name string `query:"name"`                             // Query parameter  
}

type CreateUserRequest struct {
    Name  string `json:"name" validate:"required"`        // JSON body
    Email string `json:"email" validate:"required"`       // JSON body
}
```

### Problems with Current Approach
1. **Inconsistent Annotations**: Different tag names for different sources (`path:`, `query:`, `json:`)
2. **Limited Source Support**: No support for headers, cookies, form data, etc.
3. **Ambiguous Precedence**: Unclear what happens when multiple sources could provide the same field
4. **No Multi-Source Fields**: Can't easily extract the same logical field from multiple potential sources
5. **Complex Validation**: Validation logic is scattered across different decoders

### Success Criteria
1. **Ergonomic API**: Easy to read and write request type annotations
2. **Comprehensive Source Support**: Headers, cookies, path, query, body, form data, etc.
3. **Clear Precedence Rules**: Well-defined behavior when multiple sources are available
4. **Type Safety**: Maintain compile-time type checking
5. **Validation Integration**: Seamless integration with validation rules
6. **Backward Compatibility**: Existing code should continue to work
7. **Performance**: Minimal runtime overhead, efficient field mapping

## Decision

We will implement a **unified annotation system** using Go struct tags to declaratively specify request data sources with clear precedence rules and comprehensive source support.

## Proposed Solutions

### Option 1: Single Unified Tag with Source Specification

```go
type GetUserRequest struct {
    ID          string `http:"path:id" validate:"required"`
    Name        string `http:"query:name"`
    ContentType string `http:"header:Content-Type"`
    SessionID   string `http:"cookie:session_id"`
    
    // Body fields (default behavior)
    Description string `json:"description"`
}

type AdvancedRequest struct {
    // Multi-source with precedence (path takes precedence over query)
    UserID string `http:"path:id,query:user_id" validate:"required"`
    
    // Header with default value
    UserAgent string `http:"header:User-Agent" default:"unknown"`
    
    // Form data from multipart/form-data
    File *multipart.FileHeader `http:"form:upload_file"`
    
    // Custom extraction with transformation
    Timestamp time.Time `http:"header:X-Timestamp" format:"unix"`
}
```

**Pros:**
- Single tag system is consistent and easy to remember
- Clear source specification
- Supports multi-source with precedence
- Extensible for new sources

**Cons:**
- Breaks compatibility with existing `json:`, `query:` tags
- More verbose than current approach
- Custom parsing logic needed

### Option 2: Multiple Source-Specific Tags (Current + Extensions)

```go
type GetUserRequest struct {
    ID          string `path:"id" validate:"required"`
    Name        string `query:"name"`
    ContentType string `header:"Content-Type"`
    SessionID   string `cookie:"session_id"`
    Description string `json:"description"`
    
    // Multi-source support with precedence annotations
    UserID string `path:"id" query:"user_id" precedence:"path,query"`
}

type FormRequest struct {
    Name     string                `form:"name" validate:"required"`
    Email    string                `form:"email" validate:"email"`
    Avatar   *multipart.FileHeader `form:"avatar"`
    Category string                `form:"category" json:"category"` // Fallback to JSON
}
```

**Pros:**
- Maintains backward compatibility
- Familiar to Go developers (similar to existing `json:` tags)
- Each source has dedicated, clear tags
- Easy to add new source types

**Cons:**
- Multiple tags can be verbose
- Precedence rules need separate annotation
- Risk of conflicting tag names

### Option 3: Structured Tag Format

```go
type UnifiedRequest struct {
    // Simple cases
    ID   string `source:"path=id"`
    Name string `source:"query=name"`
    
    // Complex cases with options
    UserAgent string `source:"header=User-Agent,default=unknown"`
    
    // Multi-source with precedence
    UserID string `source:"path=id|query=user_id|header=X-User-ID"`
    
    // Transformation options
    CreatedAt time.Time `source:"header=X-Created-At,format=rfc3339"`
    
    // Body fields (implicit)
    Email string `json:"email" validate:"email"`
}
```

**Pros:**
- Powerful and flexible
- Single tag system
- Supports complex scenarios
- Clear syntax for multi-source

**Cons:**
- Custom parsing logic required
- More complex syntax to learn
- Potential parsing errors

### Option 4: Method-Based Approach (Alternative Pattern)

```go
type RequestBuilder struct {
    ID          string
    Name        string
    ContentType string
    Body        CreateUserBody
}

// Define extraction rules separately
func (r *RequestBuilder) ExtractFrom() []FieldExtractor {
    return []FieldExtractor{
        PathParam("id").Into(&r.ID),
        QueryParam("name").Into(&r.Name),
        Header("Content-Type").Into(&r.ContentType),
        JSONBody().Into(&r.Body),
    }
}
```

**Pros:**
- Very explicit and type-safe
- Compile-time checking of field assignments
- Highly flexible for complex scenarios

**Cons:**
- Much more verbose
- Breaks from idiomatic Go struct tag pattern
- Requires more boilerplate code

## Recommended Solution: Option 2 (Enhanced Multi-Tag System)

After analyzing the trade-offs, **Option 2** provides the best balance of ergonomics, compatibility, and Go idioms.

### Enhanced Multi-Tag System Design

```go
type ComprehensiveRequest struct {
    // === Path Parameters ===
    UserID   string `path:"id" validate:"required"`
    Category string `path:"category"`
    
    // === Query Parameters ===
    Limit    int    `query:"limit" default:"10" validate:"min=1,max=100"`
    Offset   int    `query:"offset" default:"0" validate:"min=0"`
    Sort     string `query:"sort" default:"created_at"`
    
    // === Headers ===
    UserAgent     string `header:"User-Agent"`
    Authorization string `header:"Authorization" validate:"required"`
    ContentType   string `header:"Content-Type"`
    AcceptLang    string `header:"Accept-Language" default:"en"`
    
    // === Cookies ===
    SessionID  string `cookie:"session_id"`
    CSRF       string `cookie:"csrf_token"`
    
    // === Form Data (multipart/form-data or application/x-www-form-urlencoded) ===
    Name   string                `form:"name" validate:"required"`
    Email  string                `form:"email" validate:"email"`
    Avatar *multipart.FileHeader `form:"avatar"`
    
    // === JSON Body ===
    Metadata map[string]interface{} `json:"metadata"`
    Settings UserSettings            `json:"settings"`
    
    // === Multi-Source Fields (with precedence) ===
    TraceID string `header:"X-Trace-ID" query:"trace_id" precedence:"header,query"`
    
    // === Custom Transformations ===
    Timestamp time.Time `header:"X-Timestamp" format:"unix"`
    IPAddress net.IP    `header:"X-Forwarded-For" transform:"first_ip"`
}
```

### Tag Specifications

#### Core Source Tags
- `path:"param_name"` - Extract from URL path parameters
- `query:"param_name"` - Extract from URL query parameters  
- `header:"Header-Name"` - Extract from HTTP headers (case-insensitive)
- `cookie:"cookie_name"` - Extract from HTTP cookies
- `form:"field_name"` - Extract from form data (multipart or urlencoded)
- `json:"field_name"` - Extract from JSON request body

#### Enhancement Tags
- `default:"value"` - Default value if source is empty/missing
- `validate:"rules"` - Validation rules (existing validator integration)
- `precedence:"source1,source2,source3"` - Order of precedence for multi-source fields
- `format:"layout"` - Custom format for time/date parsing (e.g., "unix", "rfc3339")
- `transform:"function"` - Custom transformation function name
- `required` - Alternative to `validate:"required"`

#### Special Behaviors
- **Multi-Source Support**: Fields can have multiple source tags
- **Precedence Rules**: Explicit precedence via `precedence:` tag or implicit (path > header > query > form > json)
- **Type Coercion**: Automatic conversion between compatible types (string to int, string to time.Time, etc.)
- **Validation Integration**: Seamless integration with go-playground/validator

### Implementation Architecture

```go
// Enhanced field extraction metadata
type FieldSource struct {
    Type      SourceType          // path, query, header, cookie, form, json
    Name      string              // parameter/field name in source
    Default   string              // default value
    Format    string              // format specification  
    Transform string              // transformation function name
    Required  bool                // required field
}

type FieldExtractor struct {
    FieldName   string              // Go struct field name
    FieldType   reflect.Type        // Go field type
    Sources     []FieldSource       // All possible sources
    Precedence  []SourceType        // Order of precedence
    Validation  string              // validation rules
}

// Enhanced CombinedDecoder with comprehensive source support
type CombinedDecoder[T any] struct {
    extractors    []FieldExtractor    // Pre-computed field extractors
    pathDecoder   *PathDecoder[T]
    queryDecoder  *QueryDecoder[T]
    headerDecoder *HeaderDecoder[T]   // New
    cookieDecoder *CookieDecoder[T]   // New
    formDecoder   *FormDecoder[T]     // New
    jsonDecoder   *JSONDecoder[T]
    validator     *validator.Validate
    transformers  map[string]TransformFunc // Custom transformers
}
```

### Example Usage Patterns

#### Simple REST API Endpoint
```go
type GetUserRequest struct {
    ID     string `path:"id" validate:"required,uuid"`
    Fields string `query:"fields" default:"id,name,email"`
}

// GET /users/{id}?fields=id,name,email
```

#### Complex Multi-Source Request
```go
type ComplexRequest struct {
    // Authentication
    UserID string `header:"X-User-ID" cookie:"user_id" validate:"required"`
    Token  string `header:"Authorization" validate:"required,prefix=Bearer "`
    
    // Pagination
    Page  int `query:"page" default:"1" validate:"min=1"`
    Limit int `query:"limit" default:"20" validate:"min=1,max=100"`
    
    // Filtering (can come from query or JSON body)
    Filters map[string]string `query:"filter" json:"filters" precedence:"json,query"`
    
    // File upload
    Document *multipart.FileHeader `form:"document" validate:"required"`
    
    // Metadata
    ClientIP  net.IP    `header:"X-Forwarded-For" transform:"first_ip"`
    Timestamp time.Time `header:"X-Timestamp" format:"unix" default:"now"`
}
```

#### Backward Compatibility
```go
// Existing code continues to work unchanged
type LegacyRequest struct {
    Name  string `json:"name" validate:"required"`
    Email string `json:"email" validate:"email"`
}

// New enhanced version
type EnhancedRequest struct {
    Name  string `json:"name" form:"name" validate:"required"`
    Email string `json:"email" form:"email" validate:"email"`
    
    // New fields with enhanced annotations
    Source string `header:"X-Source" default:"web"`
}
```

### Error Handling and Validation

```go
type ExtractionError struct {
    Field   string      `json:"field"`
    Source  string      `json:"source"`
    Value   string      `json:"value"`
    Error   string      `json:"error"`
}

type ValidationResult struct {
    Extracted map[string]interface{} `json:"extracted"`
    Errors    []ExtractionError      `json:"errors"`
    Warnings  []string               `json:"warnings"`
}
```

### Custom Transformers

```go
// Built-in transformers
var BuiltinTransformers = map[string]TransformFunc{
    "first_ip":    extractFirstIP,
    "to_lower":    strings.ToLower,
    "to_upper":    strings.ToUpper,
    "trim_space":  strings.TrimSpace,
    "parse_json":  parseJSONString,
}

// Custom transformer registration
decoder.RegisterTransformer("custom_transform", func(value string) (interface{}, error) {
    // Custom transformation logic
    return transformedValue, nil
})
```

## Implementation Plan

### Phase 1: Core Multi-Source Support (Week 1-2)
- [ ] Implement `HeaderDecoder[T]` and `CookieDecoder[T]`
- [ ] Enhance `CombinedDecoder` with multi-source support
- [ ] Add precedence rule handling
- [ ] Implement basic default value support

### Phase 2: Advanced Features (Week 3-4)
- [ ] Add `FormDecoder[T]` for multipart/form-data
- [ ] Implement custom format support (time parsing, etc.)
- [ ] Add custom transformer system
- [ ] Enhance error reporting and validation

### Phase 3: Optimization and Polish (Week 5-6)
- [ ] Performance optimization with field extraction caching
- [ ] Comprehensive test coverage
- [ ] Documentation and examples
- [ ] Backward compatibility testing

## Benefits

1. **Developer Experience**: Single, consistent way to specify data sources
2. **Flexibility**: Supports complex multi-source scenarios
3. **Type Safety**: Maintains compile-time type checking
4. **Validation Integration**: Seamless validation of extracted data
5. **Performance**: Efficient field extraction with minimal overhead
6. **Extensibility**: Easy to add new source types and transformers

## Considerations

### Performance Impact
- **Field Extraction Caching**: Pre-compute extraction rules at startup
- **Efficient Parsing**: Minimize reflection usage during request processing
- **Memory Allocation**: Reuse extraction context objects

### Security Considerations
- **Header Injection**: Validate and sanitize header values
- **Path Traversal**: Validate path parameters
- **Size Limits**: Impose reasonable limits on extracted data size

### Testing Strategy
- **Unit Tests**: Each decoder component independently
- **Integration Tests**: End-to-end request processing
- **Performance Tests**: Benchmark extraction performance
- **Compatibility Tests**: Ensure backward compatibility

## Conclusion

The enhanced multi-tag system provides a comprehensive, ergonomic solution for request data source annotations while maintaining Go idioms and backward compatibility. This approach positions TypedHTTP as a complete solution for complex HTTP request handling scenarios.

## Open Questions

1. Should we support dynamic field extraction based on runtime conditions?
2. How should we handle conflicting validation rules from different sources?
3. What's the best approach for custom type coercion beyond built-in types?
4. Should we provide a code generation tool for complex request types?

---

**Next Steps**: Review this ADR with the team and decide on implementation priorities. The enhanced multi-tag system represents the most practical and Go-idiomatic approach for comprehensive request data handling.