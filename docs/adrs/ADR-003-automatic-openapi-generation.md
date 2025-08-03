# ADR-003: Automatic OpenAPI Specification Generation

## Status

**Accepted** - Implemented ✅
**Implementation Date**: July 2025

### Implementation Summary

The Enhanced Hybrid Approach with comment-based documentation has been successfully implemented:

✅ **Core OpenAPI Generator** - Complete OpenAPI 3.0.3 specification generation from TypedHTTP routers
✅ **Comment-Based Documentation** - Clean `//openapi:` comment syntax for field documentation  
✅ **Multi-Source Parameter Detection** - Automatic extraction from path, query, header, cookie, form, and JSON tags
✅ **Validation Integration** - Schema constraints generated from validation tags
✅ **File Upload Support** - Proper multipart/form-data documentation for file uploads
✅ **Multiple Output Formats** - JSON and YAML specification generation
✅ **TDD Implementation** - Comprehensive test suite with 7 test cases covering all functionality
✅ **Working Example** - Complete demonstration with HTTP server and OpenAPI endpoints

**Key Features Delivered:**
- Automatic parameter extraction from struct tags with precedence rules
- Comment-based OpenAPI metadata (`//openapi:description=...,example=...`)
- Support for complex request types (multipart forms, file uploads, nested objects)
- Validation constraint mapping to OpenAPI schema properties
- Clean separation between data extraction (struct tags) and documentation (comments)
- Production-ready generator with proper error handling and edge case coverage

## Executive Summary

This ADR proposes a comprehensive system for automatically generating OpenAPI 3.0+ specifications from TypedHTTP handlers and request/response types. The goal is to provide accurate, up-to-date API documentation without requiring manual specification writing, while maintaining Go's principles of simplicity, explicit behavior, and type safety.

## Context

Our TypedHTTP library now provides rich type information through:
- Multi-source request data annotations (ADR-002)
- Typed handlers with generic request/response types
- Validation rules integration
- Path parameter extraction
- Complex data transformations and formats

However, we currently lack automatic API documentation generation, which means:

### Current Pain Points
1. **Manual Documentation**: Developers must manually write and maintain OpenAPI specs
2. **Documentation Drift**: API docs become stale as code evolves
3. **Duplicate Information**: Request validation rules and types are defined in both code and specs
4. **Inconsistent Documentation**: Different teams document APIs differently
5. **Development Overhead**: Significant effort required to maintain accurate API documentation

### Success Criteria
1. **Zero Manual Maintenance**: OpenAPI specs generated automatically from code
2. **Rich Type Information**: Leverage our multi-source annotations for accurate parameter documentation
3. **Validation Integration**: Extract validation rules for request/response schemas
4. **Go Idiomatic**: Use familiar Go patterns (struct tags, interfaces, etc.)
5. **Extensible**: Allow customization without breaking core functionality
6. **Performance**: Generation should not impact runtime performance
7. **Standard Compliance**: Generate valid OpenAPI 3.0+ specifications
8. **Multiple Output Formats**: Support JSON, YAML, and potentially others

## Decision

We will implement a **comprehensive OpenAPI generation system** that leverages our existing type information and multi-source annotations to automatically generate accurate, up-to-date API specifications.

## Analysis of Options

### Option 1: Runtime Reflection-Based Generation

```go
type APIHandler struct{}

func (h *APIHandler) Handle(ctx context.Context, req GetUserRequest) (GetUserResponse, error) {
    // Business logic
    return GetUserResponse{}, nil
}

// Generate spec at runtime
func GenerateOpenAPI(router *TypedRouter) *openapi3.T {
    spec := &openapi3.T{
        OpenAPI: "3.0.3",
        Info: &openapi3.Info{
            Title:   "My API",
            Version: "1.0.0",
        },
    }
    
    // Analyze registered handlers using reflection
    for _, handler := range router.GetHandlers() {
        // Extract path, method, request/response types
        // Generate parameter schemas from struct tags
        // Build operation documentation
    }
    
    return spec
}
```

**Pros:**
- Simple implementation
- No build-time tooling required
- Dynamic spec generation
- Can include runtime information

