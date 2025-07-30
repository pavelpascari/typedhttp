package typedhttp

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// PathDecoder implements RequestDecoder for URL path parameters.
type PathDecoder[T any] struct {
	validator *validator.Validate
}

// NewPathDecoder creates a new path parameter decoder.
func NewPathDecoder[T any](validator *validator.Validate) *PathDecoder[T] {
	return &PathDecoder[T]{
		validator: validator,
	}
}

// Decode decodes path parameters into the target type using reflection.
func (d *PathDecoder[T]) Decode(r *http.Request) (T, error) {
	var result T

	// Use reflection to map path parameters to struct fields
	resultValue := reflect.ValueOf(&result).Elem()
	resultType := resultValue.Type()

	for i := 0; i < resultType.NumField(); i++ {
		field := resultType.Field(i)
		fieldValue := resultValue.Field(i)

		// Skip unexported fields
		if !fieldValue.CanSet() {
			continue
		}

		// Get the path parameter name from struct tag
		pathName := field.Tag.Get("path")
		if pathName == "" {
			continue // Only process fields with path tags
		}

		// Extract path parameter from URL
		pathValue := extractPathParam(r.URL.Path, pathName)
		if pathValue == "" {
			continue
		}

		// Set the field value based on its type
		if err := setFieldValueFromString(fieldValue, pathValue); err != nil {
			return result, fmt.Errorf("failed to set field %s: %w", field.Name, err)
		}
	}

	// Perform validation if validator is available
	if d.validator != nil {
		if err := d.validator.Struct(result); err != nil {
			validationErrors := make(map[string]string)
			if validatorErrs, ok := err.(validator.ValidationErrors); ok {
				for _, validatorErr := range validatorErrs {
					field := strings.ToLower(validatorErr.Field())
					validationErrors[field] = validatorErr.Tag()
				}
			}
			return result, NewValidationError("Validation failed", validationErrors)
		}
	}

	return result, nil
}

// ContentTypes returns the supported content types for path decoding.
func (d *PathDecoder[T]) ContentTypes() []string {
	return []string{"*/*"} // Path parameters work with any content type
}

// extractPathParam extracts a path parameter from a URL path.
// This is a simple implementation that works with the {param} format.
func extractPathParam(path, paramName string) string {
	// This is a basic implementation
	// In a real router, you'd use the router's path matching capabilities

	// For now, we'll extract based on the paramName being the last segment
	// This is a simplified approach for the example
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) > 0 {
		// Return the last segment as the parameter value
		// This works for simple cases like /users/{id}
		return segments[len(segments)-1]
	}

	return ""
}

// SourceType represents the type of data source for request fields.
type SourceType string

const (
	SourcePath   SourceType = "path"
	SourceQuery  SourceType = "query"
	SourceHeader SourceType = "header"
	SourceCookie SourceType = "cookie"
	SourceForm   SourceType = "form"
	SourceJSON   SourceType = "json"
)

// FieldSource represents a single source for a request field.
type FieldSource struct {
	Type      SourceType
	Name      string
	Default   string
	Transform string
	Format    string
	Required  bool
}

// FieldExtractor contains information about how to extract a single field from the request.
type FieldExtractor struct {
	FieldName  string
	FieldType  reflect.Type
	Sources    []FieldSource
	Precedence []SourceType
	Validation string
}

// CombinedDecoder combines multiple decoders to handle different types of request data with precedence rules.
type CombinedDecoder[T any] struct {
	pathDecoder   *PathDecoder[T]
	queryDecoder  *QueryDecoder[T]
	headerDecoder *HeaderDecoder[T]
	cookieDecoder *CookieDecoder[T]
	formDecoder   *FormDecoder[T]
	jsonDecoder   *JSONDecoder[T]
	extractors    []FieldExtractor // Pre-computed field extraction rules
	validator     *validator.Validate
}

// NewCombinedDecoder creates a decoder that can handle multiple data sources.
func NewCombinedDecoder[T any](validator *validator.Validate) *CombinedDecoder[T] {
	decoder := &CombinedDecoder[T]{
		pathDecoder:   NewPathDecoder[T](validator),
		queryDecoder:  NewQueryDecoder[T](validator),
		headerDecoder: NewHeaderDecoder[T](validator),
		cookieDecoder: NewCookieDecoder[T](validator),
		formDecoder:   NewFormDecoder[T](validator),
		jsonDecoder:   NewJSONDecoder[T](validator),
		validator:     validator,
	}

	// Pre-compute field extraction rules
	decoder.extractors = decoder.buildFieldExtractors()

	return decoder
}

