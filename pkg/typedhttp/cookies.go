package typedhttp

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// CookieDecoder implements RequestDecoder for HTTP cookies.
type CookieDecoder[T any] struct {
	validator *validator.Validate
}

// NewCookieDecoder creates a new HTTP cookie decoder.
func NewCookieDecoder[T any](validator *validator.Validate) *CookieDecoder[T] {
	return &CookieDecoder[T]{
		validator: validator,
	}
}

// Decode decodes HTTP cookies into the target type using reflection.
func (d *CookieDecoder[T]) Decode(r *http.Request) (T, error) {
	var result T

	// Use reflection to map cookies to struct fields
	resultValue := reflect.ValueOf(&result).Elem()
	resultType := resultValue.Type()

	for i := 0; i < resultType.NumField(); i++ {
		field := resultType.Field(i)
		fieldValue := resultValue.Field(i)

		// Skip unexported fields
		if !fieldValue.CanSet() {
			continue
		}

		// Get the cookie name from the struct tag
		cookieName := field.Tag.Get("cookie")
		if cookieName == "" {
			continue // Only process fields with cookie tags
		}

		// Get default value from struct tag
		defaultValue := field.Tag.Get("default")

		// Get the cookie value from the request
		var cookieValue string
		if cookie, err := r.Cookie(cookieName); err == nil {
			cookieValue = cookie.Value
		}

		if cookieValue == "" && defaultValue != "" {
			cookieValue = handleDefaultValue(defaultValue)
		}

		if cookieValue == "" {
			continue // Skip empty cookies unless required (validation will catch this)
		}

		// Handle transformations
		if transform := field.Tag.Get("transform"); transform != "" {
			transformedValue, err := applyTransformation(transform, cookieValue)
			if err != nil {
				return result, fmt.Errorf("failed to transform cookie %s: %w", cookieName, err)
			}
			cookieValue = transformedValue
		}

		// Handle custom formats (e.g., time parsing)
		if format := field.Tag.Get("format"); format != "" {
			formattedValue, err := applyFormat(format, cookieValue, fieldValue.Type())
			if err != nil {
				return result, fmt.Errorf("failed to format cookie %s: %w", cookieName, err)
			}

			// Set the formatted value directly
			fieldValue.Set(reflect.ValueOf(formattedValue))
			continue
		}

		// Set the field value based on its type
		if err := setFieldValueFromString(fieldValue, cookieValue); err != nil {
			return result, fmt.Errorf("failed to set cookie field %s: %w", field.Name, err)
		}
	}

	// Perform validation if a validator is available
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
			return result, NewValidationError("Cookie validation failed", validationErrors)
		}
	}

	return result, nil
}

// ContentTypes returns the supported content types for cookie decoding.
func (d *CookieDecoder[T]) ContentTypes() []string {
	return []string{"*/*"} // Cookies work with any content type
}

// GetAllCookies returns all cookies from the request as a map for debugging/inspection.
func GetAllCookies(r *http.Request) map[string]string {
	cookies := make(map[string]string)
	for _, cookie := range r.Cookies() {
		cookies[cookie.Name] = cookie.Value
	}
	return cookies
}

// GetCookieWithDefault retrieves a cookie value with a fallback default.
func GetCookieWithDefault(r *http.Request, name, defaultValue string) string {
	if cookie, err := r.Cookie(name); err == nil {
		return cookie.Value
	}
	return defaultValue
}

// ParseSignedCookie parses a signed cookie (for security).
// This is a placeholder for more advanced cookie security features.
func ParseSignedCookie(cookieValue, secret string) (string, error) {
	// In a real implementation, you would verify the signature
	// For now, just return the value as-is
	// TODO: Implement proper cookie signing/verification
	return cookieValue, nil
}

// SecureCookieDecoder wraps CookieDecoder with additional security features.
type SecureCookieDecoder[T any] struct {
	decoder *CookieDecoder[T]
	secret  string // For signed cookies
}

// NewSecureCookieDecoder creates a new secure cookie decoder.
func NewSecureCookieDecoder[T any](validator *validator.Validate, secret string) *SecureCookieDecoder[T] {
	return &SecureCookieDecoder[T]{
		decoder: NewCookieDecoder[T](validator),
		secret:  secret,
	}
}

// Decode decodes cookies with security verification.
func (d *SecureCookieDecoder[T]) Decode(r *http.Request) (T, error) {
	// For now, delegate to the regular decoder
	// TODO: Add signature verification for secure cookies
	return d.decoder.Decode(r)
}

// ContentTypes returns the supported content types.
func (d *SecureCookieDecoder[T]) ContentTypes() []string {
	return d.decoder.ContentTypes()
}
