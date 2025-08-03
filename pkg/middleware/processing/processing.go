package processing

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// Validation constants and types
var (
	ErrValidationFailed = errors.New("validation failed")
	ErrInvalidJSON      = errors.New("invalid JSON")
	ErrNilRequest       = errors.New("nil request")
)

// ValidationFunc represents a custom validation function
type ValidationFunc func(value interface{}) error

// ValidationConfig holds validation middleware configuration
type ValidationConfig struct {
	ValidateJSON     bool
	ValidateStruct   bool
	StrictValidation bool
	CustomValidators map[string]ValidationFunc
	ErrorHandler     func(error) error
}

// ValidationMiddleware provides request validation functionality
type ValidationMiddleware struct {
	config ValidationConfig
	mu     sync.RWMutex
}

// ValidationOption configures validation middleware
type ValidationOption func(*ValidationConfig)

// WithStrictValidation enables strict validation mode
func WithStrictValidation(strict bool) ValidationOption {
	return func(c *ValidationConfig) {
		c.StrictValidation = strict
	}
}

// WithCustomValidators sets custom validation functions
func WithCustomValidators(validators map[string]ValidationFunc) ValidationOption {
	return func(c *ValidationConfig) {
		c.CustomValidators = validators
	}
}

// WithValidationErrorHandler sets a custom error handler
func WithValidationErrorHandler(handler func(error) error) ValidationOption {
	return func(c *ValidationConfig) {
		c.ErrorHandler = handler
	}
}

// NewValidationMiddleware creates a new validation middleware
func NewValidationMiddleware(opts ...ValidationOption) *ValidationMiddleware {
	config := ValidationConfig{
		ValidateJSON:     true,
		ValidateStruct:   true,
		StrictValidation: false,
		CustomValidators: make(map[string]ValidationFunc),
		ErrorHandler:     func(err error) error { return err },
	}

	for _, opt := range opts {
		opt(&config)
	}

	return &ValidationMiddleware{
		config: config,
	}
}

// GetConfig returns the validation configuration
func (m *ValidationMiddleware) GetConfig() ValidationConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// HTTPMiddleware returns HTTP middleware function
func (m *ValidationMiddleware) HTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only validate JSON POST/PUT requests
			if r.Method == http.MethodPost || r.Method == http.MethodPut {
				contentType := r.Header.Get("Content-Type")
				if strings.Contains(contentType, "application/json") {
					// Read and validate JSON
					body, err := io.ReadAll(r.Body)
					if err != nil {
						m.writeError(w, http.StatusBadRequest, "failed to read request body")
						return
					}
					defer r.Body.Close()

					// Restore body for next handler
					r.Body = io.NopCloser(bytes.NewReader(body))

					// Validate JSON syntax
					var jsonData interface{}
					if err := json.Unmarshal(body, &jsonData); err != nil {
						m.writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
						return
					}

					// Perform structural validation if enabled
					if m.config.ValidateStruct {
						if err := m.validateJSONStructure(jsonData); err != nil {
							wrappedErr := m.config.ErrorHandler(err)
							m.writeError(w, http.StatusBadRequest, "validation error: "+wrappedErr.Error())
							return
						}
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Before implements TypedPreMiddleware interface
func (m *ValidationMiddleware) Before(ctx context.Context, req interface{}) (context.Context, error) {
	if req == nil {
		return ctx, ErrNilRequest
	}

	// Validate struct using reflection and tags
	if err := m.validateStruct(req); err != nil {
		wrappedErr := m.config.ErrorHandler(err)
		return ctx, fmt.Errorf("validation failed: %w", wrappedErr)
	}

	return ctx, nil
}

// ValidateValue validates a single value using a custom validator
func (m *ValidationMiddleware) ValidateValue(value interface{}, validatorName string) error {
	m.mu.RLock()
	validator, exists := m.config.CustomValidators[validatorName]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("validator '%s' not found", validatorName)
	}

	return validator(value)
}

// validateStruct validates a struct using reflection and struct tags
func (m *ValidationMiddleware) validateStruct(obj interface{}) error {
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil // Only validate structs
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}

		// Get validation tag
		validationTag := fieldType.Tag.Get("validate")
		if validationTag == "" {
			continue
		}

		// Parse and apply validation rules
		if err := m.applyValidationRules(field.Interface(), validationTag, fieldType.Name); err != nil {
			return err
		}
	}

	return nil
}