**Cons:**
- Runtime performance impact
- Limited to reflection capabilities
- Harder to customize complex documentation
- Cannot include code comments or advanced metadata

### Option 2: Build-Time Code Generation

```go
//go:generate typedhttp-openapi-gen -package main -output api_spec.go

type GetUserRequest struct {
    ID     string `path:"id" validate:"required,uuid" doc:"User unique identifier"`
    Fields string `query:"fields" default:"id,name,email" doc:"Comma-separated list of fields to return"`
}

// Generated code:
func init() {
    RegisterOpenAPISpec(GetUserRequestSpec)
}

var GetUserRequestSpec = openapi3.Schema{
    Type: "object",
    Properties: map[string]*openapi3.SchemaRef{
        "id": {
            Value: &openapi3.Schema{
                Type:        "string",
                Format:      "uuid",
                Description: "User unique identifier",
            },
        },
        // ... more properties
    },
}
```

**Pros:**
- Zero runtime performance impact
- Can analyze source code and comments
- More powerful customization options
- Can generate multiple output formats

**Cons:**
- Requires build-time tooling
- More complex setup
- Needs to parse Go source code
- Additional build step required

### Option 3: Hybrid Approach with Annotations

```go
// OpenAPI metadata via struct tags and interfaces
type GetUserRequest struct {
    ID     string `path:"id" validate:"required,uuid" openapi:"description=User ID,example=123e4567-e89b-12d3-a456-426614174000"`
    Fields string `query:"fields" default:"id,name,email" openapi:"description=Fields to return,example=id,name"`
} 

// Handler with OpenAPI metadata
type UserHandler struct{}

func (h *UserHandler) Handle(ctx context.Context, req GetUserRequest) (GetUserResponse, error) {
    return GetUserResponse{}, nil
}

// Optional OpenAPI metadata interface
func (h *UserHandler) OpenAPIOperation() OpenAPIOperation {
    return OpenAPIOperation{
        Summary:     "Get user by ID",
        Description: "Retrieves a user's information by their unique identifier",
        Tags:        []string{"users"},
        Responses: map[string]OpenAPIResponse{
            "200": {Description: "User found"},
            "404": {Description: "User not found"},
        },
    }
}
```

**Pros:**
- Flexible runtime generation with rich metadata
- Leverages existing struct tag patterns
- Optional detailed customization
- Can include examples and descriptions
- No external tooling required

**Cons:**
- Some runtime overhead for generation
- Tags can become verbose
- Limited access to source code comments

### Option 4: Interface-Based Documentation

```go
type DocumentedHandler[TReq, TResp any] interface {
    Handler[TReq, TResp]
    OpenAPIOperation() OperationSpec
}

type GetUserHandler struct{}

func (h *GetUserHandler) Handle(ctx context.Context, req GetUserRequest) (GetUserResponse, error) {
    return GetUserResponse{}, nil
}

func (h *GetUserHandler) OpenAPIOperation() OperationSpec {
    return OperationSpec{
        Summary:     "Get user by ID",
        Description: "Retrieves a user by their unique identifier",
        Tags:        []string{"users"},
        Parameters: []ParameterSpec{
            {Name: "id", In: "path", Required: true, Schema: StringSchema{Format: "uuid"}},
            {Name: "fields", In: "query", Schema: StringSchema{Default: "id,name,email"}},
        },
        Responses: map[string]ResponseSpec{
            "200": {Description: "Success", Content: SchemaOf[GetUserResponse]()},
            "404": {Description: "User not found", Content: ErrorSchema()},
        },
    }
}
```

**Pros:**
- Very explicit and type-safe
- Complete control over documentation
- Easy to test and validate
- Clear separation of concerns

**Cons:**
- Requires implementing interface for every handler
- More boilerplate code
- Potential for documentation to drift from implementation
- No automatic inference from types

## Recommended Solution: Enhanced Hybrid Approach with Comment-Based Documentation

After analyzing the trade-offs, we recommend an **Enhanced Hybrid Approach** that combines the best aspects of multiple options with **comment-based OpenAPI documentation** to solve struct tag verbosity:

