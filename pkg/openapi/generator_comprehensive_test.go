package openapi

import (
	"context"
	"encoding/json"
	"mime/multipart"
	"net"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Test types for various comprehensive tests

// HTTP Methods test types.
type HTTPMethodsSimpleRequest struct {
	ID string `path:"id"`
}

type HTTPMethodsSimpleResponse struct {
	Message string `json:"message"`
}

type HTTPMethodsSimpleHandler struct{}

func (h *HTTPMethodsSimpleHandler) Handle(
	ctx context.Context, req HTTPMethodsSimpleRequest,
) (HTTPMethodsSimpleResponse, error) {
	return HTTPMethodsSimpleResponse{Message: "success"}, nil
}

// Complex Parameters test types.
type ComplexParametersRequest struct {
	// Path parameters
	UserID string `path:"user_id" validate:"required,uuid"`
	PostID int64  `path:"post_id" validate:"required,min=1"`

	// Query parameters
	Page     int      `query:"page" default:"1" validate:"min=1"`
	Limit    int      `query:"limit" default:"10" validate:"min=1,max=100"`
	Sort     string   `query:"sort" default:"created_at"`
	Filter   string   `query:"filter"`
	Tags     []string `query:"tags"`
	Active   bool     `query:"active" default:"true"`
	MinPrice float64  `query:"min_price"`

	// Header parameters
	UserAgent     string `header:"User-Agent"`
	Authorization string `header:"Authorization" validate:"required"`
	RequestID     string `header:"X-Request-ID"`
	ClientIP      string `header:"X-Forwarded-For"`

	// Cookie parameters
	SessionID   string `cookie:"session_id" validate:"required"`
	Preferences string `cookie:"user_prefs"`
	Language    string `cookie:"lang" default:"en"`
}

type ComplexParametersResponse struct {
	Data interface{} `json:"data"`
}

type ComplexParametersHandler struct{}

//nolint:gocritic // OpenAPI generator requires struct types, not pointers
func (h *ComplexParametersHandler) Handle(
	ctx context.Context, req ComplexParametersRequest,
) (ComplexParametersResponse, error) {
	return ComplexParametersResponse{Data: "response"}, nil
}

// Complex Schema test types.
type NestedStruct struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type ComplexSchemaRequest struct {
	// Primitive types
	StringField  string  `json:"string_field"`
	IntField     int     `json:"int_field"`
	Int8Field    int8    `json:"int8_field"`
	Int16Field   int16   `json:"int16_field"`
	Int32Field   int32   `json:"int32_field"`
	Int64Field   int64   `json:"int64_field"`
	UintField    uint    `json:"uint_field"`
	Uint8Field   uint8   `json:"uint8_field"`
	Uint16Field  uint16  `json:"uint16_field"`
	Uint32Field  uint32  `json:"uint32_field"`
	Uint64Field  uint64  `json:"uint64_field"`
	Float32Field float32 `json:"float32_field"`
	Float64Field float64 `json:"float64_field"`
	BoolField    bool    `json:"bool_field"`

	// Complex types
	SliceField     []string               `json:"slice_field"`
	IntSliceField  []int                  `json:"int_slice_field"`
	MapField       map[string]interface{} `json:"map_field"`
	StructField    NestedStruct           `json:"struct_field"`
	PointerField   *string                `json:"pointer_field,omitempty"`
	InterfaceField interface{}            `json:"interface_field"`

	// Special JSON tags
	OmitEmptyField string `json:"omit_empty_field,omitempty"`
	IgnoredField   string `json:"-"`
	NoJSONTag      string // Should be ignored
}

type ComplexSchemaResponse struct {
	Data interface{} `json:"data"`
}

type ComplexSchemaHandler struct{}

//nolint:gocritic // OpenAPI generator requires struct types, not pointers
func (h *ComplexSchemaHandler) Handle(
	ctx context.Context, req ComplexSchemaRequest,
) (ComplexSchemaResponse, error) {
	return ComplexSchemaResponse{Data: "response"}, nil
}

// File Upload test types.
type FileUploadRequest struct {
	// Regular form fields
	Title       string `form:"title" validate:"required,min=1,max=100"`
	Description string `form:"description"`
	Category    string `form:"category" default:"general"`
	Tags        string `form:"tags"`

	// File uploads
	MainFile    *multipart.FileHeader   `form:"main_file"`
	Attachments []*multipart.FileHeader `form:"attachments"`

	// JSON field in form
	Metadata map[string]interface{} `form:"metadata"`
}

type FileUploadResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

type FileUploadHandler struct{}

//nolint:gocritic // OpenAPI generator requires struct types, not pointers
func (h *FileUploadHandler) Handle(
	ctx context.Context, req FileUploadRequest,
) (FileUploadResponse, error) {
	return FileUploadResponse{ID: "123", Message: "uploaded"}, nil
}

// Validation test types.
type ValidationRequest struct {
	// String validations
	Email     string `query:"email" validate:"required,email"`
	UUID      string `query:"uuid" validate:"uuid"`
	MinLength string `query:"min_length" validate:"min=5"`
	MaxLength string `query:"max_length" validate:"max=50"`
	MinMaxStr string `query:"min_max_str" validate:"min=3,max=20"`

	// Numeric validations
	MinInt    int     `query:"min_int" validate:"min=1"`
	MaxInt    int     `query:"max_int" validate:"max=100"`
	MinMaxInt int     `query:"min_max_int" validate:"min=10,max=90"`
	MinFloat  float64 `query:"min_float" validate:"min=0"`
	MaxFloat  float64 `query:"max_float" validate:"max=999.99"`

	// Complex validation combinations
	ComplexField string `query:"complex" validate:"required,min=8,max=128,email"`
}

type ValidationResponse struct {
	Valid bool `json:"valid"`
}

type ValidationHandler struct{}

//nolint:gocritic // OpenAPI generator requires struct types, not pointers
func (h *ValidationHandler) Handle(
	ctx context.Context, req ValidationRequest,
) (ValidationResponse, error) {
	return ValidationResponse{Valid: true}, nil
}

// Default Values test types.
type DefaultValuesRequest struct {
	StringDefault     string  `query:"str" default:"hello"`
	IntDefault        int     `query:"int" default:"42"`
	Int64Default      int64   `query:"int64" default:"9223372036854775807"`
	UintDefault       uint    `query:"uint" default:"123"`
	Float64Default    float64 `query:"float64" default:"3.14159"`
	BoolTrueDefault   bool    `query:"bool_true" default:"true"`
	BoolFalseDefault  bool    `query:"bool_false" default:"false"`
	InvalidIntDefault string  `query:"invalid_int" default:"not-a-number"`
	StringPtrDefault  *string `query:"str_ptr" default:"pointer_value"`
}

type DefaultValuesResponse struct {
	Message string `json:"message"`
}

type DefaultValuesHandler struct{}

//nolint:gocritic // OpenAPI generator requires struct types, not pointers
func (h *DefaultValuesHandler) Handle(
	ctx context.Context, req DefaultValuesRequest,
) (DefaultValuesResponse, error) {
	return DefaultValuesResponse{Message: "success"}, nil
}

// Special Types test types.
type SpecialTypesRequest struct {
	// Time types
	TimeField time.Time  `json:"time_field"`
	TimePtr   *time.Time `json:"time_ptr,omitempty"`

	// Network types
	IPField net.IP `json:"ip_field"`

	// Interface types
	InterfaceField interface{} `json:"interface_field"`

	// Array vs Slice
	ArrayField [5]string `json:"array_field"`
	SliceField []string  `json:"slice_field"`

	// Nested slices/maps
	NestedSlice [][]string          `json:"nested_slice"`
	MapOfSlices map[string][]int    `json:"map_of_slices"`
	SliceOfMaps []map[string]string `json:"slice_of_maps"`
}

type SpecialTypesResponse struct {
	Data interface{} `json:"data"`
}

type SpecialTypesHandler struct{}

//nolint:gocritic // OpenAPI generator requires struct types, not pointers
func (h *SpecialTypesHandler) Handle(
	ctx context.Context, req SpecialTypesRequest,
) (SpecialTypesResponse, error) {
	return SpecialTypesResponse{Data: "response"}, nil
}

// Multiple Handlers test types.
type ComprehensiveUserRequest struct {
	ID string `path:"id"`
}

type ComprehensiveUserResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ComprehensivePostRequest struct {
	UserID string `path:"user_id"`
	PostID string `path:"post_id"`
}

type ComprehensivePostResponse struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	UserID string `json:"user_id"`
}

