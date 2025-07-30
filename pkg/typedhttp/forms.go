package typedhttp

import (
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

const (
	// MaxFormMemory is the maximum memory to use for parsing multipart forms (32MB)
	MaxFormMemory = 32 << 20
)

// FormDecoder implements RequestDecoder for form data (both multipart and URL-encoded).
type FormDecoder[T any] struct {
	validator  *validator.Validate
	maxMemory  int64
	allowFiles bool // Whether to allow file uploads
}

// NewFormDecoder creates a new form data decoder.
func NewFormDecoder[T any](validator *validator.Validate) *FormDecoder[T] {
	return &FormDecoder[T]{
		validator:  validator,
		maxMemory:  MaxFormMemory,
		allowFiles: true,
	}
}

// NewFormDecoderWithOptions creates a form decoder with custom options.
func NewFormDecoderWithOptions[T any](validator *validator.Validate, maxMemory int64, allowFiles bool) *FormDecoder[T] {
	return &FormDecoder[T]{
		validator:  validator,
		maxMemory:  maxMemory,
		allowFiles: allowFiles,
	}
}

// Decode decodes form data into the target type using reflection.
func (d *FormDecoder[T]) Decode(r *http.Request) (T, error) {
	var result T

	// Parse the form data based on content type
	contentType := r.Header.Get("Content-Type")

	var err error
	if strings.Contains(contentType, "multipart/form-data") {
		err = r.ParseMultipartForm(d.maxMemory)
	} else {
		err = r.ParseForm()
	}

	if err != nil {
		return result, fmt.Errorf("failed to parse form data: %w", err)
	}

	// Use reflection to map form fields to struct fields
	resultValue := reflect.ValueOf(&result).Elem()
	resultType := resultValue.Type()

	for i := 0; i < resultType.NumField(); i++ {
		field := resultType.Field(i)
		fieldValue := resultValue.Field(i)

		// Skip unexported fields
		if !fieldValue.CanSet() {
			continue
		}

		// Get the form field name from struct tag
		formName := field.Tag.Get("form")
		if formName == "" {
			continue // Only process fields with form tags
		}

		// Get default value from struct tag
		defaultValue := field.Tag.Get("default")

		// Handle file uploads specially
		if fieldValue.Type() == reflect.TypeOf((*multipart.FileHeader)(nil)) {
			if !d.allowFiles {
				return result, fmt.Errorf("file uploads not allowed")
			}

			if r.MultipartForm != nil && r.MultipartForm.File != nil {
				if files, exists := r.MultipartForm.File[formName]; exists && len(files) > 0 {
					fieldValue.Set(reflect.ValueOf(files[0]))
				}
			}
			continue
		}

		// Handle slice of file headers for multiple file uploads
		if fieldValue.Type() == reflect.TypeOf([]*multipart.FileHeader{}) {
			if !d.allowFiles {
				return result, fmt.Errorf("file uploads not allowed")
			}

			if r.MultipartForm != nil && r.MultipartForm.File != nil {
				if files, exists := r.MultipartForm.File[formName]; exists {
					fieldValue.Set(reflect.ValueOf(files))
				}
			}
			continue
		}

		// Get the form value
		var formValue string
		if r.Form != nil {
			formValues := r.Form[formName]
			if len(formValues) > 0 {
				formValue = formValues[0]
			}
		}

		if formValue == "" && defaultValue != "" {
			formValue = handleDefaultValue(defaultValue)
		}

		if formValue == "" {
			continue // Skip empty form fields unless required (validation will catch this)
		}

		// Handle JSON parsing for complex fields
		if field.Tag.Get("json_field") == "true" || strings.HasPrefix(formValue, "{") || strings.HasPrefix(formValue, "[") {
			if err := d.parseJSONField(fieldValue, formValue); err != nil {
				return result, fmt.Errorf("failed to parse JSON field %s: %w", formName, err)
			}
			continue
		}

		// Handle slice fields (comma-separated values)
		if fieldValue.Kind() == reflect.Slice && fieldValue.Type().Elem().Kind() == reflect.String {
			values := strings.Split(formValue, ",")
			for i, val := range values {
				values[i] = strings.TrimSpace(val)
			}
			fieldValue.Set(reflect.ValueOf(values))
			continue
		}

		// Handle transformations
		if transform := field.Tag.Get("transform"); transform != "" {
			transformedValue, err := applyTransformation(transform, formValue)
			if err != nil {
				return result, fmt.Errorf("failed to transform form field %s: %w", formName, err)
			}
			formValue = transformedValue
		}

		// Handle custom formats (e.g., time parsing)
		if format := field.Tag.Get("format"); format != "" {
			formattedValue, err := applyFormat(format, formValue, fieldValue.Type())
			if err != nil {
				return result, fmt.Errorf("failed to format form field %s: %w", formName, err)
			}

			// Set the formatted value directly
			fieldValue.Set(reflect.ValueOf(formattedValue))
			continue
		}

		// Set the field value based on its type
		if err := setFieldValueFromString(fieldValue, formValue); err != nil {
			return result, fmt.Errorf("failed to set form field %s: %w", field.Name, err)
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
			return result, NewValidationError("Form validation failed", validationErrors)
		}
	}

	return result, nil
}

// ContentTypes returns the supported content types for form decoding.
func (d *FormDecoder[T]) ContentTypes() []string {
	return []string{
		"application/x-www-form-urlencoded",
		"multipart/form-data",
	}
}

// parseJSONField parses a JSON string into a struct field.
func (d *FormDecoder[T]) parseJSONField(fieldValue reflect.Value, jsonValue string) error {
	// Create a new instance of the field type to unmarshal into
	fieldPtr := reflect.New(fieldValue.Type())

	if err := json.Unmarshal([]byte(jsonValue), fieldPtr.Interface()); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Set the field value
	fieldValue.Set(fieldPtr.Elem())
	return nil
}

// GetFormValue retrieves a form value with a fallback default.
func GetFormValue(r *http.Request, name, defaultValue string) string {
	if r.Form == nil {
		return defaultValue
	}

	values := r.Form[name]
	if len(values) > 0 && values[0] != "" {
		return values[0]
	}

	return defaultValue
}

// GetFormValues retrieves all values for a form field (for checkboxes, multi-select).
func GetFormValues(r *http.Request, name string) []string {
	if r.Form == nil {
		return nil
	}

	return r.Form[name]
}

// GetFileHeader retrieves a file header from multipart form data.
func GetFileHeader(r *http.Request, name string) (*multipart.FileHeader, error) {
	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		return nil, fmt.Errorf("no multipart form data")
	}

	files, exists := r.MultipartForm.File[name]
	if !exists || len(files) == 0 {
		return nil, fmt.Errorf("no file found for field %s", name)
	}

	return files[0], nil
}