// applyValidationRules applies validation rules from struct tags
func (m *ValidationMiddleware) applyValidationRules(value interface{}, rules, fieldName string) error {
	ruleParts := strings.Split(rules, ",")

	for _, rule := range ruleParts {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}

		if err := m.applyValidationRule(value, rule, fieldName); err != nil {
			return err
		}
	}

	return nil
}

// applyValidationRule applies a single validation rule
func (m *ValidationMiddleware) applyValidationRule(value interface{}, rule, fieldName string) error {
	switch {
	case rule == "required":
		if m.isEmpty(value) {
			return fmt.Errorf("%s is required", fieldName)
		}
	case strings.HasPrefix(rule, "min="):
		if err := m.validateMin(value, rule, fieldName); err != nil {
			return err
		}
	case strings.HasPrefix(rule, "max="):
		if err := m.validateMax(value, rule, fieldName); err != nil {
			return err
		}
	case rule == "email":
		if err := m.validateEmail(value, fieldName); err != nil {
			return err
		}
	default:
		// Check custom validators
		m.mu.RLock()
		validator, exists := m.config.CustomValidators[rule]
		m.mu.RUnlock()

		if exists {
			return validator(value)
		}

		if m.config.StrictValidation {
			return fmt.Errorf("unknown validation rule: %s", rule)
		}
	}

	return nil
}

// Helper validation methods
func (m *ValidationMiddleware) isEmpty(value interface{}) bool {
	if value == nil {
		return true
	}

	switch v := value.(type) {
	case string:
		return v == ""
	case int, int8, int16, int32, int64:
		return v == 0
	case float32, float64:
		return v == 0
	default:
		return false
	}
}

func (m *ValidationMiddleware) validateMin(value interface{}, rule, fieldName string) error {
	minStr := strings.TrimPrefix(rule, "min=")
	min, err := strconv.Atoi(minStr)
	if err != nil {
		return fmt.Errorf("invalid min rule: %s", rule)
	}

	switch v := value.(type) {
	case string:
		if len(v) < min {
			return fmt.Errorf("%s must be at least %d characters", fieldName, min)
		}
	case int:
		if v < min {
			return fmt.Errorf("%s must be at least %d", fieldName, min)
		}
	}

	return nil
}

func (m *ValidationMiddleware) validateMax(value interface{}, rule, fieldName string) error {
	maxStr := strings.TrimPrefix(rule, "max=")
	max, err := strconv.Atoi(maxStr)
	if err != nil {
		return fmt.Errorf("invalid max rule: %s", rule)
	}

	switch v := value.(type) {
	case string:
		if len(v) > max {
			return fmt.Errorf("%s must be at most %d characters", fieldName, max)
		}
	case int:
		if v > max {
			return fmt.Errorf("%s must be at most %d", fieldName, max)
		}
	}

	return nil
}

func (m *ValidationMiddleware) validateEmail(value interface{}, fieldName string) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("%s must be a string for email validation", fieldName)
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(str) {
		return fmt.Errorf("%s must be a valid email address", fieldName)
	}

	return nil
}