type ComprehensiveGetUserHandler struct{}

func (h *ComprehensiveGetUserHandler) Handle(
	ctx context.Context, req ComprehensiveUserRequest,
) (ComprehensiveUserResponse, error) {
	return ComprehensiveUserResponse{}, nil
}

type ComprehensiveUpdateUserHandler struct{}

func (h *ComprehensiveUpdateUserHandler) Handle(
	ctx context.Context, req ComprehensiveUserRequest,
) (ComprehensiveUserResponse, error) {
	return ComprehensiveUserResponse{}, nil
}

type ComprehensiveGetPostHandler struct{}

func (h *ComprehensiveGetPostHandler) Handle(
	ctx context.Context, req ComprehensivePostRequest,
) (ComprehensivePostResponse, error) {
	return ComprehensivePostResponse{}, nil
}

// Output Formats test types.
type OutputSimpleRequest struct {
	ID string `path:"id"`
}

type OutputSimpleResponse struct {
	Message string `json:"message"`
}

type OutputSimpleHandler struct{}

func (h *OutputSimpleHandler) Handle(ctx context.Context, req OutputSimpleRequest) (OutputSimpleResponse, error) {
	return OutputSimpleResponse{Message: "test"}, nil
}

// TEST FUNCTIONS

// TestOpenAPIGeneratorConfigurationOptions tests all configuration options comprehensively.
func TestOpenAPIGeneratorConfigurationOptions(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "minimal config",
			config: Config{
				Info: Info{
					Title:   "Minimal API",
					Version: "1.0.0",
				},
			},
		},
		{
			name: "complete config with all fields",
			config: Config{
				Info: Info{
					Title:          "Complete API",
					Version:        "2.1.0",
					Description:    "A comprehensive API with all features",
					TermsOfService: "https://example.com/terms",
					Contact: &Contact{
						Name:  "API Team",
						URL:   "https://example.com/contact",
						Email: "api-team@example.com",
					},
					License: &License{
						Name: "MIT",
						URL:  "https://opensource.org/licenses/MIT",
					},
				},
				Servers: []Server{
					{
						URL:         "https://api.example.com/v1",
						Description: "Production server",
					},
					{
						URL:         "https://staging-api.example.com/v1",
						Description: "Staging server",
					},
				},
				Security: map[string]SecurityScheme{
					"bearerAuth": {
						Type:         "http",
						Scheme:       "bearer",
						BearerFormat: "JWT",
					},
					"apiKeyAuth": {
						Type: "apiKey",
						In:   "header",
						Name: "X-API-Key",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := NewGenerator(&tt.config)
			require.NotNil(t, generator)
			assert.Equal(t, tt.config, generator.config)

			router := typedhttp.NewRouter()
			spec, err := generator.Generate(router)
			require.NoError(t, err)
			require.NotNil(t, spec)

			// Verify info propagation
			assert.Equal(t, tt.config.Info.Title, spec.Info.Title)
			assert.Equal(t, tt.config.Info.Version, spec.Info.Version)
			assert.Equal(t, tt.config.Info.Description, spec.Info.Description)

			// Verify servers propagation
			if len(tt.config.Servers) > 0 {
				require.Len(t, spec.Servers, len(tt.config.Servers))
				for i, server := range tt.config.Servers {
					assert.Equal(t, server.URL, spec.Servers[i].URL)
					assert.Equal(t, server.Description, spec.Servers[i].Description)
				}
			} else {
				assert.Nil(t, spec.Servers)
			}
		})
	}
}

