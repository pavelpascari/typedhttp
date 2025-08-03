package typedhttp

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

// Error variables for static error handling.
var (
	ErrJSONSourceNotImplemented = errors.New("JSON source extraction not implemented in extractFromSource")
	ErrUnknownSourceType        = errors.New("unknown source type")
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
			var validatorErrs validator.ValidationErrors
			if errors.As(err, &validatorErrs) {
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
func extractPathParam(path, _ string) string {
	// This is a basic implementation
	// In a real router, you'd use the router's path matching capabilities

	// For now, we'll extract the last segment
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

	// Handle case where T is interface{} or similar
	if resultType == nil || resultType.Kind() != reflect.Struct {
		return []FieldExtractor{}
	}

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

	if err := d.extractFieldsFromSources(r, &result); err != nil {
		return result, err
	}

	if err := d.handleSpecialCases(r, &result); err != nil {
		return result, err
	}

	if err := d.validateCombinedResult(result); err != nil {
		return result, err
	}

	return result, nil
}

// extractFieldsFromSources extracts data for each field using the pre-computed extractors.
func (d *CombinedDecoder[T]) extractFieldsFromSources(r *http.Request, result *T) error {
	resultValue := reflect.ValueOf(result).Elem()

	for _, extractor := range d.extractors {
		fieldValue := resultValue.FieldByName(extractor.FieldName)
		if !fieldValue.CanSet() {
			continue
		}

		if err := d.processFieldExtractor(r, &extractor, fieldValue); err != nil {
			return err
		}
	}

	return nil
}

// processFieldExtractor processes a single field extractor.
func (d *CombinedDecoder[T]) processFieldExtractor(
	r *http.Request, extractor *FieldExtractor, fieldValue reflect.Value,
) error {
	extractedValue, sourceFound := d.extractValueWithPrecedence(r, extractor)

	if extractedValue == "" {
		return nil
	}

	processedValue, transformedValue, err := d.processExtractedValue(extractor, extractedValue, sourceFound)
	if err != nil {
		return err
	}

	if processedValue != nil {
		fieldValue.Set(reflect.ValueOf(processedValue))

		return nil
	}

	// Use the transformed value if transformation occurred, otherwise use original
	valueToUse := transformedValue
	if valueToUse == "" {
		valueToUse = extractedValue
	}

	if err := setFieldValueFromString(fieldValue, valueToUse); err != nil {
		return fmt.Errorf("failed to set field %s: %w", extractor.FieldName, err)
	}

	return nil
}

// extractValueWithPrecedence tries each source in precedence order.
func (d *CombinedDecoder[T]) extractValueWithPrecedence(
	r *http.Request, extractor *FieldExtractor,
) (string, SourceType) {
	for _, sourceType := range extractor.Precedence {
		sourceConfig := d.findSourceConfig(extractor.Sources, sourceType)
		if sourceConfig == nil {
			continue
		}

		value, err := d.extractFromSource(r, sourceType, sourceConfig.Name)
		if err != nil {
			continue
		}

		if value != "" {
			return value, sourceType
		}
	}

	// Try default values
	for _, src := range extractor.Sources {
		if src.Default != "" {
			return handleDefaultValue(src.Default), ""
		}
	}

	return "", ""
}

// findSourceConfig finds the source configuration for a given type.
func (d *CombinedDecoder[T]) findSourceConfig(sources []FieldSource, sourceType SourceType) *FieldSource {
	for _, src := range sources {
		if src.Type == sourceType {
			return &src
		}
	}

	return nil
}

// processExtractedValue applies transformations and formats to extracted values.
// Returns (processedValue, transformedString, error).
func (d *CombinedDecoder[T]) processExtractedValue(
	extractor *FieldExtractor, extractedValue string, sourceFound SourceType,
) (processedValue interface{}, transformedString string, err error) {
	originalValue := extractedValue

	// Apply transformations if specified
	transformedValue, err := d.applyTransformationForSource(extractor.Sources, sourceFound, extractedValue)
	if err != nil {
		return nil, "", fmt.Errorf("failed to transform field %s: %w", extractor.FieldName, err)
	}

	// Apply format parsing if specified
	formatted, err := d.applyFormatForSource(extractor.Sources, sourceFound, transformedValue, extractor.FieldType)
	if err != nil {
		return nil, "", fmt.Errorf("failed to format field %s: %w", extractor.FieldName, err)
	}

	if formatted != nil {
		return formatted, transformedValue, nil
	}

	// Check if we need to handle special types after transformation
	if extractor.FieldType == reflect.TypeOf(net.IP{}) {
		ip := net.ParseIP(transformedValue)
		if ip == nil {
			return nil, "", fmt.Errorf("%w: %s", ErrInvalidIPAddress, transformedValue)
		}

		return ip, transformedValue, nil
	}

	if extractor.FieldType == reflect.TypeOf(time.Time{}) {
		// Try to parse with standard formats
		formats := []string{
			time.RFC3339,
			"2006-01-02 15:04:05",
			"2006-01-02",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, transformedValue); err == nil {
				return t, transformedValue, nil
			}
		}

		return nil, "", fmt.Errorf("%w: %s", ErrInvalidTimeValue, transformedValue)
	}

	// Return the transformed value for basic type processing
	if transformedValue != originalValue {
		return nil, transformedValue, nil
	}

	return nil, "", nil
}

// applyTransformationForSource applies transformation for a specific source.
func (d *CombinedDecoder[T]) applyTransformationForSource(
	sources []FieldSource, sourceFound SourceType, value string,
) (string, error) {
	for _, src := range sources {
		if src.Type == sourceFound && src.Transform != "" {
			return applyTransformation(src.Transform, value)
		}
	}

	return value, nil
}

// applyFormatForSource applies format parsing for a specific source.
func (d *CombinedDecoder[T]) applyFormatForSource(
	sources []FieldSource, sourceFound SourceType, value string, fieldType reflect.Type,
) (interface{}, error) {
	for _, src := range sources {
		if src.Type == sourceFound && src.Format != "" {
			return applyFormat(src.Format, value, fieldType)
		}
	}
	//nolint:nilnil
	return nil, nil
}

// validateCombinedResult validates the final combined result.
func (d *CombinedDecoder[T]) validateCombinedResult(result T) error {
	if d.validator == nil {
		return nil
	}

	if err := d.validator.Struct(result); err != nil {
		validationErrors := make(map[string]string)
		var validatorErrs validator.ValidationErrors
		if errors.As(err, &validatorErrs) {
			for _, validatorErr := range validatorErrs {
				field := strings.ToLower(validatorErr.Field())
				validationErrors[field] = validatorErr.Tag()
			}
		}

		return NewValidationError("Multi-source validation failed", validationErrors)
	}

	return nil
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
			return "", fmt.Errorf("failed to parse form: %w", err)
		}

		return r.FormValue(name), nil

	case SourceJSON:
		// For JSON, we need to decode the entire body and extract the field
		// This is more complex and should be handled separately
		return "", ErrJSONSourceNotImplemented

	default:
		return "", fmt.Errorf("%w: %s", ErrUnknownSourceType, sourceType)
	}
}

// handleSpecialCases handles file uploads and complex JSON extraction.
func (d *CombinedDecoder[T]) handleSpecialCases(r *http.Request, result *T) error {
	resultValue := reflect.ValueOf(result).Elem()
	resultType := resultValue.Type()

	// Handle case where T is interface{} or similar
	if resultType.Kind() != reflect.Struct {
		return nil
	}

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
			if jsonResult, err := d.jsonDecoder.Decode(r); err != nil {
				return err // Propagate JSON parsing errors
			} else {
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
