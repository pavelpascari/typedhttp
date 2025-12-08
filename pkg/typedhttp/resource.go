package typedhttp

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Error variables for static error handling.
var (
	ErrMethodNotFound               = errors.New("method not found on service")
	ErrMethodInvalidReturnSignature = errors.New("method should return (response, error)")
)

// ResourceService represents a service that can handle CRUD operations for a resource.
// This interface eliminates the need for individual handler wrappers by providing
// a unified interface for resource operations.
type ResourceService interface {
	// GetType and ListType return the types used for single and list operations
	GetType() (requestType, responseType reflect.Type)
	ListType() (requestType, responseType reflect.Type)
	CreateType() (requestType, responseType reflect.Type)
	UpdateType() (requestType, responseType reflect.Type)
	DeleteType() (requestType, responseType reflect.Type)
}

// CRUDService provides CRUD operations for a resource.
// Business logic services should implement this interface to be used with Resource registration.
type CRUDService[TGetReq, TGetResp, TListReq, TListResp, TCreateReq, TCreateResp, TUpdateReq, TUpdateResp, TDeleteReq, TDeleteResp any] interface {
	Get(ctx context.Context, req TGetReq) (TGetResp, error)
	List(ctx context.Context, req TListReq) (TListResp, error)
	Create(ctx context.Context, req TCreateReq) (TCreateResp, error)
	Update(ctx context.Context, req TUpdateReq) (TUpdateResp, error)
	Delete(ctx context.Context, req TDeleteReq) (TDeleteResp, error)
}