// validateJSONStructure performs JSON structure validation based on common patterns
func (m *ValidationMiddleware) validateJSONStructure(data interface{}) error {
	switch v := data.(type) {
	case map[string]interface{}:
		// Validate common fields with typical validation rules
		if name, ok := v["name"]; ok {
			if err := m.validateJSONField(name, "name", "required,min=2,max=50"); err != nil {
				return err
			}
		}

		if email, ok := v["email"]; ok {
			if err := m.validateJSONField(email, "email", "required,email"); err != nil {
				return err
			}
		}

		if age, ok := v["age"]; ok {
			if err := m.validateJSONField(age, "age", "min=18,max=120"); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateJSONField validates a single JSON field with validation rules
func (m *ValidationMiddleware) validateJSONField(value interface{}, fieldName, rules string) error {
	// Convert JSON number to int for age validation
	if fieldName == "age" {
		if floatVal, ok := value.(float64); ok {
			value = int(floatVal)
		}
	}

	return m.applyValidationRules(value, rules, fieldName)
}

// writeError writes a validation error response
func (m *ValidationMiddleware) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

// Compression constants
const (
	DefaultCompressionLevel = 6
	BestCompression         = 9
	DefaultMinSize          = 1024
)

// CompressionConfig holds compression middleware configuration
type CompressionConfig struct {
	Level   int
	Types   []string
	MinSize int
	Headers map[string]string
}

// CompressionMiddleware provides response compression functionality
type CompressionMiddleware struct {
	config CompressionConfig
}

// CompressionOption configures compression middleware
type CompressionOption func(*CompressionConfig)

// WithCompressionLevel sets the compression level
func WithCompressionLevel(level int) CompressionOption {
	return func(c *CompressionConfig) {
		c.Level = level
	}
}

// WithCompressionTypes sets the content types to compress
func WithCompressionTypes(types []string) CompressionOption {
	return func(c *CompressionConfig) {
		c.Types = types
	}
}

// WithMinCompressionSize sets the minimum size to trigger compression
func WithMinCompressionSize(size int) CompressionOption {
	return func(c *CompressionConfig) {
		c.MinSize = size
	}
}

// WithCompressionHeaders sets additional headers for compressed responses
func WithCompressionHeaders(headers map[string]string) CompressionOption {
	return func(c *CompressionConfig) {
		c.Headers = headers
	}
}

// NewCompressionMiddleware creates a new compression middleware
func NewCompressionMiddleware(opts ...CompressionOption) *CompressionMiddleware {
	config := CompressionConfig{
		Level:   DefaultCompressionLevel,
		Types:   []string{"application/json", "text/html", "text/plain", "text/css", "application/javascript"},
		MinSize: DefaultMinSize,
		Headers: make(map[string]string),
	}

	for _, opt := range opts {
		opt(&config)
	}

	return &CompressionMiddleware{
		config: config,
	}
}

// GetConfig returns the compression configuration
func (m *CompressionMiddleware) GetConfig() CompressionConfig {
	return m.config
}

// HTTPMiddleware returns HTTP middleware function
func (m *CompressionMiddleware) HTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if client accepts gzip compression
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				next.ServeHTTP(w, r)
				return
			}

			// Wrap response writer
			cw := &compressionWriter{
				ResponseWriter: w,
				middleware:     m,
				request:        r,
			}

			next.ServeHTTP(cw, r)
		})
	}
}

// Before implements TypedPreMiddleware interface
func (m *CompressionMiddleware) Before(ctx context.Context, req interface{}) (context.Context, error) {
	// Add compression context if accept-encoding header indicates support
	if acceptEncoding, ok := ctx.Value("accept_encoding").(string); ok {
		if strings.Contains(acceptEncoding, "gzip") {
			ctx = context.WithValue(ctx, "compression_enabled", true)
		}
	}
	return ctx, nil
}

// After implements TypedPostMiddleware interface
func (m *CompressionMiddleware) After(ctx context.Context, req interface{}, resp interface{}, err error) (interface{}, error) {
	// In a real implementation, this would compress the response if enabled
	// For this middleware pattern, compression is typically handled at the HTTP layer
	return resp, err
}

