package typedhttp

import (
	"fmt"
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
		if err := setFieldValue(fieldValue, pathValue); err != nil {
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

// CombinedDecoder combines multiple decoders to handle different types of request data.
type CombinedDecoder[T any] struct {
	pathDecoder  *PathDecoder[T]
	queryDecoder *QueryDecoder[T]
	jsonDecoder  *JSONDecoder[T]
}

// NewCombinedDecoder creates a decoder that can handle path, query, and JSON data.
func NewCombinedDecoder[T any](validator *validator.Validate) *CombinedDecoder[T] {
	return &CombinedDecoder[T]{
		pathDecoder:  NewPathDecoder[T](validator),
		queryDecoder: NewQueryDecoder[T](validator),
		jsonDecoder:  NewJSONDecoder[T](validator),
	}
}

// Decode decodes request data from multiple sources (path, query, body).
func (d *CombinedDecoder[T]) Decode(r *http.Request) (T, error) {
	var result T
	
	// Start with path parameters
	if pathResult, err := d.pathDecoder.Decode(r); err == nil {
		result = pathResult
	}
	
	// Merge query parameters
	if queryResult, err := d.queryDecoder.Decode(r); err == nil {
		result = mergeStructs(result, queryResult)
	}
	
	// If there's a body and it's JSON, merge that too
	if r.Body != nil && r.ContentLength > 0 {
		contentType := r.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/json") {
			if jsonResult, err := d.jsonDecoder.Decode(r); err != nil {
				return result, err // Return JSON decode error immediately
			} else {
				result = mergeStructs(result, jsonResult)
			}
		}
	}
	
	// Validate the final result if we have a validator
	if d.jsonDecoder.validator != nil {
		if err := d.jsonDecoder.validator.Struct(result); err != nil {
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