// OperationConfig defines configuration for individual resource operations.
type OperationConfig struct {
	Summary     string   `json:"summary,omitempty"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Enabled     bool     `json:"enabled"`
	Path        string   `json:"path,omitempty"` // Custom path override
}

// ResourceConfig defines configuration for registering a complete resource.
type ResourceConfig struct {
	// Base configuration
	Tags       []string `json:"tags,omitempty"`
	PathPrefix string   `json:"path_prefix,omitempty"` // e.g., "/api/v1"

	// Operation-specific configurations
	Operations map[string]OperationConfig `json:"operations,omitempty"`

	// Middleware
	Middleware []MiddlewareEntry `json:"middleware,omitempty"`

	// Default options for all operations
	DefaultOptions []HandlerOption `json:"-"`
}

// DomainRouter extends TypedRouter with resource-based registration capabilities.
type DomainRouter struct {
	*TypedRouter
	pathPrefix string
	middleware []MiddlewareEntry
}

// NewDomainRouter creates a new domain router with optional path prefix and middleware.
func NewDomainRouter(pathPrefix string, middleware ...MiddlewareEntry) *DomainRouter {
	return &DomainRouter{
		TypedRouter: NewRouter(),
		pathPrefix:  pathPrefix,
		middleware:  middleware,
	}
}

// Resource registers a complete CRUD resource with automatic operation mapping.
// This eliminates the need for individual handler wrappers and repetitive registration code.
func Resource[TGetReq, TGetResp, TListReq, TListResp, TCreateReq, TCreateResp, TUpdateReq, TUpdateResp, TDeleteReq, TDeleteResp any](
	router *DomainRouter,
	path string,
	service CRUDService[TGetReq, TGetResp, TListReq, TListResp, TCreateReq, TCreateResp, TUpdateReq, TUpdateResp, TDeleteReq, TDeleteResp],
	config ResourceConfig,
) {
	// Apply path prefix
	fullPath := router.pathPrefix + path

	// Merge middleware
	allMiddleware := append(router.middleware, config.Middleware...)

	// Merge default options with middleware
	defaultOpts := append(config.DefaultOptions, withMiddlewareEntries(allMiddleware))

	// Register individual operations based on configuration
	operations := map[string]struct {
		method string
		path   string
		setup  func()
	}{
		"GET": {
			method: "GET",
			path:   fullPath + "/{id}",
			setup: func() {
				handler := &resourceHandler[TGetReq, TGetResp]{
					service:   service,
					operation: "Get",
				}
				opts := append(defaultOpts, buildOperationOptions("GET", config)...)
				GET(router.TypedRouter, fullPath+"/{id}", handler, opts...)
			},
		},
		"LIST": {
			method: "GET",
			path:   fullPath,
			setup: func() {
				handler := &resourceHandler[TListReq, TListResp]{
					service:   service,
					operation: "List",
				}
				opts := append(defaultOpts, buildOperationOptions("LIST", config)...)
				GET(router.TypedRouter, fullPath, handler, opts...)
			},
		},
		"POST": {
			method: "POST",
			path:   fullPath,
			setup: func() {
				handler := &resourceHandler[TCreateReq, TCreateResp]{
					service:   service,
					operation: "Create",
				}
				opts := append(defaultOpts, buildOperationOptions("POST", config)...)
				POST(router.TypedRouter, fullPath, handler, opts...)
			},
		},
		"PUT": {
			method: "PUT",
			path:   fullPath + "/{id}",
			setup: func() {
				handler := &resourceHandler[TUpdateReq, TUpdateResp]{
					service:   service,
					operation: "Update",
				}
				opts := append(defaultOpts, buildOperationOptions("PUT", config)...)
				PUT(router.TypedRouter, fullPath+"/{id}", handler, opts...)
			},
		},
		"DELETE": {
			method: "DELETE",
			path:   fullPath + "/{id}",
			setup: func() {
				handler := &resourceHandler[TDeleteReq, TDeleteResp]{
					service:   service,
					operation: "Delete",
				}
				opts := append(defaultOpts, buildOperationOptions("DELETE", config)...)
				DELETE(router.TypedRouter, fullPath+"/{id}", handler, opts...)
			},
		},
	}

	// Register enabled operations
	for op, setup := range operations {
		if shouldEnableOperation(op, config) {
			setup.setup()
		}
	}
}

// resourceHandler is a generic handler that delegates to service methods.
// This eliminates the need for individual handler wrappers.
type resourceHandler[TReq, TResp any] struct {
	service   interface{}
	operation string
}

// Handle implements the Handler interface by delegating to the appropriate service method.
func (h *resourceHandler[TReq, TResp]) Handle(ctx context.Context, req TReq) (TResp, error) {
	var zero TResp

	// Use reflection to call the appropriate method on the service
	serviceValue := reflect.ValueOf(h.service)
	method := serviceValue.MethodByName(h.operation)

	if !method.IsValid() {
		return zero, fmt.Errorf("%w: %s", ErrMethodNotFound, h.operation)
	}

	// Call the method with context and request
	args := []reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(req),
	}

	results := method.Call(args)
	if len(results) != 2 {
		return zero, fmt.Errorf("%w: %s", ErrMethodInvalidReturnSignature, h.operation)
	}

	// Check for error
	if !results[1].IsNil() {
		err := results[1].Interface().(error)
		return zero, err
	}

	// Return response
	response := results[0].Interface().(TResp)
	return response, nil
}

// Helper functions

func shouldEnableOperation(operation string, config ResourceConfig) bool {
	if opConfig, exists := config.Operations[operation]; exists {
		return opConfig.Enabled
	}
	// Enable all operations by default
	return true
}

func buildOperationOptions(operation string, config ResourceConfig) []HandlerOption {
	var opts []HandlerOption

	// Add base tags
	if len(config.Tags) > 0 {
		opts = append(opts, WithTags(config.Tags...))
	}

	// Add operation-specific configuration
	if opConfig, exists := config.Operations[operation]; exists {
		if opConfig.Summary != "" {
			opts = append(opts, WithSummary(opConfig.Summary))
		}
		if opConfig.Description != "" {
			opts = append(opts, WithDescription(opConfig.Description))
		}
		if len(opConfig.Tags) > 0 {
			opts = append(opts, WithTags(opConfig.Tags...))
		}
	} else {
		// Add default summary based on operation
		switch operation {
		case "GET":
			opts = append(opts, WithSummary(fmt.Sprintf("Get %s by ID", inferResourceName(config))))
		case "LIST":
			opts = append(opts, WithSummary(fmt.Sprintf("List %s", inferResourceName(config))))
		case "POST":
			opts = append(opts, WithSummary(fmt.Sprintf("Create %s", inferResourceName(config))))
		case "PUT":
			opts = append(opts, WithSummary(fmt.Sprintf("Update %s", inferResourceName(config))))
		case "DELETE":
			opts = append(opts, WithSummary(fmt.Sprintf("Delete %s", inferResourceName(config))))
		}
	}

	return opts
}

func inferResourceName(config ResourceConfig) string {
	if len(config.Tags) > 0 {
		return strings.TrimSuffix(config.Tags[0], "s") // Remove trailing 's' for singular form
	}
	return "resource"
}

func withMiddlewareEntries(middleware []MiddlewareEntry) HandlerOption {
	return func(config *HandlerConfig) {
		config.TypedMiddleware = middleware
	}
}
