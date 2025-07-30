package main

import (
	"context"
	"fmt"
	"mime/multipart"
	"net"
	"time"
)

// === Current Implementation (Limited) ===

type CurrentGetUserRequest struct {
	ID   string `path:"id" validate:"required"` // Path only
	Name string `query:"name"`                  // Query only
}

type CurrentCreateUserRequest struct {
	Name  string `json:"name" validate:"required"`  // JSON body only
	Email string `json:"email" validate:"required"` // JSON body only
}

// === Proposed Enhanced Implementation ===

// Simple case - similar to current but more explicit
type EnhancedGetUserRequest struct {
	ID     string `path:"id" validate:"required"`
	Fields string `query:"fields" default:"id,name,email"`
}

// Complex multi-source request showcasing the full power
type ComplexAPIRequest struct {
	// === Authentication & Authorization ===
	UserID        string `header:"X-User-ID" cookie:"user_id" validate:"required" precedence:"header,cookie"`
	Authorization string `header:"Authorization" validate:"required,prefix=Bearer "`
	SessionToken  string `cookie:"session_token"`

	// === Path Parameters ===
	ResourceID string `path:"id" validate:"required,uuid"`
	Action     string `path:"action" validate:"required,oneof=view edit delete"`

	// === Query Parameters ===
	Page   int      `query:"page" default:"1" validate:"min=1"`
	Limit  int      `query:"limit" default:"20" validate:"min=1,max=100"`
	Sort   string   `query:"sort" default:"created_at"`
	Fields []string `query:"fields" transform:"comma_split"`

	// === Headers for Metadata ===
	UserAgent   string    `header:"User-Agent"`
	ContentType string    `header:"Content-Type"`
	AcceptLang  string    `header:"Accept-Language" default:"en"`
	ClientIP    net.IP    `header:"X-Forwarded-For" transform:"first_ip"`
	RequestTime time.Time `header:"X-Request-Time" format:"rfc3339" default:"now"`
	TraceID     string    `header:"X-Trace-ID" query:"trace_id" precedence:"header,query"`

	// === Form Data (for file uploads or form submissions) ===
	Name        string                `form:"name" json:"name" validate:"required" precedence:"form,json"`
	Email       string                `form:"email" json:"email" validate:"email" precedence:"form,json"`
	Avatar      *multipart.FileHeader `form:"avatar"`
	Description string                `form:"description" json:"description" precedence:"json,form"`

	// === JSON Body (for complex structured data) ===
	Metadata map[string]interface{} `json:"metadata"`
	Settings UserSettings           `json:"settings"`
	Filters  SearchFilters          `json:"filters" query:"filter" precedence:"json,query"`

	// === Cookies (for session management) ===
	Theme    string `cookie:"theme" default:"light"`
	Language string `cookie:"lang" header:"Accept-Language" default:"en" precedence:"cookie,header"`
	CSRF     string `cookie:"csrf_token" header:"X-CSRF-Token" validate:"required" precedence:"header,cookie"`

	// === Computed/Derived Fields ===
	IsAdmin   bool   `header:"X-User-Role" transform:"is_admin"`
	Timezone  string `header:"X-Timezone" cookie:"timezone" default:"UTC" precedence:"header,cookie"`
	RequestID string `header:"X-Request-ID" default:"generate_uuid"`
}

type UserSettings struct {
	Notifications bool   `json:"notifications"`
	Privacy       string `json:"privacy"`
	Theme         string `json:"theme"`
}

type SearchFilters struct {
	Status     string   `json:"status"`
	Categories []string `json:"categories"`
	DateRange  struct {
		From time.Time `json:"from"`
		To   time.Time `json:"to"`
	} `json:"date_range"`
}

// === Handler Implementation ===

type ComplexAPIHandler struct{}

func (h *ComplexAPIHandler) Handle(ctx context.Context, req ComplexAPIRequest) (ComplexAPIResponse, error) {
	// Business logic can now access all the rich data without worrying about HTTP concerns

	fmt.Printf("Processing request for user %s\n", req.UserID)
	fmt.Printf("Resource: %s, Action: %s\n", req.ResourceID, req.Action)
	fmt.Printf("Client IP: %s, User Agent: %s\n", req.ClientIP, req.UserAgent)
	fmt.Printf("Pagination: page=%d, limit=%d\n", req.Page, req.Limit)
	fmt.Printf("Auth method: %s\n", req.Authorization)

	if req.Avatar != nil {
		fmt.Printf("File upload: %s (%d bytes)\n", req.Avatar.Filename, req.Avatar.Size)
	}

	if req.Metadata != nil {
		fmt.Printf("Metadata: %+v\n", req.Metadata)
	}

	return ComplexAPIResponse{
		Status:    "success",
		Message:   "Request processed successfully",
		RequestID: req.RequestID,
		UserID:    req.UserID,
		Data: map[string]interface{}{
			"resource_id": req.ResourceID,
			"action":      req.Action,
			"client_ip":   req.ClientIP.String(),
		},
	}, nil
}