// compressionWriter wraps http.ResponseWriter to provide compression
type compressionWriter struct {
	http.ResponseWriter
	middleware *CompressionMiddleware
	request    *http.Request
	buffer     bytes.Buffer
	gzWriter   *gzip.Writer
	wrote      bool
}

func (cw *compressionWriter) WriteHeader(statusCode int) {
	cw.ResponseWriter.WriteHeader(statusCode)
}

func (cw *compressionWriter) Write(data []byte) (int, error) {
	if !cw.wrote {
		cw.wrote = true

		// Check if content should be compressed
		contentType := cw.Header().Get("Content-Type")
		if cw.shouldCompress(contentType, len(data)) {
			cw.Header().Set("Content-Encoding", "gzip")
			cw.Header().Del("Content-Length") // Let gzip set this

			// Add custom headers
			for k, v := range cw.middleware.config.Headers {
				cw.Header().Set(k, v)
			}

			// Initialize gzip writer
			cw.gzWriter, _ = gzip.NewWriterLevel(cw.ResponseWriter, cw.middleware.config.Level)
			defer cw.gzWriter.Close()

			return cw.gzWriter.Write(data)
		}
	}

	if cw.gzWriter != nil {
		return cw.gzWriter.Write(data)
	}

	return cw.ResponseWriter.Write(data)
}

func (cw *compressionWriter) shouldCompress(contentType string, size int) bool {
	// Check minimum size
	if size < cw.middleware.config.MinSize {
		return false
	}

	// Check content type
	for _, t := range cw.middleware.config.Types {
		if strings.Contains(contentType, t) {
			return true
		}
	}

	return false
}

// CORS configuration and middleware
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// CORSMiddleware provides Cross-Origin Resource Sharing functionality
type CORSMiddleware struct {
	config CORSConfig
}

// CORSOption configures CORS middleware
type CORSOption func(*CORSConfig)

// WithAllowedOrigins sets allowed origins
func WithAllowedOrigins(origins []string) CORSOption {
	return func(c *CORSConfig) {
		c.AllowedOrigins = origins
	}
}

// WithAllowedMethods sets allowed HTTP methods
func WithAllowedMethods(methods []string) CORSOption {
	return func(c *CORSConfig) {
		c.AllowedMethods = methods
	}
}

// WithAllowedHeaders sets allowed headers
func WithAllowedHeaders(headers []string) CORSOption {
	return func(c *CORSConfig) {
		c.AllowedHeaders = headers
	}
}

// WithExposedHeaders sets exposed headers
func WithExposedHeaders(headers []string) CORSOption {
	return func(c *CORSConfig) {
		c.ExposedHeaders = headers
	}
}

// WithAllowCredentials sets whether to allow credentials
func WithAllowCredentials(allow bool) CORSOption {
	return func(c *CORSConfig) {
		c.AllowCredentials = allow
	}
}

// WithMaxAge sets the preflight cache max age
func WithMaxAge(maxAge int) CORSOption {
	return func(c *CORSConfig) {
		c.MaxAge = maxAge
	}
}

// NewCORSMiddleware creates a new CORS middleware
func NewCORSMiddleware(opts ...CORSOption) *CORSMiddleware {
	config := CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		ExposedHeaders:   []string{},
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
	}

	for _, opt := range opts {
		opt(&config)
	}

	return &CORSMiddleware{
		config: config,
	}
}

// GetConfig returns the CORS configuration
func (m *CORSMiddleware) GetConfig() CORSConfig {
	return m.config
}

// HTTPMiddleware returns HTTP middleware function
func (m *CORSMiddleware) HTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			if origin != "" && !m.isOriginAllowed(origin) {
				http.Error(w, "Origin not allowed", http.StatusForbidden)
				return
			}

			// Handle preflight request
			if r.Method == http.MethodOptions {
				m.handlePreflight(w, r)
				return
			}

			// Set CORS headers for actual request
			m.setCORSHeaders(w, origin)

			next.ServeHTTP(w, r)
		})
	}
}