// buildFieldExtractors analyzes the target type and builds extraction rules.
func (d *CombinedDecoder[T]) buildFieldExtractors() []FieldExtractor {
	var result T
	resultType := reflect.TypeOf(result)

	var extractors []FieldExtractor

	for i := 0; i < resultType.NumField(); i++ {
		field := resultType.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		extractor := FieldExtractor{
			FieldName: field.Name,
			FieldType: field.Type,
			Sources:   []FieldSource{},
		}

		// Check for each source type
		if pathName := field.Tag.Get("path"); pathName != "" {
			extractor.Sources = append(extractor.Sources, FieldSource{
				Type: SourcePath,
				Name: pathName,
			})
		}

		if queryName := field.Tag.Get("query"); queryName != "" {
			extractor.Sources = append(extractor.Sources, FieldSource{
				Type:    SourceQuery,
				Name:    queryName,
				Default: field.Tag.Get("default"),
			})
		}

		if headerName := field.Tag.Get("header"); headerName != "" {
			extractor.Sources = append(extractor.Sources, FieldSource{
				Type:      SourceHeader,
				Name:      headerName,
				Default:   field.Tag.Get("default"),
				Transform: field.Tag.Get("transform"),
				Format:    field.Tag.Get("format"),
			})
		}

		if cookieName := field.Tag.Get("cookie"); cookieName != "" {
			extractor.Sources = append(extractor.Sources, FieldSource{
				Type:    SourceCookie,
				Name:    cookieName,
				Default: field.Tag.Get("default"),
			})
		}

		if formName := field.Tag.Get("form"); formName != "" {
			extractor.Sources = append(extractor.Sources, FieldSource{
				Type:    SourceForm,
				Name:    formName,
				Default: field.Tag.Get("default"),
			})
		}

		if jsonName := field.Tag.Get("json"); jsonName != "" {
			extractor.Sources = append(extractor.Sources, FieldSource{
				Type: SourceJSON,
				Name: jsonName,
			})
		}

		// Parse precedence rules
		if precedenceStr := field.Tag.Get("precedence"); precedenceStr != "" {
			precedenceParts := strings.Split(precedenceStr, ",")
			for _, part := range precedenceParts {
				part = strings.TrimSpace(part)
				extractor.Precedence = append(extractor.Precedence, SourceType(part))
			}
		} else {
			// Default precedence: path > header > cookie > query > form > json
			extractor.Precedence = []SourceType{
				SourcePath, SourceHeader, SourceCookie, SourceQuery, SourceForm, SourceJSON,
			}
		}

		// Store validation rules
		extractor.Validation = field.Tag.Get("validate")

		// Only add extractor if it has sources
		if len(extractor.Sources) > 0 {
			extractors = append(extractors, extractor)
		}
	}

	return extractors
}

// Decode decodes request data from multiple sources using precedence rules.
func (d *CombinedDecoder[T]) Decode(r *http.Request) (T, error) {
	var result T

	// Use reflection to set field values
	resultValue := reflect.ValueOf(&result).Elem()

	// Extract data for each field using the pre-computed extractors
	for _, extractor := range d.extractors {
		fieldValue := resultValue.FieldByName(extractor.FieldName)
		if !fieldValue.CanSet() {
			continue
		}

		// Try each source in precedence order until we find a value
		var extractedValue string
		var sourceFound SourceType

		for _, sourceType := range extractor.Precedence {
			// Find the source configuration for this type
			var sourceConfig *FieldSource
			for _, src := range extractor.Sources {
				if src.Type == sourceType {
					sourceConfig = &src
					break
				}
			}

			if sourceConfig == nil {
				continue // This field doesn't have this source type
			}

			// Extract value from the appropriate source
			value, err := d.extractFromSource(r, sourceType, sourceConfig.Name)
			if err != nil {
				continue // Try next source
			}

			if value != "" {
				extractedValue = value
				sourceFound = sourceType
				break
			}
		}

		// If no value found, try default
		if extractedValue == "" {
			for _, src := range extractor.Sources {
				if src.Default != "" {
					extractedValue = handleDefaultValue(src.Default)
					break
				}
			}
		}

		// Skip if still no value
		if extractedValue == "" {
			continue
		}

		// Apply transformations if specified
		for _, src := range extractor.Sources {
			if src.Type == sourceFound && src.Transform != "" {
				transformed, err := applyTransformation(src.Transform, extractedValue)
				if err != nil {
					return result, fmt.Errorf("failed to transform field %s: %w", extractor.FieldName, err)
				}
				extractedValue = transformed
				break
			}
		}

		// Apply format parsing if specified
		for _, src := range extractor.Sources {
			if src.Type == sourceFound && src.Format != "" {
				formatted, err := applyFormat(src.Format, extractedValue, extractor.FieldType)
				if err != nil {
					return result, fmt.Errorf("failed to format field %s: %w", extractor.FieldName, err)
				}
				fieldValue.Set(reflect.ValueOf(formatted))
				continue
			}
		}

		// Set the field value based on its type
		if err := setFieldValueFromString(fieldValue, extractedValue); err != nil {
			return result, fmt.Errorf("failed to set field %s: %w", extractor.FieldName, err)
		}
	}

	// Handle special cases for file uploads and complex JSON from form data
	if err := d.handleSpecialCases(r, &result); err != nil {
		return result, err
	}

	// Validate the final result if we have a validator
	if d.validator != nil {
		if err := d.validator.Struct(result); err != nil {
			validationErrors := make(map[string]string)
			if validatorErrs, ok := err.(validator.ValidationErrors); ok {
				for _, validatorErr := range validatorErrs {
					field := strings.ToLower(validatorErr.Field())
					validationErrors[field] = validatorErr.Tag()
				}
			}
			return result, NewValidationError("Multi-source validation failed", validationErrors)
		}
	}

	return result, nil
}