// TestOpenAPIGeneratorAllHTTPMethods tests comprehensive HTTP method support.
func TestOpenAPIGeneratorAllHTTPMethods(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		setupHandler   func(*typedhttp.TypedRouter)
		expectedStatus int
		expectedDesc   string
	}{
		{
			name:   "GET method",
			method: "GET",
			setupHandler: func(router *typedhttp.TypedRouter) {
				typedhttp.GET(router, "/test/{id}", &HTTPMethodsSimpleHandler{})
			},
			expectedStatus: 200,
			expectedDesc:   "Success",
		},
		{
			name:   "POST method",
			method: "POST",
			setupHandler: func(router *typedhttp.TypedRouter) {
				typedhttp.POST(router, "/test/{id}", &HTTPMethodsSimpleHandler{})
			},
			expectedStatus: 201,
			expectedDesc:   "Created",
		},
		{
			name:   "PUT method",
			method: "PUT",
			setupHandler: func(router *typedhttp.TypedRouter) {
				typedhttp.PUT(router, "/test/{id}", &HTTPMethodsSimpleHandler{})
			},
			expectedStatus: 200,
			expectedDesc:   "Success",
		},
		{
			name:   "PATCH method",
			method: "PATCH",
			setupHandler: func(router *typedhttp.TypedRouter) {
				typedhttp.PATCH(router, "/test/{id}", &HTTPMethodsSimpleHandler{})
			},
			expectedStatus: 200,
			expectedDesc:   "Success",
		},
		{
			name:   "DELETE method",
			method: "DELETE",
			setupHandler: func(router *typedhttp.TypedRouter) {
				typedhttp.DELETE(router, "/test/{id}", &HTTPMethodsSimpleHandler{})
			},
			expectedStatus: 200,
			expectedDesc:   "Success",
		},
		{
			name:   "HEAD method",
			method: "HEAD",
			setupHandler: func(router *typedhttp.TypedRouter) {
				typedhttp.HEAD(router, "/test/{id}", &HTTPMethodsSimpleHandler{})
			},
			expectedStatus: 200,
			expectedDesc:   "Success",
		},
		{
			name:   "OPTIONS method",
			method: "OPTIONS",
			setupHandler: func(router *typedhttp.TypedRouter) {
				typedhttp.OPTIONS(router, "/test/{id}", &HTTPMethodsSimpleHandler{})
			},
			expectedStatus: 200,
			expectedDesc:   "Success",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Info: Info{
					Title:   "HTTP Methods API",
					Version: "1.0.0",
				},
			}

			generator := NewGenerator(&config)
			router := typedhttp.NewRouter()
			tt.setupHandler(router)

			spec, err := generator.Generate(router)
			require.NoError(t, err)

			pathItem := spec.Paths.Find("/test/{id}")
			require.NotNil(t, pathItem)

			var operation *openapi3.Operation
			switch tt.method {
			case "GET":
				operation = pathItem.Get
			case "POST":
				operation = pathItem.Post
			case "PUT":
				operation = pathItem.Put
			case "PATCH":
				operation = pathItem.Patch
			case "DELETE":
				operation = pathItem.Delete
			case "HEAD":
				operation = pathItem.Head
			case "OPTIONS":
				operation = pathItem.Options
			}

			require.NotNil(t, operation, "Operation for %s should exist", tt.method)

			// Verify response
			response := operation.Responses.Status(tt.expectedStatus)
			require.NotNil(t, response)
			assert.Equal(t, tt.expectedDesc, *response.Value.Description)
		})
	}
}

