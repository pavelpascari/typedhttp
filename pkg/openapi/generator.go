package openapi

import (
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
	"gopkg.in/yaml.v3"
)

// Config holds OpenAPI generation configuration.
type Config struct {
	Info     Info                      `json:"info"`
	Servers  []Server                  `json:"servers,omitempty"`
	Security map[string]SecurityScheme `json:"security,omitempty"`
}

// Info represents OpenAPI info object.
type Info struct {
	Title          string   `json:"title"`
	Version        string   `json:"version"`
	Description    string   `json:"description,omitempty"`
	TermsOfService string   `json:"terms_of_service,omitempty"`
	Contact        *Contact `json:"contact,omitempty"`
	License        *License `json:"license,omitempty"`
}

// Server represents OpenAPI server object.
type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// Contact represents OpenAPI contact object.
type Contact struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

// License represents OpenAPI license object.
type License struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// SecurityScheme represents OpenAPI security scheme.
type SecurityScheme struct {
	Type         string `json:"type"`
	Scheme       string `json:"scheme,omitempty"`
	BearerFormat string `json:"bearer_format,omitempty"`
	In           string `json:"in,omitempty"`
	Name         string `json:"name,omitempty"`
}

// Generator generates OpenAPI specifications from TypedHTTP routers.
type Generator struct {
	config Config
}

// NewGenerator creates a new OpenAPI generator.
func NewGenerator(config *Config) *Generator {
	return &Generator{
		config: *config,
	}
}

// Generate creates an OpenAPI specification from a TypedHTTP router.
func (g *Generator) Generate(router *typedhttp.TypedRouter) (*openapi3.T, error) {
	spec := &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:       g.config.Info.Title,
			Version:     g.config.Info.Version,
			Description: g.config.Info.Description,
		},
		Paths: &openapi3.Paths{},
		Components: &openapi3.Components{
			Schemas: make(map[string]*openapi3.SchemaRef),
		},
	}

	// Add servers if configured
	if len(g.config.Servers) > 0 {
		spec.Servers = make([]*openapi3.Server, len(g.config.Servers))
		for i, server := range g.config.Servers {
			spec.Servers[i] = &openapi3.Server{
				URL:         server.URL,
				Description: server.Description,
			}
		}
	}

	// Process each registered handler
	handlers := router.GetHandlers()
	for i := range handlers {
		err := g.processHandler(spec, &handlers[i])
		if err != nil {
			return nil, fmt.Errorf("failed to process handler %s %s: %w",
				handlers[i].Method, handlers[i].Path, err)
		}
	}

	return spec, nil
}

// processHandler processes a single handler registration.
func (g *Generator) processHandler(spec *openapi3.T, reg *typedhttp.HandlerRegistration) error {
	// Get or create path item
	pathItem := spec.Paths.Find(reg.Path)
	if pathItem == nil {
		pathItem = &openapi3.PathItem{}
		spec.Paths.Set(reg.Path, pathItem)
	}

	// Create operation
	operation := &openapi3.Operation{
		Responses: &openapi3.Responses{},
	}

	// Extract parameters from request type
	parameters, err := g.extractParameters(reg.RequestType)
	if err != nil {
		return fmt.Errorf("failed to extract parameters: %w", err)
	}
	operation.Parameters = parameters

	// Check if we need a request body
	if g.needsRequestBody(reg.RequestType) {
		requestBody, err := g.createRequestBody(reg.RequestType)
		if err != nil {
			return fmt.Errorf("failed to create request body: %w", err)
		}
		operation.RequestBody = requestBody
	}

	// Create base response schema
	baseResponseSchema, err := g.createResponseSchema(reg.ResponseType)
	if err != nil {
		return fmt.Errorf("failed to create response schema: %w", err)
	}

	// Apply middleware schema transformations
	finalResponseSchema, err := g.applyMiddlewareSchemaTransformations(
		context.Background(),
		reg.MiddlewareEntries,
		baseResponseSchema,
	)
	if err != nil {
		return fmt.Errorf("failed to apply middleware schema transformations: %w", err)
	}

	statusCode := "200"
	description := "Success"
	if reg.Method == http.MethodPost {
		statusCode = "201"
		description = "Created"
	}

	operation.Responses.Set(statusCode, &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Description: &description,
			Content: map[string]*openapi3.MediaType{
				"application/json": {
					Schema: finalResponseSchema,
				},
			},
		},
	})

	// Add error responses if envelope middleware is present
	if g.hasEnvelopeMiddleware(reg.MiddlewareEntries) {
		g.addEnvelopeErrorResponses(operation)
	}

	// Assign operation to method
	switch reg.Method {
	case http.MethodGet:
		pathItem.Get = operation
	case http.MethodPost:
		pathItem.Post = operation
	case http.MethodPut:
		pathItem.Put = operation
	case http.MethodPatch:
		pathItem.Patch = operation
	case http.MethodDelete:
		pathItem.Delete = operation
	case http.MethodHead:
		pathItem.Head = operation
	case http.MethodOptions:
		pathItem.Options = operation
	}

	return nil
}

