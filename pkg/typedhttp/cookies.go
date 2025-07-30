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

	if err := d.processCookieFields(r, &result); err != nil {
		return result, err
	}

	if err := d.validateCookieResult(result); err != nil {
		return result, err
	}

	return result, nil
}

// processCookieFields processes all cookie fields using reflection.
func (d *CookieDecoder[T]) processCookieFields(r *http.Request, result *T) error {
	resultValue := reflect.ValueOf(result).Elem()
	resultType := resultValue.Type()

	for i := 0; i < resultType.NumField(); i++ {
		field := resultType.Field(i)
		fieldValue := resultValue.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		cookieName := field.Tag.Get("cookie")
		if cookieName == "" {
			continue
		}

		if err := d.processCookieField(r, &field, fieldValue, cookieName); err != nil {
			return err
		}
	}

	return nil
}

// processCookieField processes a single cookie field.
func (d *CookieDecoder[T]) processCookieField(
	r *http.Request, field *reflect.StructField, fieldValue reflect.Value, cookieName string,
) error {
	cookieValue := d.getCookieValue(r, cookieName, field.Tag.Get("default"))
	if cookieValue == "" {
		return nil
	}

	processedValue, err := d.processCookieValue(field, cookieValue, fieldValue.Type())
	if err != nil {
		return fmt.Errorf("failed to process cookie %s: %w", cookieName, err)
	}

	if processedValue != nil {
		fieldValue.Set(reflect.ValueOf(processedValue))

		return nil
	}

	if err := setFieldValueFromString(fieldValue, cookieValue); err != nil {
		return fmt.Errorf("failed to set cookie field %s: %w", field.Name, err)
	}

	return nil
}

// getCookieValue retrieves a cookie value with default fallback.
func (d *CookieDecoder[T]) getCookieValue(r *http.Request, cookieName, defaultValue string) string {
	var cookieValue string
	if cookie, err := r.Cookie(cookieName); err == nil {
		cookieValue = cookie.Value
	}

	if cookieValue == "" && defaultValue != "" {
		cookieValue = handleDefaultValue(defaultValue)
	}

	return cookieValue
}

// processCookieValue applies transformations and formats to cookie values.
func (d *CookieDecoder[T]) processCookieValue(
	field *reflect.StructField, cookieValue string, fieldType reflect.Type,
) (interface{}, error) {
	originalValue := cookieValue

	// Handle transformations
	if transform := field.Tag.Get("transform"); transform != "" {
		transformedValue, err := applyTransformation(transform, cookieValue)
		if err != nil {
			return nil, err
		}
		cookieValue = transformedValue
	}

	// Handle custom formats
	if format := field.Tag.Get("format"); format != "" {
		return applyFormat(format, cookieValue, fieldType)
	}

	// Return transformed value if transformation was applied
	if cookieValue != originalValue {
		return cookieValue, nil
	}

	// No processing needed
	//nolint:nilnil
	return nil, nil
}

// validateCookieResult validates the final cookie result.
func (d *CookieDecoder[T]) validateCookieResult(result T) error {
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

		return NewValidationError("Cookie validation failed", validationErrors)
	}

	return nil
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
	// NOTE: Implement proper cookie signing/verification in production
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
	// NOTE: Add signature verification for secure cookies in production
	return d.decoder.Decode(r)
}

// ContentTypes returns the supported content types.
func (d *SecureCookieDecoder[T]) ContentTypes() []string {
	return d.decoder.ContentTypes()
}