// TestOpenAPIGeneratorParameterTypes tests all parameter types and locations.
func TestOpenAPIGeneratorParameterTypes(t *testing.T) {
	config := Config{
		Info: Info{
			Title:   "Parameter Types API",
			Version: "1.0.0",
		},
	}

	generator := NewGenerator(&config)
	router := typedhttp.NewRouter()
	typedhttp.GET(router, "/users/{user_id}/posts/{post_id}", &ComplexParametersHandler{})

	spec, err := generator.Generate(router)
	require.NoError(t, err)

	pathItem := spec.Paths.Find("/users/{user_id}/posts/{post_id}")
	require.NotNil(t, pathItem)
	require.NotNil(t, pathItem.Get)

	operation := pathItem.Get
	require.NotEmpty(t, operation.Parameters)

	// Verify all parameters exist and have correct properties
	paramMap := make(map[string]*openapi3.Parameter)
	for _, param := range operation.Parameters {
		paramMap[param.Value.Name+":"+param.Value.In] = param.Value
	}

	// Test path parameters
	pathParam := paramMap["user_id:path"]
	require.NotNil(t, pathParam)
	assert.True(t, pathParam.Required)
	assert.Equal(t, "string", (*pathParam.Schema.Value.Type)[0])
	assert.Equal(t, "uuid", pathParam.Schema.Value.Format)

	postIDParam := paramMap["post_id:path"]
	require.NotNil(t, postIDParam)
	assert.True(t, postIDParam.Required)
	assert.Equal(t, "integer", (*postIDParam.Schema.Value.Type)[0])
	assert.Equal(t, float64(1), *postIDParam.Schema.Value.Min)

	// Test query parameters with defaults
	pageParam := paramMap["page:query"]
	require.NotNil(t, pageParam)
	assert.False(t, pageParam.Required) // Has default, so not required
	assert.Equal(t, int64(1), pageParam.Schema.Value.Default)

	limitParam := paramMap["limit:query"]
	require.NotNil(t, limitParam)
	assert.Equal(t, int64(10), limitParam.Schema.Value.Default)
	assert.Equal(t, float64(1), *limitParam.Schema.Value.Min)
	assert.Equal(t, float64(100), *limitParam.Schema.Value.Max)

	// Test header parameters
	authParam := paramMap["Authorization:header"]
	require.NotNil(t, authParam)
	assert.True(t, authParam.Required)

	userAgentParam := paramMap["User-Agent:header"]
	require.NotNil(t, userAgentParam)
	assert.False(t, userAgentParam.Required)

	// Test cookie parameters
	sessionParam := paramMap["session_id:cookie"]
	require.NotNil(t, sessionParam)
	assert.True(t, sessionParam.Required)

	langParam := paramMap["lang:cookie"]
	require.NotNil(t, langParam)
	assert.False(t, langParam.Required) // Has default
}

