package typedhttp

import (
	"fmt"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
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
	
	// Use reflection to map headers to struct fields
	resultValue := reflect.ValueOf(&result).Elem()
	resultType := resultValue.Type()

	for i := 0; i < resultType.NumField(); i++ {
		field := resultType.Field(i)
		fieldValue := resultValue.Field(i)
		
		// Skip unexported fields
		if !fieldValue.CanSet() {
			continue
		}

		// Get the header name from struct tag
		headerName := field.Tag.Get("header")
		if headerName == "" {
			continue // Only process fields with header tags
		}

		// Get default value from struct tag
		defaultValue := field.Tag.Get("default")

		// Get the header value from request (case-insensitive)
		headerValue := r.Header.Get(headerName)
		if headerValue == "" && defaultValue != "" {
			headerValue = handleDefaultValue(defaultValue)
		}

		if headerValue == "" {
			continue // Skip empty headers unless required (validation will catch this)
		}

		// Handle transformations
		if transform := field.Tag.Get("transform"); transform != "" {
			transformedValue, err := applyTransformation(transform, headerValue)
			if err != nil {
				return result, fmt.Errorf("failed to transform header %s: %w", headerName, err)
			}
			headerValue = transformedValue
		}

		// Handle custom formats (e.g., time parsing)
		if format := field.Tag.Get("format"); format != "" {
			formattedValue, err := applyFormat(format, headerValue, fieldValue.Type())
			if err != nil {
				return result, fmt.Errorf("failed to format header %s: %w", headerName, err)
			}
			
			// Set the formatted value directly
			fieldValue.Set(reflect.ValueOf(formattedValue))
			continue
		}

		// Set the field value based on its type
		if err := setFieldValueFromString(fieldValue, headerValue); err != nil {
			return result, fmt.Errorf("failed to set header field %s: %w", field.Name, err)
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
			return result, NewValidationError("Header validation failed", validationErrors)
		}
	}

	return result, nil
}

// ContentTypes returns the supported content types for header decoding.
func (d *HeaderDecoder[T]) ContentTypes() []string {
	return []string{"*/*"} // Headers work with any content type
}

// setFieldValueFromString sets a reflect.Value from a string value with type conversion.
func setFieldValueFromString(fieldValue reflect.Value, value string) error {
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
		// Handle special types
		if fieldValue.Type() == reflect.TypeOf(net.IP{}) {
			ip := net.ParseIP(value)
			if ip == nil {
				return fmt.Errorf("invalid IP address: %s", value)
			}
			fieldValue.Set(reflect.ValueOf(ip))
		} else if fieldValue.Type() == reflect.TypeOf(time.Time{}) {
			// Try RFC3339 first, then other common formats
			if t, err := time.Parse(time.RFC3339, value); err == nil {
				fieldValue.Set(reflect.ValueOf(t))
			} else if t, err := time.Parse("2006-01-02 15:04:05", value); err == nil {
				fieldValue.Set(reflect.ValueOf(t))
			} else if t, err := time.Parse("2006-01-02", value); err == nil {
				fieldValue.Set(reflect.ValueOf(t))
			} else {
				return fmt.Errorf("invalid time value: %s", value)
			}
		} else {
			return fmt.Errorf("unsupported field type: %s", fieldValue.Kind())
		}
	}
	
	return nil
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
		return strconv.FormatBool(strings.ToLower(value) == "admin"), nil
		
	default:
		return "", fmt.Errorf("unknown transformation: %s", transform)
	}
}

// applyFormat applies custom format parsing to header values.
func applyFormat(format, value string, targetType reflect.Type) (interface{}, error) {
	switch {
	case targetType == reflect.TypeOf(time.Time{}):
		return parseTimeWithFormat(format, value)
	default:
		return nil, fmt.Errorf("format %s not supported for type %s", format, targetType)
	}
}

// parseTimeWithFormat parses time strings with various formats.
func parseTimeWithFormat(format, value string) (time.Time, error) {
	switch format {
	case "unix":
		timestamp, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid unix timestamp: %s", value)
		}
		return time.Unix(timestamp, 0), nil
		
	case "rfc3339":
		return time.Parse(time.RFC3339, value)
		
	case "rfc822":
		return time.Parse(time.RFC822, value)
		
	case "2006-01-02":
		return time.Parse("2006-01-02", value)
		
	case "2006-01-02 15:04:05":
		return time.Parse("2006-01-02 15:04:05", value)
		
	default:
		// Try custom format
		return time.Parse(format, value)
	}
}