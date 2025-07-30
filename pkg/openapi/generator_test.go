package openapi

import (
	"context"
	"mime/multipart"
	"testing"

	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types for TDD..
type GetUserRequest struct {
	//openapi:description=User unique identifier,example=123e4567-e89b-12d3-a456-426614174000
	ID string `path:"id" validate:"required,uuid"`

	//openapi:description=Fields to return,example=id,name,email
	Fields string `query:"fields" default:"id,name,email"`
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

	//openapi:description=Creation timestamp
	CreatedAt string `json:"created_at"`
}

// Test handlers.
type GetUserHandler struct{}

func (h *GetUserHandler) Handle(ctx context.Context, req GetUserRequest) (GetUserResponse, error) {
	return GetUserResponse{
		ID:    req.ID,
		Name:  "John Doe",
		Email: "john@example.com",
	}, nil
}

type CreateUserHandler struct{}

func (h *CreateUserHandler) Handle(ctx context.Context, req CreateUserRequest) (CreateUserResponse, error) {
	return CreateUserResponse{
		ID:        "123e4567-e89b-12d3-a456-426614174000",
		CreatedAt: "2025-01-30T10:00:00Z",
	}, nil
}

// TDD Test 1: Basic Generator Creation.
func TestNewGenerator(t *testing.T) {
	config := Config{
		Info: Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
	}

	generator := NewGenerator(&config)
	require.NotNil(t, generator)
	assert.Equal(t, "Test API", generator.config.Info.Title)
	assert.Equal(t, "1.0.0", generator.config.Info.Version)
}

// TDD Test 2: Generate Empty Spec from Empty Router.
func TestGenerate_EmptyRouter(t *testing.T) {
	config := Config{
		Info: Info{
			Title:   "Empty API",
			Version: "1.0.0",
		},
	}

	generator := NewGenerator(&config)
	router := typedhttp.NewRouter()

	spec, err := generator.Generate(router)
	require.NoError(t, err)
	require.NotNil(t, spec)

	assert.Equal(t, "3.0.3", spec.OpenAPI)
	assert.Equal(t, "Empty API", spec.Info.Title)
	assert.Equal(t, "1.0.0", spec.Info.Version)
	assert.Empty(t, spec.Paths)
}

// TDD Test 3: Generate Spec with Single GET Handler.
func TestGenerate_SingleGETHandler(t *testing.T) {
	config := Config{
		Info: Info{
			Title:   "User API",
			Version: "1.0.0",
		},
	}

	generator := NewGenerator(&config)
	router := typedhttp.NewRouter()

	// Register GET handler.
	typedhttp.GET(router, "/users/{id}", &GetUserHandler{})

	spec, err := generator.Generate(router)
	require.NoError(t, err)
	require.NotNil(t, spec)

	// Verify basic structure.
	assert.Equal(t, "3.0.3", spec.OpenAPI)
	assert.Equal(t, "User API", spec.Info.Title)

	// Verify paths.
	pathItem := spec.Paths.Find("/users/{id}")
	require.NotNil(t, pathItem)
	require.NotNil(t, pathItem.Get)

	// Verify GET operation.
	operation := pathItem.Get
	assert.NotEmpty(t, operation.Parameters)

	// Verify path parameter.
	found := false
	for _, param := range operation.Parameters {
		if param.Value.Name != "id" || param.Value.In != "path" {
			continue
		}
		found = true
		assert.True(t, param.Value.Required)
		if param.Value.Schema.Value.Type != nil && len(*param.Value.Schema.Value.Type) > 0 {
			assert.Equal(t, "string", (*param.Value.Schema.Value.Type)[0])
		}
		assert.Equal(t, "uuid", param.Value.Schema.Value.Format)

		break
	}
	assert.True(t, found, "Path parameter 'id' not found")

	// Verify query parameter.
	found = false
	for _, param := range operation.Parameters {
		if param.Value.Name != "fields" || param.Value.In != "query" {
			continue
		}
		found = true
		assert.False(t, param.Value.Required)
		if param.Value.Schema.Value.Type != nil && len(*param.Value.Schema.Value.Type) > 0 {
			assert.Equal(t, "string", (*param.Value.Schema.Value.Type)[0])
		}
		assert.Equal(t, "id,name,email", param.Value.Schema.Value.Default)

		break
	}
	assert.True(t, found, "Query parameter 'fields' not found")

	// Verify response.
	response := operation.Responses.Status(200)
	require.NotNil(t, response)
	require.NotNil(t, response.Value)
	assert.Equal(t, "Success", *response.Value.Description)

	// Verify response content.
	require.Contains(t, response.Value.Content, "application/json")
	mediaType := response.Value.Content["application/json"]
	require.NotNil(t, mediaType.Schema)
	require.NotNil(t, mediaType.Schema.Value)
	if mediaType.Schema.Value.Type != nil && len(*mediaType.Schema.Value.Type) > 0 {
		assert.Equal(t, "object", (*mediaType.Schema.Value.Type)[0])
	}
}

// TDD Test 4: Generate Spec with POST Handler and Request Body.
func TestGenerate_POSTHandlerWithRequestBody(t *testing.T) {
	config := Config{
		Info: Info{
			Title:   "User API",
			Version: "1.0.0",
		},
	}

	generator := NewGenerator(&config)
	router := typedhttp.NewRouter()

	// Register POST handler.
	typedhttp.POST(router, "/users", &CreateUserHandler{})

	spec, err := generator.Generate(router)
	require.NoError(t, err)
	require.NotNil(t, spec)

	// Verify paths.
	pathItem := spec.Paths.Find("/users")
	require.NotNil(t, pathItem)
	require.NotNil(t, pathItem.Post)

	// Verify POST operation.
	operation := pathItem.Post

	// Verify request body for multipart form (because of file upload).
	require.NotNil(t, operation.RequestBody)
	require.NotNil(t, operation.RequestBody.Value)

	// Should have multipart/form-data for file upload.
	require.Contains(t, operation.RequestBody.Value.Content, "multipart/form-data")
	mediaType := operation.RequestBody.Value.Content["multipart/form-data"]
	require.NotNil(t, mediaType.Schema)
	require.NotNil(t, mediaType.Schema.Value)
	if mediaType.Schema.Value.Type != nil && len(*mediaType.Schema.Value.Type) > 0 {
		assert.Equal(t, "object", (*mediaType.Schema.Value.Type)[0])
	}

	// Verify response.
	response := operation.Responses.Status(201)
	require.NotNil(t, response)
	require.NotNil(t, response.Value)
	assert.Equal(t, "Created", *response.Value.Description)
}

// TDD Test 5: Extract OpenAPI Comments from Source.
func TestParseOpenAPIComment(t *testing.T) {
	tests := []struct {
		name     string
		comment  string
		expected map[string]string
		wantErr  bool
	}{
		{
			name:    "simple description",
			comment: "//openapi:description=User ID",
			expected: map[string]string{
				"description": "User ID",
			},
		},
		{
			name:    "multiple properties",
			comment: "//openapi:description=User full name,example=John Doe",
			expected: map[string]string{
				"description": "User full name",
				"example":     "John Doe",
			},
		},
		{
			name:    "file type",
			comment: "//openapi:description=File to upload,type=file,format=binary",
			expected: map[string]string{
				"description": "File to upload",
				"type":        "file",
				"format":      "binary",
			},
		},
		{
			name:    "with spaces",
			comment: "//openapi:description=A complex description with spaces,example=Test Value",
			expected: map[string]string{
				"description": "A complex description with spaces",
				"example":     "Test Value",
			},
		},
		{
			name:     "not openapi comment",
			comment:  "// Regular comment",
			expected: nil,
		},
		{
			name:     "empty openapi comment",
			comment:  "//openapi:",
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseOpenAPIComment(tt.comment)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TDD Test 6: JSON Output Generation.
func TestGenerator_GenerateJSON(t *testing.T) {
	config := Config{
		Info: Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
	}

	generator := NewGenerator(&config)
	router := typedhttp.NewRouter()
	typedhttp.GET(router, "/users/{id}", &GetUserHandler{})

	// Generate spec.
	spec, err := generator.Generate(router)
	require.NoError(t, err)

	// Test JSON generation.
	jsonData, err := generator.GenerateJSON(spec)
	require.NoError(t, err)
	assert.Contains(t, string(jsonData), `"openapi": "3.0.3"`)
	assert.Contains(t, string(jsonData), `"title": "Test API"`)
	assert.Contains(t, string(jsonData), `"/users/{id}"`)
}

// TDD Test 7: YAML Output Generation.
func TestGenerator_GenerateYAML(t *testing.T) {
	config := Config{
		Info: Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
	}

	generator := NewGenerator(&config)
	router := typedhttp.NewRouter()
	typedhttp.GET(router, "/users/{id}", &GetUserHandler{})

	// Generate spec.
	spec, err := generator.Generate(router)
	require.NoError(t, err)

	// Test YAML generation.
	yamlData, err := generator.GenerateYAML(spec)
	require.NoError(t, err)
	assert.Contains(t, string(yamlData), "openapi: 3.0.3")
	assert.Contains(t, string(yamlData), "title: Test API")
	assert.Contains(t, string(yamlData), "/users/{id}")
}