// TestOpenAPIGeneratorSchemaTypes tests comprehensive schema type generation.
func TestOpenAPIGeneratorSchemaTypes(t *testing.T) {
	config := Config{
		Info: Info{
			Title:   "Schema Types API",
			Version: "1.0.0",
		},
	}

	generator := NewGenerator(&config)
	router := typedhttp.NewRouter()
	typedhttp.POST(router, "/complex", &ComplexSchemaHandler{})

	spec, err := generator.Generate(router)
	require.NoError(t, err)

	pathItem := spec.Paths.Find("/complex")
	require.NotNil(t, pathItem)
	require.NotNil(t, pathItem.Post)

	operation := pathItem.Post
	require.NotNil(t, operation.RequestBody)

	// Should be JSON content type since no file uploads
	require.Contains(t, operation.RequestBody.Value.Content, "application/json")
	mediaType := operation.RequestBody.Value.Content["application/json"]
	require.NotNil(t, mediaType.Schema)

	schema := mediaType.Schema.Value
	require.NotNil(t, schema)
	assert.Equal(t, "object", (*schema.Type)[0])
	require.NotNil(t, schema.Properties)

	// Test primitive type schemas
	stringProp := schema.Properties["string_field"]
	require.NotNil(t, stringProp)
	assert.Equal(t, "string", (*stringProp.Value.Type)[0])

	intProp := schema.Properties["int_field"]
	require.NotNil(t, intProp)
	assert.Equal(t, "integer", (*intProp.Value.Type)[0])

	floatProp := schema.Properties["float32_field"]
	require.NotNil(t, floatProp)
	assert.Equal(t, "number", (*floatProp.Value.Type)[0])

	boolProp := schema.Properties["bool_field"]
	require.NotNil(t, boolProp)
	assert.Equal(t, "boolean", (*boolProp.Value.Type)[0])

	// Test complex type schemas
	sliceProp := schema.Properties["slice_field"]
	require.NotNil(t, sliceProp)
	assert.Equal(t, "array", (*sliceProp.Value.Type)[0])
	require.NotNil(t, sliceProp.Value.Items)
	assert.Equal(t, "string", (*sliceProp.Value.Items.Value.Type)[0])

	mapProp := schema.Properties["map_field"]
	require.NotNil(t, mapProp)
	assert.Equal(t, "object", (*mapProp.Value.Type)[0])
	assert.NotNil(t, mapProp.Value.AdditionalProperties.Has)
	assert.True(t, *mapProp.Value.AdditionalProperties.Has)

	structProp := schema.Properties["struct_field"]
	require.NotNil(t, structProp)
	assert.Equal(t, "object", (*structProp.Value.Type)[0])
	require.NotNil(t, structProp.Value.Properties)
	assert.Contains(t, structProp.Value.Properties, "id")
	assert.Contains(t, structProp.Value.Properties, "name")

	// Test omitempty handling
	assert.Contains(t, schema.Properties, "omit_empty_field")
	assert.NotContains(t, schema.Required, "omit_empty_field")

	// Test ignored fields
	assert.NotContains(t, schema.Properties, "IgnoredField")
	assert.NotContains(t, schema.Properties, "NoJSONTag")

	// Test required fields (non-omitempty)
	assert.Contains(t, schema.Required, "string_field")
	assert.Contains(t, schema.Required, "int_field")
}

// TestOpenAPIGeneratorFormDataAndFileUploads tests form data and file upload handling.
func TestOpenAPIGeneratorFormDataAndFileUploads(t *testing.T) {
	config := Config{
		Info: Info{
			Title:   "File Upload API",
			Version: "1.0.0",
		},
	}

	generator := NewGenerator(&config)
	router := typedhttp.NewRouter()
	typedhttp.POST(router, "/upload", &FileUploadHandler{})

	spec, err := generator.Generate(router)
	require.NoError(t, err)

	pathItem := spec.Paths.Find("/upload")
	require.NotNil(t, pathItem)
	require.NotNil(t, pathItem.Post)

	operation := pathItem.Post
	require.NotNil(t, operation.RequestBody)

	// Should be multipart/form-data due to file uploads
	require.Contains(t, operation.RequestBody.Value.Content, "multipart/form-data")
	mediaType := operation.RequestBody.Value.Content["multipart/form-data"]
	require.NotNil(t, mediaType.Schema)

	schema := mediaType.Schema.Value
	require.NotNil(t, schema)
	assert.Equal(t, "object", (*schema.Type)[0])

	// Test regular form fields
	titleProp := schema.Properties["title"]
	require.NotNil(t, titleProp)
	assert.Equal(t, "string", (*titleProp.Value.Type)[0])

	// Test single file upload
	mainFileProp := schema.Properties["main_file"]
	require.NotNil(t, mainFileProp)
	assert.Equal(t, "string", (*mainFileProp.Value.Type)[0])
	assert.Equal(t, "binary", mainFileProp.Value.Format)

	// Test multiple file uploads
	attachmentsProp := schema.Properties["attachments"]
	require.NotNil(t, attachmentsProp)
	assert.Equal(t, "array", (*attachmentsProp.Value.Type)[0])
	require.NotNil(t, attachmentsProp.Value.Items)
	assert.Equal(t, "string", (*attachmentsProp.Value.Items.Value.Type)[0])
	assert.Equal(t, "binary", attachmentsProp.Value.Items.Value.Format)

	// Test other form fields are included
	assert.Contains(t, schema.Properties, "description")
	assert.Contains(t, schema.Properties, "category")
	assert.Contains(t, schema.Properties, "tags")
	assert.Contains(t, schema.Properties, "metadata")
}