### Core Design Principles

1. **Automatic Inference**: Extract as much information as possible from existing type annotations
2. **Comment-Based Enhancement**: Use Go comments with `//openapi:` prefix for rich documentation
3. **Clean Struct Tags**: Keep struct tags focused on data extraction, not documentation
4. **Zero Runtime Impact**: Generate specs at application startup, cache results
5. **Go Idiomatic**: Use familiar patterns (comments, struct tags, interfaces)
6. **Extensible**: Support custom documentation without breaking core functionality

### Key Innovation: Comment-Based Documentation

Instead of cluttering struct tags with verbose OpenAPI metadata:
```go
// ❌ Verbose struct tags
File *multipart.FileHeader `form:"file" openapi:"description=File to upload,type=file,format=binary"`
```

We use clean comment-based documentation:
```go
// ✅ Clean separation of concerns
//openapi:description=File to upload,type=file,format=binary
File *multipart.FileHeader `form:"file"`
```

**Benefits:**
- **Clean Tags**: Struct tags remain focused on data extraction
- **Rich Documentation**: Comments can contain extensive metadata
- **Editor Friendly**: Better syntax highlighting and formatting
- **Source Control**: Easier to diff and review documentation changes
- **Maintainable**: Documentation is close to the field it describes

### Proposed Implementation

#### Phase 1: Automatic Schema Generation

```go
type GetUserRequest struct {
    // Automatic parameter detection from multi-source tags
    ID     string `path:"id" validate:"required,uuid"`
    Page   int    `query:"page" default:"1" validate:"min=1,max=100"`
    Limit  int    `query:"limit" default:"20" validate:"min=1,max=1000"`
    
    // Headers automatically documented
    Auth   string `header:"Authorization" validate:"required"`
    
    // Multi-source with precedence automatically documented
    UserID string `header:"X-User-ID" cookie:"user_id" precedence:"header,cookie"`
}

type GetUserResponse struct {
    ID    string `json:"id" validate:"required,uuid"`
    Name  string `json:"name" validate:"required"`
    Email string `json:"email,omitempty" validate:"omitempty,email"`
}

// Automatic generation extracts:
// - Path parameters from `path:` tags
// - Query parameters from `query:` tags with defaults and validation
// - Headers from `header:` tags
// - Request/response schemas from JSON tags and validation rules
// - Example values from `default:` tags
```

#### Phase 2: Enhanced Documentation via Comments

```go
type CreateUserRequest struct {
    //openapi:description=User full name,example=John Doe
    Name  string `json:"name" validate:"required,min=2,max=50"`
    
    //openapi:description=User email address,example=john@example.com
    Email string `json:"email" validate:"required,email"`
    
    //openapi:description=User age in years,example=25
    Age   int    `json:"age" validate:"required,min=18,max=120"`
    
    //openapi:description=User profile picture,type=file,format=binary
    Avatar *multipart.FileHeader `form:"avatar"`
}
```

#### Phase 3: Handler-Level Documentation

```go
type UserHandler struct{}

func (h *UserHandler) Handle(ctx context.Context, req GetUserRequest) (GetUserResponse, error) {
    return GetUserResponse{}, nil
}

// Optional interface for detailed operation documentation
func (h *UserHandler) OpenAPIOperation() OpenAPIOperation {
    return OpenAPIOperation{
        Summary:     "Get user by ID",
        Description: "Retrieves a user's detailed information using their unique identifier",
        Tags:        []string{"users", "public"},
        Security:    []SecurityRequirement{{"bearerAuth": {}}},
        Responses: map[string]OpenAPIResponse{
            "200": {Description: "User retrieved successfully"},
            "401": {Description: "Authentication required"},
            "404": {Description: "User not found"},
            "500": {Description: "Internal server error"},
        },
        Examples: map[string]OpenAPIExample{
            "success": {
                Summary: "Successful user retrieval",
                Value: GetUserResponse{
                    ID:   "123e4567-e89b-12d3-a456-426614174000",
                    Name: "John Doe",
                    Email: "john@example.com",
                },
            },
        },
    }
}
```

