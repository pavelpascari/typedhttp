package typedhttp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
)

// JSONDecoder implements RequestDecoder for JSON content.
type JSONDecoder[T any] struct {
	validator *validator.Validate
}

// NewJSONDecoder creates a new JSON decoder with optional validation.
func NewJSONDecoder[T any](validator *validator.Validate) *JSONDecoder[T] {
	return &JSONDecoder[T]{
		validator: validator,
	}
}

// Decode decodes a JSON request body into the target type.
func (d *JSONDecoder[T]) Decode(r *http.Request) (T, error) {
	var result T

	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		return result, fmt.Errorf("invalid JSON: %w", err)
	}

	// Perform validation if validator is available
	if d.validator != nil {
		if err := d.validator.Struct(result); err != nil {
			// Convert validator errors to ValidationError
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

// ContentTypes returns the supported content types for JSON decoding.
func (d *JSONDecoder[T]) ContentTypes() []string {
	return []string{"application/json"}
}

// QueryDecoder implements RequestDecoder for URL query parameters.
type QueryDecoder[T any] struct {
	validator *validator.Validate
}

// NewQueryDecoder creates a new query parameter decoder.
func NewQueryDecoder[T any](validator *validator.Validate) *QueryDecoder[T] {
	return &QueryDecoder[T]{
		validator: validator,
	}
}

// Decode decodes query parameters into the target type using reflection.
func (d *QueryDecoder[T]) Decode(r *http.Request) (T, error) {
	var result T

	// Use reflection to map query parameters to struct fields
	resultValue := reflect.ValueOf(&result).Elem()
	resultType := resultValue.Type()

	for i := 0; i < resultType.NumField(); i++ {
		field := resultType.Field(i)
		fieldValue := resultValue.Field(i)

		// Skip unexported fields
		if !fieldValue.CanSet() {
			continue
		}

		// Get the query parameter name from struct tag
		queryName := field.Tag.Get("query")
		if queryName == "" {
			queryName = strings.ToLower(field.Name)
		}

		// Get the value from query parameters
		queryValue := r.URL.Query().Get(queryName)
		if queryValue == "" {
			// Check for default value
			if defaultValue := field.Tag.Get("default"); defaultValue != "" {
				queryValue = defaultValue
			} else {
				continue
			}
		}

		// Set the field value based on its type
		if err := setFieldValue(fieldValue, queryValue); err != nil {
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

// ContentTypes returns the supported content types for query decoding.
func (d *QueryDecoder[T]) ContentTypes() []string {
	return []string{"application/x-www-form-urlencoded"}
}

// setFieldValue sets a reflect.Value based on a string value.
func setFieldValue(fieldValue reflect.Value, value string) error {
	switch fieldValue.Kind() {
	case reflect.String:
		fieldValue.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid integer value: %s", value)
		}
		fieldValue.SetInt(intValue)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintValue, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid unsigned integer value: %s", value)
		}
		fieldValue.SetUint(uintValue)
	case reflect.Float32, reflect.Float64:
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid float value: %s", value)
		}
		fieldValue.SetFloat(floatValue)
	case reflect.Bool:
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value: %s", value)
		}
		fieldValue.SetBool(boolValue)
	default:
		return fmt.Errorf("unsupported field type: %s", fieldValue.Kind())
	}
	return nil
}

// JSONEncoder implements ResponseEncoder for JSON content.
type JSONEncoder[T any] struct{}

// NewJSONEncoder creates a new JSON encoder.
func NewJSONEncoder[T any]() *JSONEncoder[T] {
	return &JSONEncoder[T]{}
}

// Encode encodes the response data as JSON and writes it to the response writer.
func (e *JSONEncoder[T]) Encode(w http.ResponseWriter, data T, statusCode int) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	return json.NewEncoder(w).Encode(data)
}

// ContentType returns the content type for JSON encoding.
func (e *JSONEncoder[T]) ContentType() string {
	return "application/json"
}

// EnvelopeEncoder wraps responses in a standard envelope format.
type EnvelopeEncoder[T any] struct {
	encoder ResponseEncoder[EnvelopeResponse[T]]
}

// EnvelopeResponse represents a standard response envelope.
type EnvelopeResponse[T any] struct {
	Data      T      `json:"data,omitempty"`
	Error     string `json:"error,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// NewEnvelopeEncoder creates a new envelope encoder.
func NewEnvelopeEncoder[T any](encoder ResponseEncoder[EnvelopeResponse[T]]) *EnvelopeEncoder[T] {
	return &EnvelopeEncoder[T]{
		encoder: encoder,
	}
}

// Encode encodes the response data in an envelope format.
func (e *EnvelopeEncoder[T]) Encode(w http.ResponseWriter, data T, statusCode int) error {
	envelope := EnvelopeResponse[T]{
		Data: data,
	}

	// Add request ID if available from context
	if requestID := w.Header().Get("X-Request-ID"); requestID != "" {
		envelope.RequestID = requestID
	}

	return e.encoder.Encode(w, envelope, statusCode)
}

// ContentType returns the content type of the underlying encoder.
func (e *EnvelopeEncoder[T]) ContentType() string {
	return e.encoder.ContentType()
}