// TestOpenAPIGeneratorValidationConstraints tests validation rule processing.
func TestOpenAPIGeneratorValidationConstraints(t *testing.T) {
	config := Config{
		Info: Info{
			Title:   "Validation API",
			Version: "1.0.0",
		},
	}

	generator := NewGenerator(&config)
	router := typedhttp.NewRouter()
	typedhttp.GET(router, "/validate", &ValidationHandler{})

	spec, err := generator.Generate(router)
	require.NoError(t, err)

	pathItem := spec.Paths.Find("/validate")
	require.NotNil(t, pathItem)
	require.NotNil(t, pathItem.Get)

	operation := pathItem.Get
	require.NotEmpty(t, operation.Parameters)

	// Build parameter map for easier testing
	paramMap := make(map[string]*openapi3.Parameter)
	for _, param := range operation.Parameters {
		paramMap[param.Value.Name] = param.Value
	}

	// Test email format
	emailParam := paramMap["email"]
	require.NotNil(t, emailParam)
	assert.Equal(t, "email", emailParam.Schema.Value.Format)
	assert.True(t, emailParam.Required)

	// Test UUID format
	uuidParam := paramMap["uuid"]
	require.NotNil(t, uuidParam)
	assert.Equal(t, "uuid", uuidParam.Schema.Value.Format)

	// Test string length constraints
	minLengthParam := paramMap["min_length"]
	require.NotNil(t, minLengthParam)
	assert.Equal(t, uint64(5), minLengthParam.Schema.Value.MinLength)

	maxLengthParam := paramMap["max_length"]
	require.NotNil(t, maxLengthParam)
	assert.Equal(t, uint64(50), *maxLengthParam.Schema.Value.MaxLength)

	minMaxStrParam := paramMap["min_max_str"]
	require.NotNil(t, minMaxStrParam)
	assert.Equal(t, uint64(3), minMaxStrParam.Schema.Value.MinLength)
	assert.Equal(t, uint64(20), *minMaxStrParam.Schema.Value.MaxLength)

	// Test numeric constraints
	minIntParam := paramMap["min_int"]
	require.NotNil(t, minIntParam)
	assert.Equal(t, float64(1), *minIntParam.Schema.Value.Min)

	maxIntParam := paramMap["max_int"]
	require.NotNil(t, maxIntParam)
	assert.Equal(t, float64(100), *maxIntParam.Schema.Value.Max)

	minMaxIntParam := paramMap["min_max_int"]
	require.NotNil(t, minMaxIntParam)
	assert.Equal(t, float64(10), *minMaxIntParam.Schema.Value.Min)
	assert.Equal(t, float64(90), *minMaxIntParam.Schema.Value.Max)

	// Test complex validation combinations
	complexParam := paramMap["complex"]
	require.NotNil(t, complexParam)
	assert.True(t, complexParam.Required)
	assert.Equal(t, uint64(8), complexParam.Schema.Value.MinLength)
	assert.Equal(t, uint64(128), *complexParam.Schema.Value.MaxLength)
	assert.Equal(t, "email", complexParam.Schema.Value.Format)
}