#### Phase 4: Application-Level Configuration

```go
func main() {
    router := typedhttp.NewRouter()
    
    // Register handlers
    typedhttp.GET(router, "/users/{id}", &UserHandler{})
    typedhttp.POST(router, "/users", &CreateUserHandler{})
    
    // Generate OpenAPI spec
    specGenerator := openapi.NewGenerator(openapi.Config{
        Info: openapi.Info{
            Title:       "User Management API",
            Version:     "1.0.0",
            Description: "API for managing users in the system",
        },
        Servers: []openapi.Server{
            {URL: "https://api.example.com", Description: "Production"},
            {URL: "https://staging-api.example.com", Description: "Staging"},
        },
        Security: []openapi.SecurityScheme{
            "bearerAuth": {
                Type: "http",
                Scheme: "bearer",
                BearerFormat: "JWT",
            },
        },
    })
    
    spec, err := specGenerator.Generate(router)
    if err != nil {
        log.Fatal(err)
    }
    
    // Serve OpenAPI spec
    http.Handle("/openapi.json", openapi.JSONHandler(spec))
    http.Handle("/openapi.yaml", openapi.YAMLHandler(spec))
    http.Handle("/docs", swagger.UIHandler(spec))
    
    http.ListenAndServe(":8080", router)
}
```

## Detailed Design

### Schema Generation Strategy

#### Automatic Parameter Detection
```go
// From this request type:
type SearchRequest struct {
    // Path parameter
    Category string `path:"category" validate:"required,oneof=users posts comments"`
    
    // Query parameters with validation and defaults
    Query  string `query:"q" validate:"required,min=1"`
    Page   int    `query:"page" default:"1" validate:"min=1,max=1000"`
    Limit  int    `query:"limit" default:"20" validate:"min=1,max=100"`
    Sort   string `query:"sort" default:"created_at" validate:"oneof=created_at updated_at name"`
    
    // Headers
    Auth   string `header:"Authorization" validate:"required,prefix=Bearer "`
    
    // Multi-source (documented with precedence)
    UserID string `header:"X-User-ID" cookie:"user_id" precedence:"header,cookie"`
}

// Generate this OpenAPI parameter documentation:
parameters:
  - name: category
    in: path
    required: true
    schema:
      type: string
      enum: [users, posts, comments]
  - name: q
    in: query
    required: true
    schema:
      type: string
      minLength: 1
  - name: page
    in: query
    schema:
      type: integer
      default: 1
      minimum: 1
      maximum: 1000
  - name: Authorization
    in: header
    required: true
    schema:
      type: string
      pattern: "^Bearer .+"
  - name: X-User-ID
    in: header
    description: "Primary source for user ID (fallback: user_id cookie)"
    schema:
      type: string
  - name: user_id
    in: cookie
    description: "Fallback source for user ID"
    schema:
      type: string
```

#### Request/Response Schema Generation
```go
// From this response type:
type UserResponse struct {
    ID        string    `json:"id" validate:"required,uuid"`
    Name      string    `json:"name" validate:"required,min=2,max=50"`
    Email     string    `json:"email,omitempty" validate:"omitempty,email"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at,omitempty"`
    Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Generate this schema:
UserResponse:
  type: object
  required: [id, name, created_at]
  properties:
    id:
      type: string
      format: uuid
    name:
      type: string
      minLength: 2
      maxLength: 50
    email:
      type: string
      format: email
    created_at:
      type: string
      format: date-time
    updated_at:
      type: string
      format: date-time
    metadata:
      type: object
      additionalProperties: true
```

### Enhanced Documentation via Comments

```go
type OpenAPIFieldMetadata struct {
    Description string            // Field description
    Example     interface{}       // Example value
    Deprecated  bool             // Mark as deprecated
    Extensions  map[string]interface{} // Custom extensions
}

