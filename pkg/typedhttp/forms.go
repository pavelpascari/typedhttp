package typedhttp

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

const (
	// MaxFormMemory is the maximum memory to use for parsing multipart forms (32MB).
	MaxFormMemory = 32 << 20
	// JSONFieldTagValue is the value for the json_field tag that indicates a field should be parsed as JSON.
	JSONFieldTagValue = "true"
)

// Error variables for static error handling.
var (
	ErrFileUploadsNotAllowed = errors.New("file uploads not allowed")
	ErrNoMultipartFormData   = errors.New("no multipart form data")
	ErrNoFileFound           = errors.New("no file found for field")
	ErrNoFilesFound          = errors.New("no files found for field")
	ErrFileTypeMismatch      = errors.New("file type mismatch")
	ErrFileTooLarge          = errors.New("file too large")
	ErrInvalidFileType       = errors.New("invalid file type")
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

	if err := d.parseFormData(r); err != nil {
		return result, err
	}

	if err := d.processFormFields(r, &result); err != nil {
		return result, err
	}

	if err := d.validateResult(result); err != nil {
		return result, err
	}

	return result, nil
}

// parseFormData parses the form data based on content type.
func (d *FormDecoder[T]) parseFormData(r *http.Request) error {
	contentType := r.Header.Get("Content-Type")

	var err error
	if strings.Contains(contentType, "multipart/form-data") {
		err = r.ParseMultipartForm(d.maxMemory)
	} else {
		err = r.ParseForm()
	}

	if err != nil {
		return fmt.Errorf("failed to parse form data: %w", err)
	}

	return nil
}

// processFormFields processes all form fields using reflection.
func (d *FormDecoder[T]) processFormFields(r *http.Request, result *T) error {
	resultValue := reflect.ValueOf(result).Elem()
	resultType := resultValue.Type()

	for i := 0; i < resultType.NumField(); i++ {
		field := resultType.Field(i)
		fieldValue := resultValue.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		formName := field.Tag.Get("form")
		if formName == "" {
			continue
		}

		if err := d.processFormField(r, &field, fieldValue, formName); err != nil {
			return err
		}
	}

	return nil
}

// processFormField processes a single form field.
func (d *FormDecoder[T]) processFormField(
	r *http.Request, field *reflect.StructField, fieldValue reflect.Value, formName string,
) error {
	// Handle file uploads
	if d.isFileUploadField(fieldValue) {
		return d.handleFileUpload(r, fieldValue, formName)
	}

	// Get form value
	formValue := d.getFormValue(r, formName, field.Tag.Get("default"))
	if formValue == "" {
		return nil
	}

	// Handle special field types
	if d.isJSONField(field, formValue) {
		return d.parseJSONField(fieldValue, formValue)
	}

	if d.isStringSliceField(fieldValue) {
		d.handleStringSlice(fieldValue, formValue)

		return nil
	}

	// Apply transformations and formats
	processedValue, err := d.processFieldValue(field, formValue, fieldValue.Type())
	if err != nil {
		return fmt.Errorf("failed to process form field %s: %w", formName, err)
	}

	if processedValue != nil {
		fieldValue.Set(reflect.ValueOf(processedValue))

		return nil
	}

	// Set the field value based on its type
	if err := setFieldValueFromString(fieldValue, formValue); err != nil {
		return fmt.Errorf("failed to set form field %s: %w", field.Name, err)
	}

	return nil
}

// isFileUploadField checks if a field is for file upload.
func (d *FormDecoder[T]) isFileUploadField(fieldValue reflect.Value) bool {
	return fieldValue.Type() == reflect.TypeOf((*multipart.FileHeader)(nil)) ||
		fieldValue.Type() == reflect.TypeOf([]*multipart.FileHeader{})
}

// handleFileUpload handles file upload fields.
func (d *FormDecoder[T]) handleFileUpload(r *http.Request, fieldValue reflect.Value, formName string) error {
	if !d.allowFiles {
		return ErrFileUploadsNotAllowed
	}

	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		return nil
	}

	files, exists := r.MultipartForm.File[formName]
	if !exists {
		return nil
	}

	if fieldValue.Type() == reflect.TypeOf((*multipart.FileHeader)(nil)) {
		if len(files) > 0 {
			fieldValue.Set(reflect.ValueOf(files[0]))
		}
	} else {
		fieldValue.Set(reflect.ValueOf(files))
	}

	return nil
}

// getFormValue retrieves a form value with default fallback.
func (d *FormDecoder[T]) getFormValue(r *http.Request, formName, defaultValue string) string {
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

	return formValue
}

// isJSONField checks if a field should be parsed as JSON.
func (d *FormDecoder[T]) isJSONField(field *reflect.StructField, formValue string) bool {
	return field.Tag.Get("json_field") == JSONFieldTagValue ||
		strings.HasPrefix(formValue, "{") ||
		strings.HasPrefix(formValue, "[")
}

// isStringSliceField checks if a field is a string slice.
func (d *FormDecoder[T]) isStringSliceField(fieldValue reflect.Value) bool {
	return fieldValue.Kind() == reflect.Slice &&
		fieldValue.Type().Elem().Kind() == reflect.String
}

// handleStringSlice handles comma-separated string values.
func (d *FormDecoder[T]) handleStringSlice(fieldValue reflect.Value, formValue string) {
	values := strings.Split(formValue, ",")
	for i, val := range values {
		values[i] = strings.TrimSpace(val)
	}
	fieldValue.Set(reflect.ValueOf(values))
}

// processFieldValue applies transformations and formats to field values.
func (d *FormDecoder[T]) processFieldValue(
	field *reflect.StructField, formValue string, fieldType reflect.Type,
) (interface{}, error) {
	originalValue := formValue

	// Handle transformations
	if transform := field.Tag.Get("transform"); transform != "" {
		transformedValue, err := applyTransformation(transform, formValue)
		if err != nil {
			return nil, err
		}
		formValue = transformedValue
	}

	// Handle custom formats
	if format := field.Tag.Get("format"); format != "" {
		return applyFormat(format, formValue, fieldType)
	}

	// Return transformed value if transformation was applied
	if formValue != originalValue {
		return formValue, nil
	}

	//nolint:nilnil
	return nil, nil
}

// validateResult performs validation on the final result.
func (d *FormDecoder[T]) validateResult(result T) error {
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

		return NewValidationError("Form validation failed", validationErrors)
	}

	return nil
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
		return nil, ErrNoMultipartFormData
	}

	files, exists := r.MultipartForm.File[name]
	if !exists || len(files) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrNoFileFound, name)
	}

	return files[0], nil
}

// GetFileHeaders retrieves all file headers for a field (for multiple file uploads).
func GetFileHeaders(r *http.Request, name string) ([]*multipart.FileHeader, error) {
	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		return nil, ErrNoMultipartFormData
	}

	files, exists := r.MultipartForm.File[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrNoFilesFound, name)
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

// FormOptions Advanced form processing options.
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
		return ErrFileUploadsNotAllowed
	}

	if options.MaxFileSize > 0 && file.Size > options.MaxFileSize {
		return fmt.Errorf("%w: size %d exceeds maximum %d", ErrFileTooLarge, file.Size, options.MaxFileSize)
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
			return fmt.Errorf("%w: %s", ErrInvalidFileType, fileHeader)
		}
	}

	return nil
}