type ComplexAPIResponse struct {
	Status    string                 `json:"status"`
	Message   string                 `json:"message"`
	RequestID string                 `json:"request_id"`
	UserID    string                 `json:"user_id"`
	Data      map[string]interface{} `json:"data"`
}

// === Comparison: What This Enables ===

func demonstrateCapabilities() {
	fmt.Println("=== CURRENT LIMITATIONS ===")
	fmt.Println("❌ Can only get data from one source per field")
	fmt.Println("❌ No headers, cookies, or form data support")
	fmt.Println("❌ No default values or transformations")
	fmt.Println("❌ No multi-source fallback logic")
	fmt.Println("")

	fmt.Println("=== ENHANCED CAPABILITIES ===")
	fmt.Println("✅ Multi-source fields with precedence rules")
	fmt.Println("✅ Headers, cookies, form data, path, query, JSON body")
	fmt.Println("✅ Default values and custom transformations")
	fmt.Println("✅ Rich validation integration")
	fmt.Println("✅ Type safety and compile-time checking")
	fmt.Println("✅ Backward compatibility with existing code")
	fmt.Println("")

	fmt.Println("=== REAL-WORLD USE CASES ===")
	fmt.Println("🔹 Authentication: Token from header OR cookie")
	fmt.Println("🔹 Tracing: Trace ID from header OR query parameter")
	fmt.Println("🔹 User preferences: Language from cookie OR Accept-Language header")
	fmt.Println("🔹 File uploads: Form data + JSON metadata")
	fmt.Println("🔹 Admin APIs: Complex filtering from query OR JSON body")
	fmt.Println("🔹 Mobile APIs: Client info from multiple headers")
	fmt.Println("")

	fmt.Println("=== HTTP REQUEST MAPPING ===")
	fmt.Println("POST /api/v1/resources/{id}/{action}?page=2&limit=50&trace_id=abc123")
	fmt.Println("Headers:")
	fmt.Println("  Authorization: Bearer token123")
	fmt.Println("  X-User-ID: user456")
	fmt.Println("  X-Forwarded-For: 192.168.1.100, 10.0.0.1")
	fmt.Println("  Content-Type: multipart/form-data")
	fmt.Println("Cookies:")
	fmt.Println("  session_token=sess789; theme=dark; lang=es")
	fmt.Println("Body (multipart):")
	fmt.Println("  name=John Doe")
	fmt.Println("  email=john@example.com")
	fmt.Println("  avatar=<binary file data>")
	fmt.Println("  metadata={\"priority\": \"high\"}")
	fmt.Println("")

	fmt.Println("Would be automatically mapped to:")
	fmt.Println("  ResourceID: 'id' (from path)")
	fmt.Println("  Action: 'action' (from path)")
	fmt.Println("  Page: 2 (from query)")
	fmt.Println("  Limit: 50 (from query)")
	fmt.Println("  TraceID: 'abc123' (from query, fallback from header)")
	fmt.Println("  Authorization: 'Bearer token123' (from header)")
	fmt.Println("  UserID: 'user456' (from header)")
	fmt.Println("  ClientIP: 192.168.1.100 (from header, first IP)")
	fmt.Println("  SessionToken: 'sess789' (from cookie)")
	fmt.Println("  Theme: 'dark' (from cookie)")
	fmt.Println("  Language: 'es' (from cookie)")
	fmt.Println("  Name: 'John Doe' (from form)")
	fmt.Println("  Email: 'john@example.com' (from form)")
	fmt.Println("  Avatar: <FileHeader> (from form)")
	fmt.Println("  Metadata: map[priority:high] (from form field as JSON)")
}

// === Implementation Preview ===

// This shows how the enhanced decoder would work internally
type EnhancedFieldExtractor struct {
	FieldName  string
	Sources    []DataSource
	Precedence []string
	Default    string
	Transform  string
	Validate   string
}

type DataSource struct {
	Type string // "path", "query", "header", "cookie", "form", "json"
	Name string // parameter name in that source
}

// Example extraction rules that would be generated from the struct tags
var complexAPIExtractionRules = []EnhancedFieldExtractor{
	{
		FieldName:  "UserID",
		Sources:    []DataSource{{"header", "X-User-ID"}, {"cookie", "user_id"}},
		Precedence: []string{"header", "cookie"},
		Validate:   "required",
	},
	{
		FieldName:  "TraceID",
		Sources:    []DataSource{{"header", "X-Trace-ID"}, {"query", "trace_id"}},
		Precedence: []string{"header", "query"},
	},
	{
		FieldName: "ClientIP",
		Sources:   []DataSource{{"header", "X-Forwarded-For"}},
		Transform: "first_ip",
	},
	// ... etc
}