// Usage examples with comment-based documentation:
type ProductRequest struct {
    //openapi:description=Product name,example=iPhone 13 Pro
    Name     string  `json:"name" validate:"required"`
    
    //openapi:description=Product price in USD,example=999.99
    Price    float64 `json:"price" validate:"required,min=0"`
    
    //openapi:description=Product category,example=electronics,deprecated=false
    Category string  `json:"category" validate:"required"`
}
```

### File Upload Documentation

```go
type UploadRequest struct {
    //openapi:description=File name
    Name        string                  `form:"name" validate:"required"`
    
    //openapi:description=File description
    Description string                  `form:"description"`
    
    //openapi:description=File to upload,type=file,format=binary
    File        *multipart.FileHeader   `form:"file"`
    
    //openapi:description=Optional thumbnail,type=file,format=binary
    Thumbnail   *multipart.FileHeader   `form:"thumbnail"`
    
    //openapi:description=Additional documents,type=array,items.type=file
    Documents   []*multipart.FileHeader `form:"documents"`
}

// Generated OpenAPI:
requestBody:
  content:
    multipart/form-data:
      schema:
        type: object
        properties:
          name:
            type: string
            description: "File name"
          description:
            type: string
            description: "File description"
          file:
            type: string
            format: binary
            description: "File to upload"
          thumbnail:
            type: string
            format: binary
            description: "Optional thumbnail"
          documents:
            type: array
            items:
              type: string
              format: binary
            description: "Additional documents"
```

### Error Response Documentation

```go
// Leverage existing error types for automatic error documentation
type ValidationError struct {
    Message string            `json:"message"`
    Fields  map[string]string `json:"fields"`
}

type NotFoundError struct {
    Message string `json:"message"`
    Code    string `json:"code"`
}

// Automatic generation of error responses:
responses:
  "400":
    description: "Validation error"
    content:
      application/json:
        schema:
          $ref: "#/components/schemas/ValidationError"
  "404":
    description: "Resource not found"
    content:
      application/json:
        schema:
          $ref: "#/components/schemas/NotFoundError"
```

## Implementation Architecture

### Core Components

```go
// OpenAPI generator interface
type Generator interface {
    Generate(router *TypedRouter) (*openapi3.T, error)
    GenerateJSON() ([]byte, error)
    GenerateYAML() ([]byte, error)
}

// Schema analyzer for extracting type information
type SchemaAnalyzer interface {
    AnalyzeRequest(requestType reflect.Type) (*openapi3.SchemaRef, []openapi3.Parameter, error)
    AnalyzeResponse(responseType reflect.Type) (*openapi3.SchemaRef, error)
    AnalyzeValidation(tag string) (*openapi3.Schema, error)
}

// Operation documentation extractor
type OperationExtractor interface {
    ExtractOperation(handler interface{}) (*openapi3.Operation, error)
    ExtractTags(handler interface{}) []string
    ExtractSecurity(handler interface{}) []openapi3.SecurityRequirement
}
```

### Generator Implementation

```go
type DefaultGenerator struct {
    config          Config
    schemaAnalyzer  SchemaAnalyzer
    operationExtractor OperationExtractor
    customComponents map[string]*openapi3.SchemaRef
}

func (g *DefaultGenerator) Generate(router *TypedRouter) (*openapi3.T, error) {
    spec := &openapi3.T{
        OpenAPI: "3.0.3",
        Info:    &g.config.Info,
        Servers: g.config.Servers,
        Components: &openapi3.Components{
            Schemas:         make(map[string]*openapi3.SchemaRef),
            SecuritySchemes: g.config.Security,
        },
        Paths: make(map[string]*openapi3.PathItem),
    }
    
    // Process each registered handler
    for _, registration := range router.GetHandlers() {
        pathItem, err := g.processHandler(registration)
        if err != nil {
            return nil, fmt.Errorf("failed to process handler %s %s: %w", 
                registration.Method, registration.Path, err)
        }
        
        // Add to spec
        spec.Paths[registration.Path] = pathItem
    }
    
    return spec, nil
}