// extractParameters extracts OpenAPI parameters from request type.
func (g *Generator) extractParameters(requestType reflect.Type) (openapi3.Parameters, error) {
	var parameters openapi3.Parameters

	for i := 0; i < requestType.NumField(); i++ {
		field := requestType.Field(i)

		if !field.IsExported() {
			continue
		}

		fieldParams, err := g.extractFieldParameters(&field)
		if err != nil {
			return nil, err
		}

		parameters = append(parameters, fieldParams...)
	}

	return parameters, nil
}

// extractFieldParameters extracts parameters from a single field.
func (g *Generator) extractFieldParameters(field *reflect.StructField) (openapi3.Parameters, error) {
	var parameters openapi3.Parameters

	// Check for path parameters
	if pathName := field.Tag.Get("path"); pathName != "" {
		param, err := g.createParameter(field, "path", pathName, true)
		if err != nil {
			return nil, err
		}
		parameters = append(parameters, param)
	}

	// Check for query parameters
	if queryName := field.Tag.Get("query"); queryName != "" {
		param, err := g.createQueryParameter(field, queryName)
		if err != nil {
			return nil, err
		}
		parameters = append(parameters, param)
	}

	// Check for header parameters
	if headerName := field.Tag.Get("header"); headerName != "" {
		required := strings.Contains(field.Tag.Get("validate"), "required")
		param, err := g.createParameter(field, "header", headerName, required)
		if err != nil {
			return nil, err
		}
		parameters = append(parameters, param)
	}

	// Check for cookie parameters
	if cookieName := field.Tag.Get("cookie"); cookieName != "" {
		required := strings.Contains(field.Tag.Get("validate"), "required")
		param, err := g.createParameter(field, "cookie", cookieName, required)
		if err != nil {
			return nil, err
		}
		parameters = append(parameters, param)
	}

	return parameters, nil
}

// createQueryParameter creates a query parameter with default value handling.
func (g *Generator) createQueryParameter(field *reflect.StructField, queryName string) (*openapi3.ParameterRef, error) {
	defaultValue := field.Tag.Get("default")
	required := defaultValue == ""

	param, err := g.createParameter(field, "query", queryName, required)
	if err != nil {
		return nil, err
	}

	if defaultValue != "" {
		param.Value.Schema.Value.Default = g.parseDefaultValue(defaultValue, field.Type)
	}

	return param, nil
}

// createParameter creates an OpenAPI parameter from a struct field.
func (g *Generator) createParameter(
	field *reflect.StructField, in, name string, required bool,
) (*openapi3.ParameterRef, error) {
	schema, err := g.createSchemaFromType(field.Type)
	if err != nil {
		return nil, err
	}

	// Apply validation constraints
	g.applyValidationToSchema(schema, field.Tag.Get("validate"))

	param := &openapi3.Parameter{
		Name:     name,
		In:       in,
		Required: required,
		Schema:   schema,
	}

	return &openapi3.ParameterRef{Value: param}, nil
}

// needsRequestBody determines if a request type needs a request body.
func (g *Generator) needsRequestBody(requestType reflect.Type) bool {
	for i := 0; i < requestType.NumField(); i++ {
		field := requestType.Field(i)

		// Check for JSON body fields
		if field.Tag.Get("json") != "" {
			return true
		}

		// Check for form fields (including file uploads)
		if field.Tag.Get("form") != "" {
			return true
		}
	}

	return false
}