// TestOpenAPIGeneratorDefaultValues tests default value parsing for all types.
func TestOpenAPIGeneratorDefaultValues(t *testing.T) {
	config := Config{
		Info: Info{
			Title:   "Default Values API",
			Version: "1.0.0",
		},
	}

	generator := NewGenerator(&config)
	router := typedhttp.NewRouter()
	typedhttp.GET(router, "/defaults", &DefaultValuesHandler{})

	spec, err := generator.Generate(router)
	require.NoError(t, err)

	pathItem := spec.Paths.Find("/defaults")
	require.NotNil(t, pathItem)
	require.NotNil(t, pathItem.Get)

	operation := pathItem.Get
	require.NotEmpty(t, operation.Parameters)

	// Build parameter map for easier testing
	paramMap := make(map[string]*openapi3.Parameter)
	for _, param := range operation.Parameters {
		paramMap[param.Value.Name] = param.Value
	}

	// Test string default
	strParam := paramMap["str"]
	require.NotNil(t, strParam)
	assert.Equal(t, "hello", strParam.Schema.Value.Default)
	assert.False(t, strParam.Required) // Has default

	// Test integer default
	intParam := paramMap["int"]
	require.NotNil(t, intParam)
	assert.Equal(t, int64(42), intParam.Schema.Value.Default)

	// Test int64 default
	int64Param := paramMap["int64"]
	require.NotNil(t, int64Param)
	assert.Equal(t, int64(9223372036854775807), int64Param.Schema.Value.Default)

	// Test uint default
	uintParam := paramMap["uint"]
	require.NotNil(t, uintParam)
	assert.Equal(t, uint64(123), uintParam.Schema.Value.Default)

	// Test float default
	floatParam := paramMap["float64"]
	require.NotNil(t, floatParam)
	assert.Equal(t, 3.14159, floatParam.Schema.Value.Default)

	// Test boolean defaults
	boolTrueParam := paramMap["bool_true"]
	require.NotNil(t, boolTrueParam)
	assert.Equal(t, true, boolTrueParam.Schema.Value.Default)

	boolFalseParam := paramMap["bool_false"]
	require.NotNil(t, boolFalseParam)
	assert.Equal(t, false, boolFalseParam.Schema.Value.Default)

	// Test invalid default fallback
	invalidParam := paramMap["invalid_int"]
	require.NotNil(t, invalidParam)
	assert.Equal(t, "not-a-number", invalidParam.Schema.Value.Default)

	// Test pointer type default
	ptrParam := paramMap["str_ptr"]
	require.NotNil(t, ptrParam)
	assert.Equal(t, "pointer_value", ptrParam.Schema.Value.Default)
}

// TestOpenAPIGeneratorSpecialTypes tests handling of special Go types.
func TestOpenAPIGeneratorSpecialTypes(t *testing.T) {
	config := Config{
		Info: Info{
			Title:   "Special Types API",
			Version: "1.0.0",
		},
	}

	generator := NewGenerator(&config)
	router := typedhttp.NewRouter()
	typedhttp.POST(router, "/special", &SpecialTypesHandler{})

	spec, err := generator.Generate(router)
	require.NoError(t, err)

	pathItem := spec.Paths.Find("/special")
	require.NotNil(t, pathItem)
	require.NotNil(t, pathItem.Post)

	operation := pathItem.Post
	require.NotNil(t, operation.RequestBody)
	require.Contains(t, operation.RequestBody.Value.Content, "application/json")

	schema := operation.RequestBody.Value.Content["application/json"].Schema.Value
	require.NotNil(t, schema)

	// Test array field (should be treated as array)
	arrayProp := schema.Properties["array_field"]
	require.NotNil(t, arrayProp)
	assert.Equal(t, "array", (*arrayProp.Value.Type)[0])
	require.NotNil(t, arrayProp.Value.Items)
	assert.Equal(t, "string", (*arrayProp.Value.Items.Value.Type)[0])

	// Test slice field
	sliceProp := schema.Properties["slice_field"]
	require.NotNil(t, sliceProp)
	assert.Equal(t, "array", (*sliceProp.Value.Type)[0])

	// Test nested slice
	nestedSliceProp := schema.Properties["nested_slice"]
	require.NotNil(t, nestedSliceProp)
	assert.Equal(t, "array", (*nestedSliceProp.Value.Type)[0])
	require.NotNil(t, nestedSliceProp.Value.Items)
	assert.Equal(t, "array", (*nestedSliceProp.Value.Items.Value.Type)[0])
	require.NotNil(t, nestedSliceProp.Value.Items.Value.Items)
	assert.Equal(t, "string", (*nestedSliceProp.Value.Items.Value.Items.Value.Type)[0])

	// Test map of slices
	mapOfSlicesProp := schema.Properties["map_of_slices"]
	require.NotNil(t, mapOfSlicesProp)
	assert.Equal(t, "object", (*mapOfSlicesProp.Value.Type)[0])
	assert.True(t, *mapOfSlicesProp.Value.AdditionalProperties.Has)

	// Test slice of maps
	sliceOfMapsProp := schema.Properties["slice_of_maps"]
	require.NotNil(t, sliceOfMapsProp)
	assert.Equal(t, "array", (*sliceOfMapsProp.Value.Type)[0])
	require.NotNil(t, sliceOfMapsProp.Value.Items)
	assert.Equal(t, "object", (*sliceOfMapsProp.Value.Items.Value.Type)[0])

	// Test interface field
	interfaceProp := schema.Properties["interface_field"]
	require.NotNil(t, interfaceProp)
	assert.Equal(t, "object", (*interfaceProp.Value.Type)[0])
}

