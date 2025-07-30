package typedhttp

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

// Error variables for static error handling.
var (
	ErrInvalidIPAddress      = errors.New("invalid IP address")
	ErrInvalidTimeValue      = errors.New("invalid time value")
	ErrUnknownTransformation = errors.New("unknown transformation")
	ErrFormatNotSupported    = errors.New("format not supported for type")
	ErrInvalidUnixTimestamp  = errors.New("invalid unix timestamp")
)

// HeaderDecoder implements RequestDecoder for HTTP headers.
type HeaderDecoder[T any] struct {
	validator *validator.Validate
}

// NewHeaderDecoder creates a new HTTP header decoder.
func NewHeaderDecoder[T any](validator *validator.Validate) *HeaderDecoder[T] {
	return &HeaderDecoder[T]{
		validator: validator,
	}
}

// Decode decodes HTTP headers into the target type using reflection.
func (d *HeaderDecoder[T]) Decode(r *http.Request) (T, error) {
	var result T

	if err := d.processHeaderFields(r, &result); err != nil {
		return result, err
	}

	if err := d.validateHeaderResult(result); err != nil {
		return result, err
	}

	return result, nil
}

// processHeaderFields processes all header fields using reflection.
func (d *HeaderDecoder[T]) processHeaderFields(r *http.Request, result *T) error {
	resultValue := reflect.ValueOf(result).Elem()
	resultType := resultValue.Type()

	for i := 0; i < resultType.NumField(); i++ {
		field := resultType.Field(i)
		fieldValue := resultValue.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		headerName := field.Tag.Get("header")
		if headerName == "" {
			continue
		}

		if err := d.processHeaderField(r, &field, fieldValue, headerName); err != nil {
			return err
		}
	}

	return nil
}

// processHeaderField processes a single header field.
func (d *HeaderDecoder[T]) processHeaderField(
	r *http.Request, field *reflect.StructField, fieldValue reflect.Value, headerName string,
) error {
	headerValue := d.getHeaderValue(r, headerName, field.Tag.Get("default"))
	if headerValue == "" {
		return nil
	}

	processedValue, transformedValue, err := d.processHeaderValue(field, headerValue, fieldValue.Type())
	if err != nil {
		return fmt.Errorf("failed to process header %s: %w", headerName, err)
	}

	if processedValue != nil {
		fieldValue.Set(reflect.ValueOf(processedValue))

		return nil
	}

	// Use the transformed value if transformation occurred, otherwise use original
	valueToUse := transformedValue
	if valueToUse == "" {
		valueToUse = headerValue
	}

	if err := setFieldValueFromString(fieldValue, valueToUse); err != nil {
		return fmt.Errorf("failed to set header field %s: %w", field.Name, err)
	}

	return nil
}

// getHeaderValue retrieves a header value with default fallback.
func (d *HeaderDecoder[T]) getHeaderValue(r *http.Request, headerName, defaultValue string) string {
	headerValue := r.Header.Get(headerName)
	if headerValue == "" && defaultValue != "" {
		headerValue = handleDefaultValue(defaultValue)
	}

	return headerValue
}

// processHeaderValue applies transformations and formats to header values.
// Returns (processedValue, transformedString, error).
// processedValue is non-nil if the value was fully processed (formats, special types).
// transformedString contains the transformed value for fallback processing.
func (d *HeaderDecoder[T]) processHeaderValue(
	field *reflect.StructField, headerValue string, fieldType reflect.Type,
) (processedValue interface{}, transformedString string, err error) {
	originalValue := headerValue

	// Handle transformations
	if transform := field.Tag.Get("transform"); transform != "" {
		transformedValue, err := applyTransformation(transform, headerValue)
		if err != nil {
			return nil, "", err
		}
		headerValue = transformedValue
	}

	// Handle custom formats
	if format := field.Tag.Get("format"); format != "" {
		result, err := applyFormat(format, headerValue, fieldType)

		return result, headerValue, err
	}

	// Check if we need to handle special types after transformation
	if fieldType == reflect.TypeOf(net.IP{}) {
		ip := net.ParseIP(headerValue)
		if ip == nil {
			return nil, "", fmt.Errorf("%w: %s", ErrInvalidIPAddress, headerValue)
		}

		return ip, headerValue, nil
	}

	if fieldType == reflect.TypeOf(time.Time{}) {
		// Try to parse with standard formats
		formats := []string{
			time.RFC3339,
			"2006-01-02 15:04:05",
			"2006-01-02",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, headerValue); err == nil {
				return t, headerValue, nil
			}
		}

		return nil, "", fmt.Errorf("%w: %s", ErrInvalidTimeValue, headerValue)
	}

	// Return the transformed value for basic type processing
	if headerValue != originalValue {
		return nil, headerValue, nil
	}

	return nil, "", nil
}

// validateHeaderResult validates the final header result.
func (d *HeaderDecoder[T]) validateHeaderResult(result T) error {
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

		return NewValidationError("Header validation failed", validationErrors)
	}

	return nil
}