// GetFileHeaders retrieves all file headers for a field (for multiple file uploads).
func GetFileHeaders(r *http.Request, name string) ([]*multipart.FileHeader, error) {
	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		return nil, fmt.Errorf("no multipart form data")
	}

	files, exists := r.MultipartForm.File[name]
	if !exists {
		return nil, fmt.Errorf("no files found for field %s", name)
	}

	return files, nil
}

// FormInfo provides information about the parsed form for debugging.
type FormInfo struct {
	ContentType string                             `json:"content_type"`
	Fields      map[string][]string                `json:"fields"`
	Files       map[string][]*multipart.FileHeader `json:"files"`
	TotalSize   int64                              `json:"total_size"`
	FieldCount  int                                `json:"field_count"`
	FileCount   int                                `json:"file_count"`
}

// GetFormInfo returns detailed information about the parsed form.
func GetFormInfo(r *http.Request) *FormInfo {
	info := &FormInfo{
		ContentType: r.Header.Get("Content-Type"),
		Fields:      make(map[string][]string),
		Files:       make(map[string][]*multipart.FileHeader),
	}

	if r.Form != nil {
		info.Fields = r.Form
		info.FieldCount = len(r.Form)
	}

	if r.MultipartForm != nil && r.MultipartForm.File != nil {
		info.Files = r.MultipartForm.File
		for _, files := range r.MultipartForm.File {
			info.FileCount += len(files)
			for _, file := range files {
				info.TotalSize += file.Size
			}
		}
	}

	return info
}

// FormOptions Advanced form processing options
type FormOptions struct {
	MaxMemory    int64    // Maximum memory for multipart forms
	AllowFiles   bool     // Whether to allow file uploads
	MaxFileSize  int64    // Maximum size per file
	MaxFiles     int      // Maximum number of files
	AllowedTypes []string // Allowed MIME types for files
}

// ValidateFileUpload validates an uploaded file against the form options.
func ValidateFileUpload(file *multipart.FileHeader, options FormOptions) error {
	if !options.AllowFiles {
		return fmt.Errorf("file uploads not allowed")
	}

	if options.MaxFileSize > 0 && file.Size > options.MaxFileSize {
		return fmt.Errorf("file size %d exceeds maximum %d", file.Size, options.MaxFileSize)
	}

	if len(options.AllowedTypes) > 0 {
		// Get the content type from the file header
		fileHeader := file.Header.Get("Content-Type")
		allowed := false
		for _, allowedType := range options.AllowedTypes {
			if strings.Contains(fileHeader, allowedType) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("file type %s not allowed", fileHeader)
		}
	}

	return nil
}