func (g *DefaultGenerator) processHandler(reg HandlerRegistration) (*openapi3.PathItem, error) {
    // Extract request parameters and schema
    requestSchema, parameters, err := g.schemaAnalyzer.AnalyzeRequest(reg.RequestType)
    if err != nil {
        return nil, err
    }
    
    // Extract response schema
    responseSchema, err := g.schemaAnalyzer.AnalyzeResponse(reg.ResponseType)
    if err != nil {
        return nil, err
    }
    
    // Build operation
    operation := &openapi3.Operation{
        Parameters: parameters,
        Responses: map[string]*openapi3.ResponseRef{
            "200": {
                Value: &openapi3.Response{
                    Description: "Success",
                    Content: map[string]*openapi3.MediaType{
                        "application/json": {
                            Schema: responseSchema,
                        },
                    },
                },
            },
        },
    }
    
    // Add request body if needed
    if needsRequestBody(reg.RequestType) {
        operation.RequestBody = &openapi3.RequestBodyRef{
            Value: &openapi3.RequestBody{
                Content: map[string]*openapi3.MediaType{
                    "application/json": {
                        Schema: requestSchema,
                    },
                },
            },
        }
    }
    
    // Create path item
    pathItem := &openapi3.PathItem{}
    switch reg.Method {
    case "GET":
        pathItem.Get = operation
    case "POST":
        pathItem.Post = operation
    case "PUT":
        pathItem.Put = operation
    case "PATCH":
        pathItem.Patch = operation
    case "DELETE":
        pathItem.Delete = operation
    }
    
    return pathItem, nil
}
```

## Usage Examples

### Basic Usage

```go
func main() {
    router := typedhttp.NewRouter()
    
    // Register handlers (existing code unchanged)
    typedhttp.GET(router, "/users/{id}", &GetUserHandler{})
    typedhttp.POST(router, "/users", &CreateUserHandler{})
    
    // Generate OpenAPI spec
    generator := openapi.NewGenerator(openapi.Config{
        Info: openapi.Info{
            Title:   "User API",
            Version: "1.0.0",
        },
    })
    
    spec, _ := generator.Generate(router)
    
    // Serve documentation
    http.Handle("/openapi.json", openapi.JSONHandler(spec))
    http.Handle("/docs", swagger.UIHandler(spec))
    http.ListenAndServe(":8080", router)
}
```

### Advanced Configuration

```go
func main() {
    config := openapi.Config{
        Info: openapi.Info{
            Title:          "Advanced API",
            Version:        "2.1.0",
            Description:    "Comprehensive API with full documentation",
            TermsOfService: "https://example.com/terms",
            Contact: &openapi.Contact{
                Name:  "API Support",
                URL:   "https://example.com/support",
                Email: "support@example.com",
            },
            License: &openapi.License{
                Name: "MIT",
                URL:  "https://opensource.org/licenses/MIT",
            },
        },
        Servers: []openapi.Server{
            {URL: "https://api.example.com/v2", Description: "Production"},
            {URL: "https://staging.example.com/v2", Description: "Staging"},
        },
        Security: map[string]openapi.SecurityScheme{
            "bearerAuth": {
                Type:         "http",
                Scheme:       "bearer",
                BearerFormat: "JWT",
            },
            "apiKey": {
                Type: "apiKey",
                In:   "header",
                Name: "X-API-Key",
            },
        },
        Tags: []openapi.Tag{
            {Name: "users", Description: "User management operations"},
            {Name: "posts", Description: "Blog post operations"},
        },
        ExternalDocs: &openapi.ExternalDocs{
            Description: "Full API Documentation",
            URL:         "https://docs.example.com",
        },
    }
    
    generator := openapi.NewGenerator(config)
    spec, _ := generator.Generate(router)
    
    // Multiple output formats
    http.Handle("/openapi.json", openapi.JSONHandler(spec))
    http.Handle("/openapi.yaml", openapi.YAMLHandler(spec))
    http.Handle("/docs", swagger.UIHandler(spec))
    http.Handle("/redoc", redoc.UIHandler(spec))
}
```

### Custom Documentation Interface

```go
type DocumentedHandler interface {
    OpenAPIOperation() OpenAPIOperation
}

type CreateUserHandler struct{}