// ContentTypes returns the supported content types for header decoding.
func (d *HeaderDecoder[T]) ContentTypes() []string {
	return []string{"*/*"} // Headers work with any content type
}

// setFieldValueFromString sets a reflect.Value from a string value with type conversion.
func setFieldValueFromString(fieldValue reflect.Value, value string) error {
	// Handle special struct types first (before checking Kind)
	if fieldValue.Type() == reflect.TypeOf(net.IP{}) {
		return handleIPParsing(fieldValue, value)
	}
	if fieldValue.Type() == reflect.TypeOf(time.Time{}) {
		return handleTimeParsing(fieldValue, value)
	}

	//nolint:dupl // This switch is reused in other files for consistent type conversion
	switch fieldValue.Kind() {
	case reflect.String:
		fieldValue.SetString(value)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrInvalidIntegerValue, value)
		}
		fieldValue.SetInt(intValue)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintValue, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrInvalidUintegerValue, value)
		}
		fieldValue.SetUint(uintValue)

	case reflect.Float32, reflect.Float64:
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrInvalidFloatValue, value)
		}
		fieldValue.SetFloat(floatValue)

	case reflect.Bool:
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrInvalidBooleanValue, value)
		}
		fieldValue.SetBool(boolValue)

	case reflect.Invalid, reflect.Uintptr, reflect.Complex64, reflect.Complex128,
		reflect.Array, reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Ptr, reflect.Slice, reflect.Struct, reflect.UnsafePointer:
		// Unsupported field types for string parsing
		return fmt.Errorf("%w: %s", ErrUnsupportedFieldType, fieldValue.Kind())

	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedFieldType, fieldValue.Kind())
	}

	return nil
}

// handleIPParsing parses an IP address string.
func handleIPParsing(fieldValue reflect.Value, value string) error {
	ip := net.ParseIP(value)
	if ip == nil {
		return fmt.Errorf("%w: %s", ErrInvalidIPAddress, value)
	}
	fieldValue.Set(reflect.ValueOf(ip))

	return nil
}

// handleTimeParsing parses a time string with multiple format attempts.
func handleTimeParsing(fieldValue reflect.Value, value string) error {
	formats := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, value); err == nil {
			fieldValue.Set(reflect.ValueOf(t))

			return nil
		}
	}

	return fmt.Errorf("%w: %s", ErrInvalidTimeValue, value)
}

// handleDefaultValue processes default values, including special cases like "now".
func handleDefaultValue(defaultValue string) string {
	switch defaultValue {
	case "now":
		return time.Now().Format(time.RFC3339)
	case "generate_uuid":
		// In a real implementation, you'd generate a proper UUID
		return fmt.Sprintf("uuid-%d", time.Now().UnixNano())
	default:
		return defaultValue
	}
}

// applyTransformation applies built-in transformations to header values.
func applyTransformation(transform, value string) (string, error) {
	switch transform {
	case "first_ip":
		// Extract first IP from comma-separated list (common for X-Forwarded-For)
		ips := strings.Split(value, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0]), nil
		}

		return value, nil

	case "to_lower":
		return strings.ToLower(value), nil

	case "to_upper":
		return strings.ToUpper(value), nil

	case "trim_space":
		return strings.TrimSpace(value), nil

	case "is_admin":
		// Transform role to boolean
		return strconv.FormatBool(strings.EqualFold(value, "admin")), nil

	default:
		return "", fmt.Errorf("%w: %s", ErrUnknownTransformation, transform)
	}
}

// applyFormat applies custom format parsing to header values.
func applyFormat(format, value string, targetType reflect.Type) (interface{}, error) {
	switch {
	case targetType == reflect.TypeOf(time.Time{}):
		return parseTimeWithFormat(format, value)
	default:
		return nil, fmt.Errorf("%w: %s for type %s", ErrFormatNotSupported, format, targetType)
	}
}

// parseTimeWithFormat parses time strings with various formats.
func parseTimeWithFormat(format, value string) (time.Time, error) {
	switch format {
	case "unix":
		timestamp, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("%w: %s", ErrInvalidUnixTimestamp, value)
		}

		return time.Unix(timestamp, 0), nil

	case "rfc3339":
		t, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse RFC3339 time: %w", err)
		}

		return t, nil

	case "rfc822":
		t, err := time.Parse(time.RFC822, value)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse RFC822 time: %w", err)
		}

		return t, nil

	case "2006-01-02":
		t, err := time.Parse("2006-01-02", value)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse date: %w", err)
		}

		return t, nil

	case "2006-01-02 15:04:05":
		t, err := time.Parse("2006-01-02 15:04:05", value)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse datetime: %w", err)
		}

		return t, nil

	default:
		// Try custom format
		t, err := time.Parse(format, value)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse custom time format %s: %w", format, err)
		}

		return t, nil
	}
}