// createRequestBody creates an OpenAPI request body from request type.
func (g *Generator) createRequestBody(requestType reflect.Type) (*openapi3.RequestBodyRef, error) {
	content := make(map[string]*openapi3.MediaType)

	// Check if we have file uploads (multipart form)
	hasFiles := g.hasFileUploads(requestType)

	if hasFiles {
		// Create multipart/form-data schema
		schema, err := g.createFormSchema(requestType)
		if err != nil {
			return nil, err
		}
		content["multipart/form-data"] = &openapi3.MediaType{Schema: schema}
	} else {
		// Create JSON schema
		schema, err := g.createSchemaFromType(requestType)
		if err != nil {
			return nil, err
		}
		content["application/json"] = &openapi3.MediaType{Schema: schema}
	}

	return &openapi3.RequestBodyRef{
		Value: &openapi3.RequestBody{
			Content: content,
		},
	}, nil
}

// hasFileUploads checks if request type has file upload fields.
func (g *Generator) hasFileUploads(requestType reflect.Type) bool {
	for i := 0; i < requestType.NumField(); i++ {
		field := requestType.Field(i)
		if field.Type == reflect.TypeOf((*multipart.FileHeader)(nil)) ||
			field.Type == reflect.TypeOf([]*multipart.FileHeader{}) {
			return true
		}
	}

	return false
}

// createFormSchema creates schema for form data.
func (g *Generator) createFormSchema(requestType reflect.Type) (*openapi3.SchemaRef, error) {
	schema := &openapi3.Schema{
		Type:       &openapi3.Types{"object"},
		Properties: make(map[string]*openapi3.SchemaRef),
	}

	for i := 0; i < requestType.NumField(); i++ {
		field := requestType.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		formName := field.Tag.Get("form")
		if formName == "" {
			continue
		}

		var fieldSchema *openapi3.SchemaRef
		var err error

		// Handle file uploads
		if field.Type == reflect.TypeOf((*multipart.FileHeader)(nil)) {
			fieldSchema = &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type:   &openapi3.Types{"string"},
					Format: "binary",
				},
			}
		} else if field.Type == reflect.TypeOf([]*multipart.FileHeader{}) {
			fieldSchema = &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type: &openapi3.Types{"array"},
					Items: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type:   &openapi3.Types{"string"},
							Format: "binary",
						},
					},
				},
			}
		} else {
			fieldSchema, err = g.createSchemaFromType(field.Type)
			if err != nil {
				return nil, err
			}
		}

		schema.Properties[formName] = fieldSchema
	}

	return &openapi3.SchemaRef{Value: schema}, nil
}

// createResponseSchema creates schema for response type.
func (g *Generator) createResponseSchema(responseType reflect.Type) (*openapi3.SchemaRef, error) {
	return g.createSchemaFromType(responseType)
}

// createSchemaFromType creates OpenAPI schema from Go type.
func (g *Generator) createSchemaFromType(t reflect.Type) (*openapi3.SchemaRef, error) {
	schema := &openapi3.Schema{}

	switch t.Kind() {
	case reflect.String:
		schema.Type = &openapi3.Types{"string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		schema.Type = &openapi3.Types{"integer"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema.Type = &openapi3.Types{"integer"}
	case reflect.Float32, reflect.Float64:
		schema.Type = &openapi3.Types{"number"}
	case reflect.Bool:
		schema.Type = &openapi3.Types{"boolean"}
	case reflect.Struct:
		schema.Type = &openapi3.Types{"object"}
		schema.Properties = make(map[string]*openapi3.SchemaRef)

		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)

			// Skip unexported fields
			if !field.IsExported() {
				continue
			}

			jsonName := field.Tag.Get("json")
			if jsonName == "" || jsonName == "-" {
				continue
			}

			// Handle omitempty
			parts := strings.Split(jsonName, ",")
			fieldName := parts[0]
			omitempty := len(parts) > 1 && parts[1] == "omitempty"

			fieldSchema, err := g.createSchemaFromType(field.Type)
			if err != nil {
				return nil, err
			}

			schema.Properties[fieldName] = fieldSchema

			// Add to required if not omitempty
			if !omitempty {
				schema.Required = append(schema.Required, fieldName)
			}
		}
	case reflect.Slice:
		schema.Type = &openapi3.Types{"array"}
		itemSchema, err := g.createSchemaFromType(t.Elem())
		if err != nil {
			return nil, err
		}
		schema.Items = itemSchema
	case reflect.Map:
		schema.Type = &openapi3.Types{"object"}
		trueVal := true
		schema.AdditionalProperties = openapi3.AdditionalProperties{Has: &trueVal}
	case reflect.Array:
		schema.Type = &openapi3.Types{"array"}
		itemSchema, err := g.createSchemaFromType(t.Elem())
		if err != nil {
			return nil, err
		}
		schema.Items = itemSchema
	case reflect.Ptr:
		// Dereference pointer and create schema for underlying type
		return g.createSchemaFromType(t.Elem())
	case reflect.Interface:
		// Use generic object type for interfaces
		schema.Type = &openapi3.Types{"object"}
	case reflect.Invalid, reflect.Uintptr, reflect.Complex64, reflect.Complex128,
		reflect.Chan, reflect.Func, reflect.UnsafePointer:
		// Unsupported types - fallback to string
		schema.Type = &openapi3.Types{"string"}
	default:
		schema.Type = &openapi3.Types{"string"} // Fallback
	}

	return &openapi3.SchemaRef{Value: schema}, nil
}

