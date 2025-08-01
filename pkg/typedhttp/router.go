package typedhttp

import (
	"net/http"
	"reflect"

	"github.com/go-playground/validator/v10"
)

// Core router types and functionality

// Router represents a typed HTTP router that provides type-safe handler registration.
// Note: Due to Go's limitation with generic interface methods, the actual implementation
// will be in a concrete type that provides generic methods.
type Router interface {
	// ServeHTTP implements http.Handler
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// HandlerRegistration stores metadata about a registered handler for OpenAPI generation.
type HandlerRegistration struct {
	Method       string
	Path         string
	RequestType  reflect.Type
	ResponseType reflect.Type
	Metadata     OpenAPIMetadata
	Config       HandlerConfig
}

// HTTPHandler wraps a typed handler with HTTP-specific functionality.
type HTTPHandler[TRequest, TResponse any] struct {
	handler     Handler[TRequest, TResponse]
	decoder     RequestDecoder[TRequest]
	encoder     ResponseEncoder[TResponse]
	errorMapper ErrorMapper
	middleware  []Middleware
	metadata    OpenAPIMetadata
	config      ObservabilityConfig
}

// ServeHTTP implements http.Handler for the typed handler.
func (h *HTTPHandler[TRequest, TResponse]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req TRequest
	var resp TResponse
	var err error

	// Apply middleware
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Decode request
		if h.decoder != nil {
			req, err = h.decoder.Decode(r)
		} else {
			// Use default combined decoder (handles path, query, and JSON)
			v := validator.New()
			decoder := NewCombinedDecoder[TRequest](v)
			req, err = decoder.Decode(r)
		}

		if err != nil {
			h.handleError(w, err)

			return
		}

		// Call business logic handler
		resp, err = h.handler.Handle(r.Context(), req)
		if err != nil {
			h.handleError(w, err)

			return
		}

		// Encode response
		statusCode := http.StatusOK
		if r.Method == http.MethodPost {
			statusCode = http.StatusCreated
		}

		if h.encoder != nil {
			err = h.encoder.Encode(w, resp, statusCode)
		} else {
			// Use default JSON encoder
			encoder := NewJSONEncoder[TResponse]()
			err = encoder.Encode(w, resp, statusCode)
		}

		if err != nil {
			h.handleError(w, err)

			return
		}
	})

	// Apply middleware chain
	var finalHandler http.Handler = handler
	for i := len(h.middleware) - 1; i >= 0; i-- {
		finalHandler = h.middleware[i](finalHandler)
	}

	finalHandler.ServeHTTP(w, r)
}

