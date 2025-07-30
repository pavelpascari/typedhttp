# TypedHTTP

> Type-safe HTTP handlers for Go with multi-source request data extraction

[![Go Reference](https://pkg.go.dev/badge/github.com/pavelpascari/typedhttp.svg)](https://pkg.go.dev/github.com/pavelpascari/typedhttp)
[![Go Report Card](https://goreportcard.com/badge/github.com/pavelpascari/typedhttp)](https://goreportcard.com/report/github.com/pavelpascari/typedhttp)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

TypedHTTP is a powerful Go library that brings type safety and declarative request handling to HTTP APIs. Extract data from multiple HTTP sources (path, query, headers, cookies, forms, JSON) with configurable precedence rules, transformations, and validation.

## 🚀 Key Features

- **🔒 Type Safety**: Leverage Go generics for compile-time type checking
- **🎯 Multi-Source Extraction**: Get data from path, query, headers, cookies, forms, and JSON body
- **⚡ Precedence Rules**: Define fallback order when data exists in multiple sources  
- **🔄 Transformations**: Built-in data transformations (IP extraction, case conversion, etc.)
- **✅ Validation**: Seamless integration with `go-playground/validator`
- **📁 File Uploads**: First-class support for multipart form uploads
- **🎨 Clean APIs**: Declarative struct tags for ergonomic request definition
- **🔧 Extensible**: Custom decoders, encoders, and middleware support

## 📦 Installation

```bash
go get github.com/pavelpascari/typedhttp
```

## 🎯 Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "net/http"
    "github.com/pavelpascari/typedhttp/pkg/typedhttp"
)

// Define your request structure with typed fields
type GetUserRequest struct {
    ID   string `path:"id" validate:"required,uuid"`
    Page int    `query:"page" default:"1" validate:"min=1"`
}

type GetUserResponse struct {
    ID   string `json:"id"`
    Name string `json:"name"`
    Page int    `json:"page"`
}

// Implement your business logic
type UserHandler struct{}

func (h *UserHandler) Handle(ctx context.Context, req GetUserRequest) (GetUserResponse, error) {
    return GetUserResponse{
        ID:   req.ID,
        Name: "John Doe",
        Page: req.Page,
    }, nil
}

func main() {
    router := typedhttp.NewRouter()
    
    // Register type-safe handlers
    typedhttp.GET(router, "/users/{id}", &UserHandler{})
    
    http.ListenAndServe(":8080", router)
}
```

## 🎨 Multi-Source Data Extraction

The real power of TypedHTTP lies in its ability to extract data from multiple HTTP sources with intelligent precedence rules:

### Authentication & Headers

```go
type APIRequest struct {
    // Multi-source authentication - header takes precedence
    UserID    string `header:"X-User-ID" cookie:"user_id" precedence:"header,cookie"`
    AuthToken string `header:"Authorization" cookie:"auth_token" precedence:"header,cookie"`
    
    // Client information with transformations
    ClientIP  net.IP `header:"X-Forwarded-For" transform:"first_ip"`
    UserAgent string `header:"User-Agent"`
    
    // Language preference - cookie overrides header
    Language  string `cookie:"lang" header:"Accept-Language" default:"en" precedence:"cookie,header"`
}
```

### Complex Request Handling

```go
type ComplexAPIRequest struct {
    // Path parameters
    ResourceID string `path:"id" validate:"required,uuid"`
    Action     string `path:"action" validate:"required,oneof=view edit delete"`
    
    // Query parameters with defaults and validation
    Page   int      `query:"page" default:"1" validate:"min=1"`
    Limit  int      `query:"limit" default:"20" validate:"min=1,max=100"`
    Fields []string `query:"fields" transform:"comma_split"`
    
    // Headers with transformations
    TraceID   string    `header:"X-Trace-ID" query:"trace_id" precedence:"header,query"`
    RequestID string    `header:"X-Request-ID" default:"generate_uuid"`
    Timestamp time.Time `header:"X-Timestamp" format:"rfc3339" default:"now"`
    
    // Form data (for POST/PUT requests)
    Name        string                `form:"name" json:"name" precedence:"form,json"`
    Email       string                `form:"email" json:"email" validate:"email" precedence:"form,json"`
    Avatar      *multipart.FileHeader `form:"avatar"`
    
    // JSON body for complex data
    Metadata map[string]interface{} `json:"metadata"`
    Settings UserSettings            `json:"settings"`
    
    // Cookies for session management
    SessionID string `cookie:"session_id" validate:"required"`
    Theme     string `cookie:"theme" default:"light"`
}

type UserSettings struct {
    Notifications bool   `json:"notifications"`
    Privacy       string `json:"privacy"`
}
```

## 📋 Supported Data Sources

| Source | Tag | Example | Description |
|--------|-----|---------|-------------|
| **Path** | `path:"name"` | `UserID string `path:"id"`` | URL path parameters |
| **Query** | `query:"name"` | `Page int `query:"page"`` | URL query parameters |
| **Headers** | `header:"name"` | `Auth string `header:"Authorization"`` | HTTP headers |
| **Cookies** | `cookie:"name"` | `Session string `cookie:"session_id"`` | HTTP cookies |
| **Form** | `form:"name"` | `Name string `form:"name"`` | Form data (URL-encoded/multipart) |
| **JSON** | `json:"name"` | `Data map[string]interface{} `json:"data"`` | JSON request body |

## 🔧 Advanced Features

### Precedence Rules

Control the order in which sources are checked:

```go
type Request struct {
    // Check header first, fallback to cookie, then query
    UserID string `header:"X-User-ID" cookie:"user_id" query:"user_id" precedence:"header,cookie,query"`
    
    // Cookie takes precedence over header
    Language string `cookie:"lang" header:"Accept-Language" precedence:"cookie,header"`
}
```

### Data Transformations

Built-in transformations for common use cases:

```go
type Request struct {
    ClientIP  net.IP `header:"X-Forwarded-For" transform:"first_ip"`        // Extract first IP from list
    Username  string `header:"X-Username" transform:"to_lower"`             // Convert to lowercase
    IsAdmin   bool   `header:"X-User-Role" transform:"is_admin"`            // Check if role is "admin"
    Trimmed   string `query:"text" transform:"trim_space"`                  // Remove leading/trailing spaces
}
```

### Custom Formats

Parse data with custom formats:

```go
type Request struct {
    CreatedAt   time.Time `header:"X-Created-At" format:"rfc3339"`
    Birthday    time.Time `query:"birthday" format:"2006-01-02"`
    UnixTime    time.Time `header:"X-Timestamp" format:"unix"`
    CustomDate  time.Time `query:"date" format:"02/01/2006"`
}
```

### Default Values

Provide sensible defaults:

```go
type Request struct {
    Page     int    `query:"page" default:"1"`
    Limit    int    `query:"limit" default:"20"`
    Sort     string `query:"sort" default:"created_at"`
    Language string `header:"Accept-Language" default:"en"`
    Theme    string `cookie:"theme" default:"light"`
    
    // Special defaults
    RequestID string    `header:"X-Request-ID" default:"generate_uuid"`
    Timestamp time.Time `header:"X-Timestamp" default:"now"`
}
```

### File Uploads

Handle file uploads seamlessly:

```go
type UploadRequest struct {
    Name        string                  `form:"name" validate:"required"`
    Description string                  `form:"description"`
    Avatar      *multipart.FileHeader   `form:"avatar"`                    // Single file
    Documents   []*multipart.FileHeader `form:"documents"`                 // Multiple files
}

func (h *UploadHandler) Handle(ctx context.Context, req UploadRequest) (UploadResponse, error) {
    if req.Avatar != nil {
        fmt.Printf("Uploaded file: %s (%d bytes)\n", req.Avatar.Filename, req.Avatar.Size)
        
        // Process the file
        file, err := req.Avatar.Open()
        if err != nil {
            return UploadResponse{}, err
        }
        defer file.Close()
        
        // Save or process the file content...
    }
    
    return UploadResponse{Message: "Upload successful"}, nil
}
```

## 🔒 Validation

Leverage `go-playground/validator` for robust validation:

```go
type CreateUserRequest struct {
    Name     string `json:"name" validate:"required,min=2,max=50"`
    Email    string `json:"email" validate:"required,email"`
    Age      int    `json:"age" validate:"required,min=18,max=120"`
    Website  string `json:"website" validate:"omitempty,url"`
    UserID   string `path:"id" validate:"required,uuid"`
    APIKey   string `header:"X-API-Key" validate:"required,len=32"`
}
```

## 🛠️ Error Handling

TypedHTTP provides structured error handling:

```go
func (h *UserHandler) Handle(ctx context.Context, req GetUserRequest) (GetUserResponse, error) {
    // Validation errors are automatically handled and return 400 Bad Request
    // Business logic errors can return custom error types
    
    if req.ID == "invalid" {
        return GetUserResponse{}, typedhttp.NewNotFoundError("User not found")
    }
    
    if !hasPermission(req.UserID) {
        return GetUserResponse{}, typedhttp.NewForbiddenError("Access denied")
    }
    
    return GetUserResponse{ID: req.ID}, nil
}
```

## 📊 Real-World Example

Here's a comprehensive example showing multiple features:

```go
type OrderRequest struct {
    // Path parameters
    OrderID string `path:"id" validate:"required,uuid"`
    
    // Authentication (header preferred, cookie fallback)
    UserID string `header:"X-User-ID" cookie:"user_id" validate:"required" precedence:"header,cookie"`
    
    // Pagination with defaults
    Page  int `query:"page" default:"1" validate:"min=1"`
    Limit int `query:"limit" default:"20" validate:"min=1,max=100"`
    
    // Client info with transformations
    ClientIP  net.IP `header:"X-Forwarded-For" transform:"first_ip"`
    UserAgent string `header:"User-Agent"`
    
    // Preferences (cookie preferred over header)
    Language string `cookie:"lang" header:"Accept-Language" default:"en" precedence:"cookie,header"`
    Currency string `query:"currency" cookie:"currency" default:"USD" precedence:"query,cookie"`
    
    // Form data for updates
    Status      string `form:"status" json:"status" validate:"oneof=pending confirmed cancelled" precedence:"form,json"`
    Notes       string `form:"notes" json:"notes"`
    Attachments []*multipart.FileHeader `form:"attachments"`
    
    // Metadata from JSON body
    CustomFields map[string]interface{} `json:"custom_fields"`
    
    // Tracing
    TraceID   string `header:"X-Trace-ID" query:"trace_id" precedence:"header,query"`
    RequestID string `header:"X-Request-ID" default:"generate_uuid"`
}

type OrderHandler struct {
    orderService OrderService
}

func (h *OrderHandler) Handle(ctx context.Context, req OrderRequest) (OrderResponse, error) {
    log.Printf("Processing order %s for user %s from IP %s", 
        req.OrderID, req.UserID, req.ClientIP)
    
    order, err := h.orderService.GetOrder(ctx, req.OrderID, req.UserID)
    if err != nil {
        return OrderResponse{}, typedhttp.NewNotFoundError("Order not found")
    }
    
    // Handle file attachments if present
    if len(req.Attachments) > 0 {
        for _, attachment := range req.Attachments {
            log.Printf("Processing attachment: %s (%d bytes)", 
                attachment.Filename, attachment.Size)
        }
    }
    
    return OrderResponse{
        ID:       order.ID,
        Status:   order.Status,
        Language: req.Language,
        Currency: req.Currency,
    }, nil
}

// Register the handler
func main() {
    router := typedhttp.NewRouter()
    typedhttp.PUT(router, "/orders/{id}", &OrderHandler{})
    
    log.Println("Server starting on :8080")
    http.ListenAndServe(":8080", router)
}
```

## 🧪 Testing

TypedHTTP makes testing easy with structured requests:

```go
func TestOrderHandler(t *testing.T) {
    handler := &OrderHandler{orderService: mockOrderService}
    
    req := OrderRequest{
        OrderID:   "123e4567-e89b-12d3-a456-426614174000",
        UserID:    "user123",
        Page:      1,
        Limit:     20,
        Language:  "en",
        Currency:  "USD",
        Status:    "confirmed",
        ClientIP:  net.ParseIP("192.168.1.1"),
        RequestID: "req-123",
    }
    
    resp, err := handler.Handle(context.Background(), req)
    assert.NoError(t, err)
    assert.Equal(t, "123e4567-e89b-12d3-a456-426614174000", resp.ID)
}
```

## 🏗️ Architecture

TypedHTTP follows hexagonal architecture principles:

- **Handlers**: Pure business logic, no HTTP concerns
- **Decoders**: Extract and validate request data
- **Encoders**: Format response data
- **Middleware**: Cross-cutting concerns (logging, auth, etc.)
- **Error Mappers**: Convert business errors to HTTP responses

## 📚 Documentation

- [Architecture Decision Records (ADRs)](docs/adrs/)
- [API Reference](https://pkg.go.dev/github.com/pavelpascari/typedhttp)
- [Examples](examples/)

## 🤝 Contributing

Contributions are always welcome! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Inspired by modern web frameworks and Go's type system
- Built with ❤️ for the Go community

---

**Ready to build type-safe HTTP APIs?** Get started with TypedHTTP today! 🚀
