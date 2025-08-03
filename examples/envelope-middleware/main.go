package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/pavelpascari/typedhttp/pkg/openapi"
	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
)

// Request and Response types
type GetUserRequest struct {
	ID string `path:"id" validate:"required,uuid"`
}

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type GetUserResponse struct {
	User User `json:"user"`
}

type CreateUserRequest struct {
	Name  string `json:"name" validate:"required,min=2,max=50"`
	Email string `json:"email" validate:"required,email"`
}

type CreateUserResponse struct {
	User    User   `json:"user"`
	Message string `json:"message"`
}

// Handler implementations
type UserHandler struct{}

func (h *UserHandler) GetUser(ctx context.Context, req GetUserRequest) (GetUserResponse, error) {
	// Simulate user lookup
	user := User{
		ID:    req.ID,
		Name:  "John Doe",
		Email: "john.doe@example.com",
	}

	return GetUserResponse{User: user}, nil
}

func (h *UserHandler) CreateUser(ctx context.Context, req CreateUserRequest) (CreateUserResponse, error) {
	// Simulate user creation
	user := User{
		ID:    "550e8400-e29b-41d4-a716-446655440000",
		Name:  req.Name,
		Email: req.Email,
	}

	return CreateUserResponse{
		User:    user,
		Message: "User created successfully",
	}, nil
}

// Individual handler structs for type safety
type GetUserHandler struct {
	userHandler *UserHandler
}

func (h *GetUserHandler) Handle(ctx context.Context, req GetUserRequest) (GetUserResponse, error) {
	return h.userHandler.GetUser(ctx, req)
}

type CreateUserHandler struct {
	userHandler *UserHandler
}

func (h *CreateUserHandler) Handle(ctx context.Context, req CreateUserRequest) (CreateUserResponse, error) {
	return h.userHandler.CreateUser(ctx, req)
}

func main() {
	// Create router
	router := typedhttp.NewRouter()

	// Create handlers
	userHandler := &UserHandler{}
	getUserHandler := &GetUserHandler{userHandler: userHandler}
	createUserHandler := &CreateUserHandler{userHandler: userHandler}

	// Register handlers with envelope middleware
	// Note: For demonstration, we'll manually add middleware to the registration
	// In a real implementation, this would be done through handler options
	typedhttp.GET(router, "/users/{id}", getUserHandler)
	typedhttp.POST(router, "/users", createUserHandler)

	// For demonstration purposes, let's manually add envelope middleware to the registrations
	// This simulates what would happen when proper middleware integration is complete
	handlers := router.GetHandlers()
	for i := range handlers {
		handlers[i].MiddlewareEntries = []typedhttp.MiddlewareEntry{
			{
				Middleware: typedhttp.NewResponseEnvelopeMiddleware[any](
					typedhttp.WithRequestID(true),
					typedhttp.WithTimestamp(true),
					typedhttp.WithMeta(true),
				),
				Config: typedhttp.MiddlewareConfig{
					Name:     "response_envelope",
					Priority: 10,
					Scope:    typedhttp.ScopeGlobal,
				},
			},
		}
	}

	// Generate OpenAPI specification
	generator := openapi.NewGenerator(&openapi.Config{
		Info: openapi.Info{
			Title:       "User Management API with Envelope Middleware",
			Version:     "1.0.0",
			Description: "Demonstrates how envelope middleware affects OpenAPI schema generation",
		},
		Servers: []openapi.Server{
			{
				URL:         "http://localhost:8080",
				Description: "Development server",
			},
		},
	})

	spec, err := generator.Generate(router)
	if err != nil {
		log.Fatalf("Failed to generate OpenAPI spec: %v", err)
	}

	// Generate JSON and YAML
	jsonSpec, err := generator.GenerateJSON(spec)
	if err != nil {
		log.Fatalf("Failed to generate JSON spec: %v", err)
	}

	yamlSpec, err := generator.GenerateYAML(spec)
	if err != nil {
		log.Fatalf("Failed to generate YAML spec: %v", err)
	}

	// Serve OpenAPI endpoints
	http.HandleFunc("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonSpec)
	})

	http.HandleFunc("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-yaml")
		w.Write(yamlSpec)
	})

	// Serve Swagger UI (simple redirect to petstore for demo)
	http.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		html := `<!DOCTYPE html>
<html>
<head>
    <title>API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@3.52.5/swagger-ui.css" />
    <style>
        html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
        *, *:before, *:after { box-sizing: inherit; }
        body { margin:0; background: #fafafa; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@3.52.5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@3.52.5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            SwaggerUIBundle({
                url: '/openapi.json',
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout"
            });
        };
    </script>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})

	// Info endpoint
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		html := `<!DOCTYPE html>
<html>
<head>
    <title>Envelope Middleware Demo</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .endpoint { background: #f5f5f5; padding: 10px; margin: 10px 0; border-radius: 5px; }
        .method { font-weight: bold; color: #007bff; }
    </style>
</head>
<body>
    <h1>Envelope Middleware Demo</h1>
    <p>This example demonstrates how envelope middleware affects OpenAPI schema generation.</p>
    
    <h2>Available Endpoints:</h2>
    <div class="endpoint">
        <span class="method">GET</span> <a href="/openapi.json">/openapi.json</a> - OpenAPI JSON specification
    </div>
    <div class="endpoint">
        <span class="method">GET</span> <a href="/openapi.yaml">/openapi.yaml</a> - OpenAPI YAML specification
    </div>
    <div class="endpoint">
        <span class="method">GET</span> <a href="/docs">/docs</a> - Swagger UI documentation
    </div>
    
    <h2>API Endpoints (with envelope middleware):</h2>
    <div class="endpoint">
        <span class="method">GET</span> /users/{id} - Get user by ID
    </div>
    <div class="endpoint">
        <span class="method">POST</span> /users - Create new user
    </div>
    
    <h2>Key Features Demonstrated:</h2>
    <ul>
        <li><strong>Response Envelope</strong>: All API responses are wrapped in a standard envelope with data, error, success, and meta fields</li>
        <li><strong>Automatic Schema Generation</strong>: OpenAPI schemas reflect the actual envelope structure clients receive</li>
        <li><strong>Error Response Documentation</strong>: Standard error responses (400, 401, 404, 500) with envelope structure</li>
        <li><strong>Metadata Integration</strong>: Request IDs and timestamps are automatically included in response metadata</li>
    </ul>
    
    <h2>Schema Comparison:</h2>
    <h3>Without Envelope Middleware:</h3>
    <pre><code>{
  "user": {
    "id": "string",
    "name": "string", 
    "email": "string"
  }
}</code></pre>
    
    <h3>With Envelope Middleware:</h3>
    <pre><code>{
  "data": {
    "user": {
      "id": "string",
      "name": "string",
      "email": "string"
    }
  },
  "error": null,
  "success": true,
  "meta": {
    "request_id": "string",
    "timestamp": "2025-01-02T10:30:00Z"
  }
}</code></pre>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})

	fmt.Println("🚀 Server starting on http://localhost:8080")
	fmt.Println("📚 Visit http://localhost:8080 for demo info")
	fmt.Println("📖 Visit http://localhost:8080/docs for API documentation")
	fmt.Println("📄 Visit http://localhost:8080/openapi.json for OpenAPI spec")

	log.Fatal(http.ListenAndServe(":8080", router))
}