// applyValidationToSchema applies validation constraints to schema.
func (g *Generator) applyValidationToSchema(schemaRef *openapi3.SchemaRef, validate string) {
	if validate == "" || schemaRef.Value == nil {
		return
	}

	schema := schemaRef.Value
	rules := strings.Split(validate, ",")

	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		g.applyValidationRule(schema, rule)
	}
}

// applyValidationRule applies a single validation rule to the schema.
func (g *Generator) applyValidationRule(schema *openapi3.Schema, rule string) {
	if strings.HasPrefix(rule, "min=") {
		g.applyMinValidation(schema, rule)
	} else if strings.HasPrefix(rule, "max=") {
		g.applyMaxValidation(schema, rule)
	} else if rule == "email" {
		schema.Format = "email"
	} else if rule == "uuid" {
		schema.Format = "uuid"
	}
}

// applyMinValidation applies minimum value validation.
func (g *Generator) applyMinValidation(schema *openapi3.Schema, rule string) {
	minVal, err := strconv.Atoi(rule[4:])
	if err != nil {
		return
	}

	if schema.Type == nil || len(*schema.Type) == 0 {
		return
	}

	schemaType := (*schema.Type)[0]
	switch schemaType {
	case "string":
		if minVal >= 0 {
			schema.MinLength = uint64(minVal)
		}
	case "integer", "number":
		minFloat := float64(minVal)
		schema.Min = &minFloat
	}
}

// applyMaxValidation applies maximum value validation.
func (g *Generator) applyMaxValidation(schema *openapi3.Schema, rule string) {
	maxVal, err := strconv.Atoi(rule[4:])
	if err != nil {
		return
	}

	if schema.Type == nil || len(*schema.Type) == 0 {
		return
	}

	schemaType := (*schema.Type)[0]
	switch schemaType {
	case "string":
		if maxVal >= 0 {
			maxPtr := uint64(maxVal)
			schema.MaxLength = &maxPtr
		}
	case "integer", "number":
		maxFloat := float64(maxVal)
		schema.Max = &maxFloat
	}
}

// parseDefaultValue parses default value based on type.
func (g *Generator) parseDefaultValue(defaultValue string, t reflect.Type) interface{} {
	switch t.Kind() {
	case reflect.String:
		return defaultValue
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if val, err := strconv.ParseInt(defaultValue, 10, 64); err == nil {
			return val
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if val, err := strconv.ParseUint(defaultValue, 10, 64); err == nil {
			return val
		}
	case reflect.Float32, reflect.Float64:
		if val, err := strconv.ParseFloat(defaultValue, 64); err == nil {
			return val
		}
	case reflect.Bool:
		if val, err := strconv.ParseBool(defaultValue); err == nil {
			return val
		}
	case reflect.Ptr:
		// Dereference pointer and parse for underlying type
		return g.parseDefaultValue(defaultValue, t.Elem())
	case reflect.Invalid, reflect.Uintptr, reflect.Complex64, reflect.Complex128,
		reflect.Array, reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Slice, reflect.Struct, reflect.UnsafePointer:
		// Unsupported types for default values - return as string
		return defaultValue
	}

	return defaultValue
}

// parseOpenAPIComment parses OpenAPI metadata from a comment.
func parseOpenAPIComment(comment string) map[string]string {
	// Remove leading // and whitespace
	comment = strings.TrimSpace(comment)
	if !strings.HasPrefix(comment, "//openapi:") {
		return nil
	}

	// Extract the content after "//openapi:"
	content := strings.TrimPrefix(comment, "//openapi:")
	if content == "" {
		return make(map[string]string)
	}

	result := make(map[string]string)

	// Split by comma and parse key=value pairs
	pairs := strings.Split(content, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		const keyValueParts = 2
		parts := strings.SplitN(pair, "=", keyValueParts)
		if len(parts) == keyValueParts {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			result[key] = value
		}
	}

	return result
}

// GenerateJSON generates JSON representation of OpenAPI spec.
func (g *Generator) GenerateJSON(spec *openapi3.T) ([]byte, error) {
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OpenAPI spec to JSON: %w", err)
	}

	return data, nil
}