// handleError handles errors using the configured error mapper.
func (h *HTTPHandler[TRequest, TResponse]) handleError(w http.ResponseWriter, err error) {
	var statusCode int
	var response interface{}

	if h.errorMapper != nil {
		statusCode, response = h.errorMapper.MapError(err)
	} else {
		// Use default error mapper
		mapper := &DefaultErrorMapper{}
		statusCode, response = mapper.MapError(err)
	}

	// Encode error response (this will set content-type and status code)
	encoder := NewJSONEncoder[interface{}]()
	if encodeErr := encoder.Encode(w, response, statusCode); encodeErr != nil {
		// Fallback to a simple error message
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// TypedRouter is a concrete implementation of Router with generic methods.
type TypedRouter struct {
	handlers []HandlerRegistration
	mux      *http.ServeMux
}

// NewRouter creates a new typed router.
func NewRouter() *TypedRouter {
	return &TypedRouter{
		handlers: make([]HandlerRegistration, 0),
		mux:      http.NewServeMux(),
	}
}

// ServeHTTP implements http.Handler.
func (r *TypedRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// GetHandlers returns all registered handlers.
func (r *TypedRouter) GetHandlers() []HandlerRegistration {
	return r.handlers
}

// registerHandler is an internal method to register handlers.
func (r *TypedRouter) registerHandler(
	method, path string,
	httpHandler http.Handler,
	requestType, responseType reflect.Type,
	metadata *OpenAPIMetadata,
) {
	// Store registration metadata
	registration := HandlerRegistration{
		Method:       method,
		Path:         path,
		RequestType:  requestType,
		ResponseType: responseType,
		Metadata:     *metadata,
	}

	r.handlers = append(r.handlers, registration)

	// Register with HTTP mux
	pattern := method + " " + path
	r.mux.HandleFunc(pattern, httpHandler.ServeHTTP)
}

// Generic registration functions (standalone functions, not methods)

// RegisterHandler registers a typed handler with the specified method and path.
func RegisterHandler[TReq, TResp any](
	router *TypedRouter,
	method, path string,
	handler Handler[TReq, TResp],
	opts ...HandlerOption,
) {
	// Create HTTP handler wrapper
	httpHandler := NewHTTPHandler(handler, opts...)

	// Register with router
	router.registerHandler(
		method,
		path,
		httpHandler,
		reflect.TypeOf((*TReq)(nil)).Elem(),
		reflect.TypeOf((*TResp)(nil)).Elem(),
		&httpHandler.metadata,
	)
}

// Convenience functions for common HTTP verbs.

func GET[TReq, TResp any](router *TypedRouter, path string, handler Handler[TReq, TResp], opts ...HandlerOption) {
	RegisterHandler(router, "GET", path, handler, opts...)
}

func POST[TReq, TResp any](router *TypedRouter, path string, handler Handler[TReq, TResp], opts ...HandlerOption) {
	RegisterHandler(router, "POST", path, handler, opts...)
}

func PUT[TReq, TResp any](router *TypedRouter, path string, handler Handler[TReq, TResp], opts ...HandlerOption) {
	RegisterHandler(router, "PUT", path, handler, opts...)
}

func PATCH[TReq, TResp any](router *TypedRouter, path string, handler Handler[TReq, TResp], opts ...HandlerOption) {
	RegisterHandler(router, "PATCH", path, handler, opts...)
}

func DELETE[TReq, TResp any](router *TypedRouter, path string, handler Handler[TReq, TResp], opts ...HandlerOption) {
	RegisterHandler(router, "DELETE", path, handler, opts...)
}

func HEAD[TReq, TResp any](router *TypedRouter, path string, handler Handler[TReq, TResp], opts ...HandlerOption) {
	RegisterHandler(router, "HEAD", path, handler, opts...)
}

func OPTIONS[TReq, TResp any](router *TypedRouter, path string, handler Handler[TReq, TResp], opts ...HandlerOption) {
	RegisterHandler(router, "OPTIONS", path, handler, opts...)
}

// NewHTTPHandler creates a new HTTP handler wrapper around a typed handler.
func NewHTTPHandler[TRequest, TResponse any](
	handler Handler[TRequest, TResponse],
	opts ...HandlerOption,
) *HTTPHandler[TRequest, TResponse] {
	config := &HandlerConfig{}
	for _, opt := range opts {
		opt(config)
	}

	httpHandler := &HTTPHandler[TRequest, TResponse]{
		handler:  handler,
		metadata: config.Metadata,
		config:   config.Observability,
	}

	// Set decoder
	if config.Decoder != nil {
		if decoder, ok := config.Decoder.(RequestDecoder[TRequest]); ok {
			httpHandler.decoder = decoder
		}
	}

	// Set encoder
	if config.Encoder != nil {
		if encoder, ok := config.Encoder.(ResponseEncoder[TResponse]); ok {
			httpHandler.encoder = encoder
		}
	}

	// Set error mapper
	if config.ErrorMapper != nil {
		httpHandler.errorMapper = config.ErrorMapper
	}

	// Set middleware
	httpHandler.middleware = config.Middleware

	return httpHandler
}

// Enhanced HTTP handler will be implemented in the next iteration

// Enhanced router implementation is in development and will be completed in the next iteration
// For now, the core middleware infrastructure is fully implemented and tested