// Before implements TypedPreMiddleware interface
func (m *CORSMiddleware) Before(ctx context.Context, req interface{}) (context.Context, error) {
	// Check origin from context
	if origin, ok := ctx.Value("origin").(string); ok {
		if origin != "" && !m.isOriginAllowed(origin) {
			return ctx, errors.New("CORS: origin not allowed")
		}
	}
	return ctx, nil
}

// After implements TypedPostMiddleware interface
func (m *CORSMiddleware) After(ctx context.Context, req interface{}, resp interface{}, err error) (interface{}, error) {
	return resp, err
}

// isOriginAllowed checks if an origin is allowed
func (m *CORSMiddleware) isOriginAllowed(origin string) bool {
	for _, allowed := range m.config.AllowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

// isMethodAllowed checks if a method is allowed
func (m *CORSMiddleware) isMethodAllowed(method string) bool {
	for _, allowed := range m.config.AllowedMethods {
		if allowed == method {
			return true
		}
	}
	return false
}

// areHeadersAllowed checks if headers are allowed
func (m *CORSMiddleware) areHeadersAllowed(headers []string) bool {
	for _, header := range headers {
		allowed := false
		for _, allowedHeader := range m.config.AllowedHeaders {
			if strings.EqualFold(header, allowedHeader) {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}
	return true
}

// handlePreflight handles CORS preflight requests
func (m *CORSMiddleware) handlePreflight(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	method := r.Header.Get("Access-Control-Request-Method")
	headers := r.Header.Get("Access-Control-Request-Headers")

	// Check method
	if !m.isMethodAllowed(method) {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check headers
	if headers != "" {
		headerList := strings.Split(headers, ",")
		for i, header := range headerList {
			headerList[i] = strings.TrimSpace(header)
		}
		if !m.areHeadersAllowed(headerList) {
			http.Error(w, "Headers not allowed", http.StatusForbidden)
			return
		}
	}

	// Set preflight response headers
	m.setCORSHeaders(w, origin)
	w.Header().Set("Access-Control-Allow-Methods", strings.Join(m.config.AllowedMethods, ", "))
	w.Header().Set("Access-Control-Allow-Headers", strings.Join(m.config.AllowedHeaders, ", "))
	w.Header().Set("Access-Control-Max-Age", strconv.Itoa(m.config.MaxAge))

	w.WriteHeader(http.StatusNoContent)
}

// setCORSHeaders sets common CORS headers
func (m *CORSMiddleware) setCORSHeaders(w http.ResponseWriter, origin string) {
	if origin != "" && m.isOriginAllowed(origin) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}

	if m.config.AllowCredentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	if len(m.config.ExposedHeaders) > 0 {
		w.Header().Set("Access-Control-Expose-Headers", strings.Join(m.config.ExposedHeaders, ", "))
	}
}

// ProcessingMiddleware combines validation, compression, and CORS middleware
type ProcessingMiddleware struct {
	validation  *ValidationMiddleware
	compression *CompressionMiddleware
	cors        *CORSMiddleware
}

// NewProcessingMiddleware creates a combined processing middleware
func NewProcessingMiddleware(validation *ValidationMiddleware, compression *CompressionMiddleware, cors *CORSMiddleware) *ProcessingMiddleware {
	return &ProcessingMiddleware{
		validation:  validation,
		compression: compression,
		cors:        cors,
	}
}

// HTTPMiddleware returns HTTP middleware function that combines all processing features
func (m *ProcessingMiddleware) HTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		handler := next

		// Apply middlewares in reverse order (CORS -> compression -> validation)
		if m.compression != nil {
			handler = m.compression.HTTPMiddleware()(handler)
		}
		if m.validation != nil {
			handler = m.validation.HTTPMiddleware()(handler)
		}
		if m.cors != nil {
			handler = m.cors.HTTPMiddleware()(handler)
		}

		return handler
	}
}