// extractFromSource extracts a value from a specific source type.
func (d *CombinedDecoder[T]) extractFromSource(r *http.Request, sourceType SourceType, name string) (string, error) {
	switch sourceType {
	case SourcePath:
		return extractPathParam(r.URL.Path, name), nil

	case SourceQuery:
		return r.URL.Query().Get(name), nil

	case SourceHeader:
		return r.Header.Get(name), nil

	case SourceCookie:
		if cookie, err := r.Cookie(name); err == nil {
			return cookie.Value, nil
		}
		return "", nil

	case SourceForm:
		if err := r.ParseForm(); err != nil {
			return "", err
		}
		return r.FormValue(name), nil

	case SourceJSON:
		// For JSON, we need to decode the entire body and extract the field
		// This is more complex and should be handled separately
		return "", fmt.Errorf("JSON source extraction not implemented in extractFromSource")

	default:
		return "", fmt.Errorf("unknown source type: %s", sourceType)
	}
}

// handleSpecialCases handles file uploads and complex JSON extraction.
func (d *CombinedDecoder[T]) handleSpecialCases(r *http.Request, result *T) error {
	resultValue := reflect.ValueOf(result).Elem()
	resultType := resultValue.Type()

	// Check if we need to handle JSON body or file uploads
	needsJSON := false
	needsForm := false

	for i := 0; i < resultType.NumField(); i++ {
		field := resultType.Field(i)

		if field.Tag.Get("json") != "" {
			needsJSON = true
		}

		if field.Tag.Get("form") != "" {
			needsForm = true
		}

		// Check for file upload fields
		if field.Type == reflect.TypeOf((*multipart.FileHeader)(nil)) ||
			field.Type == reflect.TypeOf([]*multipart.FileHeader{}) {
			needsForm = true
		}
	}

	// Handle JSON body if needed
	if needsJSON && r.Body != nil && r.ContentLength > 0 {
		contentType := r.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/json") {
			if jsonResult, err := d.jsonDecoder.Decode(r); err == nil {
				*result = mergeStructs(*result, jsonResult)
			}
		}
	}

	// Handle form data if needed (including file uploads)
	if needsForm {
		if formResult, err := d.formDecoder.Decode(r); err == nil {
			*result = mergeStructs(*result, formResult)
		}
	}

	return nil
}

// ContentTypes returns all supported content types.
func (d *CombinedDecoder[T]) ContentTypes() []string {
	return []string{"application/json", "application/x-www-form-urlencoded", "*/*"}
}

// mergeStructs merges two structs of the same type, preferring non-zero values from the second struct.
func mergeStructs[T any](dst, src T) T {
	dstValue := reflect.ValueOf(&dst).Elem()
	srcValue := reflect.ValueOf(src)

	for i := 0; i < dstValue.NumField(); i++ {
		dstField := dstValue.Field(i)
		srcField := srcValue.Field(i)

		if !dstField.CanSet() {
			continue
		}

		// If source field has a non-zero value, use it
		if !srcField.IsZero() {
			dstField.Set(srcField)
		}
	}

	return dst
}