// TestOpenAPIGeneratorMultipleHandlers tests handling multiple handlers and path conflicts.
func TestOpenAPIGeneratorMultipleHandlers(t *testing.T) {
	config := Config{
		Info: Info{
			Title:   "Multiple Handlers API",
			Version: "1.0.0",
		},
	}

	generator := NewGenerator(&config)
	router := typedhttp.NewRouter()

	// Register multiple handlers on same path but different methods
	typedhttp.GET(router, "/users/{id}", &ComprehensiveGetUserHandler{})
	typedhttp.PUT(router, "/users/{id}", &ComprehensiveUpdateUserHandler{})

	// Register handler on different path
	typedhttp.GET(router, "/users/{user_id}/posts/{post_id}", &ComprehensiveGetPostHandler{})

	spec, err := generator.Generate(router)
	require.NoError(t, err)

	// Test that both methods exist on the same path
	userPath := spec.Paths.Find("/users/{id}")
	require.NotNil(t, userPath)
	require.NotNil(t, userPath.Get)
	require.NotNil(t, userPath.Put)

	// Test that separate path exists
	postPath := spec.Paths.Find("/users/{user_id}/posts/{post_id}")
	require.NotNil(t, postPath)
	require.NotNil(t, postPath.Get)

	// Verify response types are correct
	getUserResp := userPath.Get.Responses.Status(200)
	require.NotNil(t, getUserResp)
	assert.Equal(t, "Success", *getUserResp.Value.Description)

	putUserResp := userPath.Put.Responses.Status(200)
	require.NotNil(t, putUserResp)
	assert.Equal(t, "Success", *putUserResp.Value.Description)

	getPostResp := postPath.Get.Responses.Status(200)
	require.NotNil(t, getPostResp)
	assert.Equal(t, "Success", *getPostResp.Value.Description)
}

// TestOpenAPIGeneratorErrorHandling tests error cases and edge conditions.
func TestOpenAPIGeneratorErrorHandling(t *testing.T) {
	config := Config{
		Info: Info{
			Title:   "Error Handling API",
			Version: "1.0.0",
		},
	}

	generator := NewGenerator(&config)

	t.Run("empty router", func(t *testing.T) {
		router := typedhttp.NewRouter()
		spec, err := generator.Generate(router)
		require.NoError(t, err)
		require.NotNil(t, spec)
		assert.Empty(t, spec.Paths.Map())
	})
}

// TestOpenAPIGeneratorOutputFormats tests JSON and YAML generation.
func TestOpenAPIGeneratorOutputFormats(t *testing.T) {
	config := Config{
		Info: Info{
			Title:       "Output Formats API",
			Version:     "2.1.0",
			Description: "Testing JSON and YAML output formats",
		},
		Servers: []Server{
			{
				URL:         "https://api.example.com",
				Description: "Production server",
			},
		},
	}

	generator := NewGenerator(&config)
	router := typedhttp.NewRouter()
	typedhttp.GET(router, "/test/{id}", &OutputSimpleHandler{})

	spec, err := generator.Generate(router)
	require.NoError(t, err)

	t.Run("JSON generation", func(t *testing.T) {
		jsonData, err := generator.GenerateJSON(spec)
		require.NoError(t, err)

		jsonStr := string(jsonData)
		assert.Contains(t, jsonStr, `"openapi": "3.0.3"`)
		assert.Contains(t, jsonStr, `"title": "Output Formats API"`)
		assert.Contains(t, jsonStr, `"version": "2.1.0"`)
		assert.Contains(t, jsonStr, `"description": "Testing JSON and YAML output formats"`)
		assert.Contains(t, jsonStr, `"/test/{id}"`)
		assert.Contains(t, jsonStr, `"https://api.example.com"`)

		// Verify it's valid JSON by unmarshaling
		var jsonObj map[string]interface{}
		err = json.Unmarshal(jsonData, &jsonObj)
		require.NoError(t, err)
	})

	t.Run("YAML generation", func(t *testing.T) {
		yamlData, err := generator.GenerateYAML(spec)
		require.NoError(t, err)

		yamlStr := string(yamlData)
		assert.Contains(t, yamlStr, "openapi: 3.0.3")
		assert.Contains(t, yamlStr, "title: Output Formats API")
		assert.Contains(t, yamlStr, "version: 2.1.0")
		assert.Contains(t, yamlStr, "description: Testing JSON and YAML output formats")
		assert.Contains(t, yamlStr, "/test/{id}:")
		assert.Contains(t, yamlStr, "url: https://api.example.com")

		// Verify it's valid YAML by unmarshaling
		var yamlObj map[string]interface{}
		err = yaml.Unmarshal(yamlData, &yamlObj)
		require.NoError(t, err)
	})
}