func (h *CreateUserHandler) Handle(ctx context.Context, req CreateUserRequest) (CreateUserResponse, error) {
    // Implementation
    return CreateUserResponse{}, nil
}

func (h *CreateUserHandler) OpenAPIOperation() OpenAPIOperation {
    return OpenAPIOperation{
        Summary:     "Create a new user",
        Description: "Creates a new user account with the provided information",
        Tags:        []string{"users"},
        Security:    []SecurityRequirement{{"bearerAuth": {}}},
        Responses: map[string]OpenAPIResponse{
            "201": {Description: "User created successfully"},
            "400": {Description: "Invalid request data"},  
            "409": {Description: "User already exists"},
        },
        Examples: map[string]OpenAPIExample{
            "basic_user": {
                Summary: "Basic user creation",
                Value: CreateUserRequest{
                    Name:  "John Doe",
                    Email: "john@example.com",
                    Age:   30,
                },
            },
        },
    }
}
```

## Implementation Plan

### Phase 1: Core Generation (Week 1-2)
- [ ] Implement basic schema analyzer for request/response types
- [ ] Create parameter extraction from multi-source annotations
- [ ] Build basic OpenAPI spec generation
- [ ] Add JSON/YAML output formats

### Phase 2: Enhanced Features (Week 3-4)
- [ ] Add OpenAPI tag support for enhanced documentation
- [ ] Implement custom operation documentation interface
- [ ] Add file upload and multipart form documentation
- [ ] Create Swagger UI integration

### Phase 3: Advanced Capabilities (Week 5-6)
- [ ] Add validation rule extraction to schemas
- [ ] Implement error response documentation
- [ ] Add custom component registration
- [ ] Create comprehensive test suite

### Phase 4: Developer Experience (Week 7-8)
- [ ] Add CLI tool for spec generation
- [ ] Create development middleware for live spec updates
- [ ] Add example generation and testing utilities
- [ ] Write comprehensive documentation

## Benefits

1. **Zero Maintenance Documentation**: Specs stay in sync with code automatically
2. **Rich Type Information**: Leverage multi-source annotations for complete API docs
3. **Developer Productivity**: Eliminate manual spec writing and maintenance
4. **API Consistency**: Standardized documentation across all endpoints
5. **Integration Ready**: Easy integration with existing OpenAPI tools
6. **Go Idiomatic**: Uses familiar Go patterns and conventions

## Considerations

### Performance Impact
- **Generation Strategy**: Generate specs at startup, cache in memory
- **Lazy Loading**: Generate specs on-demand for development mode
- **Optimization**: Pre-compute type analysis to minimize reflection overhead

### Customization vs Automation
- **Smart Defaults**: Provide sensible defaults extracted from types
- **Optional Enhancement**: Allow detailed customization without requiring it
- **Incremental Adoption**: Works with minimal annotations, better with more

### Testing Strategy
- **Schema Validation**: Ensure generated schemas are valid OpenAPI 3.0+
- **Round-trip Testing**: Verify generated specs match actual API behavior
- **Example Generation**: Automatic generation of realistic examples
- **Integration Testing**: Test with popular OpenAPI tools

## Open Questions

1. **Code Comments**: Should we parse Go source code to extract documentation from comments?
2. **Versioning**: How should we handle API versioning in generated specs?
3. **Custom Types**: What's the best approach for documenting custom types beyond JSON Schema?
4. **Large APIs**: How do we handle performance for APIs with hundreds of endpoints?
5. **Streaming**: How should we document streaming endpoints and WebSocket connections?

## Conclusion

The Enhanced Hybrid Approach provides the best balance of automation, customization, and Go idioms. By leveraging our existing multi-source annotation system, we can generate comprehensive, accurate OpenAPI specifications with minimal developer effort while maintaining the flexibility to add detailed documentation where needed.

This approach positions TypedHTTP as a complete solution for building well-documented, type-safe HTTP APIs in Go.

---

**Next Steps**: Review this ADR with the team and prioritize implementation phases. The automatic OpenAPI generation will significantly enhance the developer experience and API documentation quality for TypedHTTP users.