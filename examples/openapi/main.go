package main

import (
	"context"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"

	"github.com/pavelpascari/typedhttp/pkg/openapi"
	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
)

// Request types with OpenAPI comment documentation
type GetUserRequest struct {
	//openapi:description=User unique identifier,example=123e4567-e89b-12d3-a456-426614174000
	ID string `path:"id" validate:"required,uuid"`

	//openapi:description=Comma-separated list of fields to return,example=id,name,email
	Fields string `query:"fields" default:"id,name,email"`

	//openapi:description=Authorization bearer token
	Auth string `header:"Authorization" validate:"required"`
}

type GetUserResponse struct {
	//openapi:description=User unique identifier
	ID string `json:"id" validate:"required,uuid"`

	//openapi:description=User full name
	Name string `json:"name" validate:"required"`

	//openapi:description=User email address
	Email string `json:"email,omitempty" validate:"omitempty,email"`
}

type CreateUserRequest struct {
	//openapi:description=User full name,example=John Doe
	Name string `json:"name" validate:"required,min=2,max=50"`

	//openapi:description=User email address,example=john@example.com
	Email string `json:"email" validate:"required,email"`

	//openapi:description=User profile picture,type=file,format=binary
	Avatar *multipart.FileHeader `form:"avatar"`
}

type CreateUserResponse struct {
	//openapi:description=Created user unique identifier
	ID string `json:"id" validate:"required,uuid"`

	//openapi:description=User full name
	Name string `json:"name"`

	//openapi:description=User email address
	Email string `json:"email"`

	//openapi:description=Creation timestamp
	CreatedAt string `json:"created_at"`
}

// Handlers
type GetUserHandler struct{}

func (h *GetUserHandler) Handle(ctx context.Context, req GetUserRequest) (GetUserResponse, error) {
	fmt.Printf("Getting user %s with fields: %s, Auth: %s\n", req.ID, req.Fields, req.Auth)

	return GetUserResponse{
		ID:    req.ID,
		Name:  "John Doe",
		Email: "john@example.com",
	}, nil
}

type CreateUserHandler struct{}

func (h *CreateUserHandler) Handle(ctx context.Context, req CreateUserRequest) (CreateUserResponse, error) {
	fmt.Printf("Creating user: %s (%s)\n", req.Name, req.Email)
	if req.Avatar != nil {
		fmt.Printf("Avatar uploaded: %s (%d bytes)\n", req.Avatar.Filename, req.Avatar.Size)
	}

	return CreateUserResponse{
		ID:        "123e4567-e89b-12d3-a456-426614174000",
		Name:      req.Name,
		Email:     req.Email,
		CreatedAt: "2025-01-30T12:00:00Z",
	}, nil
}

func main() {
	fmt.Println("=== TypedHTTP OpenAPI Generation Demo ===")

	// Create router
	router := typedhttp.NewRouter()

	// Register handlers
	typedhttp.GET(router, "/users/{id}", &GetUserHandler{})
	typedhttp.POST(router, "/users", &CreateUserHandler{})

	// Create OpenAPI generator
	generator := openapi.NewGenerator(&openapi.Config{
		Info: openapi.Info{
			Title:       "User Management API",
			Version:     "1.0.0",
			Description: "A simple API for managing users with automatic OpenAPI generation",
		},
		Servers: []openapi.Server{
			{URL: "http://localhost:8080", Description: "Development server"},
		},
	})

	// Generate OpenAPI specification
	spec, err := generator.Generate(router)
	if err != nil {
		log.Fatalf("Failed to generate OpenAPI spec: %v", err)
	}

	fmt.Printf("✅ Generated OpenAPI spec with %d paths\n", len(spec.Paths.Map()))

	// Generate JSON output
	jsonData, err := generator.GenerateJSON(spec)
	if err != nil {
		log.Fatalf("Failed to generate JSON: %v", err)
	}

	fmt.Printf("✅ Generated JSON specification (%d bytes)\n", len(jsonData))
	fmt.Println("\n📄 OpenAPI JSON Specification Preview:")
	fmt.Println("=====================================")

	// Show first 500 characters of JSON
	preview := string(jsonData)
	if len(preview) > 500 {
		preview = preview[:500] + "..."
	}
	fmt.Println(preview)

	// Generate YAML output
	yamlData, err := generator.GenerateYAML(spec)
	if err != nil {
		log.Fatalf("Failed to generate YAML: %v", err)
	}

	fmt.Printf("\n✅ Generated YAML specification (%d bytes)\n", len(yamlData))
	fmt.Println("\n📄 OpenAPI YAML Specification Preview:")
	fmt.Println("======================================")

	// Show first 500 characters of YAML
	yamlPreview := string(yamlData)
	if len(yamlPreview) > 500 {
		yamlPreview = yamlPreview[:500] + "..."
	}
	fmt.Println(yamlPreview)

	// Setup HTTP server with OpenAPI endpoints
	http.Handle("/openapi.json", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
	}))

	http.Handle("/openapi.yaml", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-yaml")
		w.Write(yamlData)
	}))

	// Serve the API
	http.Handle("/", router)

	fmt.Println("\n🚀 Server starting on :8080")
	fmt.Println("📋 API Endpoints:")
	fmt.Println("   GET  /users/{id}    - Get user by ID")
	fmt.Println("   POST /users         - Create new user")
	fmt.Println("\n📖 Documentation:")
	fmt.Println("   GET /openapi.json   - OpenAPI JSON specification")
	fmt.Println("   GET /openapi.yaml   - OpenAPI YAML specification")
	fmt.Println("\n💡 Try these requests:")
	fmt.Println("   curl -H 'Authorization: Bearer token123' http://localhost:8080/users/123e4567-e89b-12d3-a456-426614174000")
	fmt.Println("   curl http://localhost:8080/openapi.json")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