// GenerateYAML generates YAML representation of OpenAPI spec.
func (g *Generator) GenerateYAML(spec *openapi3.T) ([]byte, error) {
	data, err := yaml.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OpenAPI spec to YAML: %w", err)
	}

	return data, nil
}

// applyMiddlewareSchemaTransformations applies middleware schema transformations to the base response schema.
func (g *Generator) applyMiddlewareSchemaTransformations(
	ctx context.Context,
	entries []typedhttp.MiddlewareEntry,
	baseSchema *openapi3.SchemaRef,
) (*openapi3.SchemaRef, error) {
	currentSchema := baseSchema

	// Apply transformations in middleware execution order
	// Post-middleware runs in reverse order, so we apply transformations in reverse
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		if modifier, ok := entry.Middleware.(typedhttp.ResponseSchemaModifier); ok {
			transformedSchema, err := modifier.ModifyResponseSchema(ctx, currentSchema)
			if err != nil {
				return nil, fmt.Errorf("middleware %q failed to modify schema: %w",
					entry.Config.Name, err)
			}
			currentSchema = transformedSchema
		}
	}

	return currentSchema, nil
}

// hasEnvelopeMiddleware checks if the middleware chain contains envelope middleware.
func (g *Generator) hasEnvelopeMiddleware(entries []typedhttp.MiddlewareEntry) bool {
	for _, entry := range entries {
		// Check if middleware is a response envelope middleware
		if _, ok := entry.Middleware.(*typedhttp.ResponseEnvelopeMiddleware[any]); ok {
			return true
		}
		// Check for interface implementation
		if _, ok := entry.Middleware.(interface {
			ModifyResponseSchema(context.Context, *openapi3.SchemaRef) (*openapi3.SchemaRef, error)
		}); ok {
			// This is a basic check - could be enhanced to specifically detect envelope patterns
			return true
		}
	}
	return false
}

// addEnvelopeErrorResponses adds standard error responses for envelope middleware.
func (g *Generator) addEnvelopeErrorResponses(operation *openapi3.Operation) {
	// Standard envelope error schema
	errorEnvelopeSchema := &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			Type:        &openapi3.Types{"object"},
			Description: "Error response envelope",
			Properties: map[string]*openapi3.SchemaRef{
				"data": {
					Value: &openapi3.Schema{
						Type:     &openapi3.Types{"null"},
						Nullable: true,
					},
				},
				"error": {
					Value: &openapi3.Schema{
						Type:        &openapi3.Types{"string"},
						Description: "Error message",
					},
				},
				"success": {
					Value: &openapi3.Schema{
						Type:    &openapi3.Types{"boolean"},
						Enum:    []interface{}{false},
						Default: false,
					},
				},
			},
			Required: []string{"success", "error"},
		},
	}

	// Add error responses
	operation.Responses.Set("400", &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Description: stringPtr("Bad Request"),
			Content: map[string]*openapi3.MediaType{
				"application/json": {
					Schema: errorEnvelopeSchema,
				},
			},
		},
	})

	operation.Responses.Set("401", &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Description: stringPtr("Unauthorized"),
			Content: map[string]*openapi3.MediaType{
				"application/json": {
					Schema: errorEnvelopeSchema,
				},
			},
		},
	})

	operation.Responses.Set("404", &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Description: stringPtr("Not Found"),
			Content: map[string]*openapi3.MediaType{
				"application/json": {
					Schema: errorEnvelopeSchema,
				},
			},
		},
	})

	operation.Responses.Set("500", &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Description: stringPtr("Internal Server Error"),
			Content: map[string]*openapi3.MediaType{
				"application/json": {
					Schema: errorEnvelopeSchema,
				},
			},
		},
	})
}

// stringPtr returns a pointer to a string value.
func stringPtr(s string) *string {
	return &s
}
