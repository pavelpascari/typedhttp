package typedhttp

import (
	"context"
	"net/http"
	"sort"
	"sync"
)

// Typed middleware interfaces for different phases

// TypedPreMiddleware operates on decoded request data before handler execution.
type TypedPreMiddleware[TRequest any] interface {
	Before(ctx context.Context, req *TRequest) (context.Context, error)
}

// TypedPostMiddleware operates on response data after handler execution.
type TypedPostMiddleware[TResponse any] interface {
	After(ctx context.Context, resp *TResponse) (*TResponse, error)
}

// TypedMiddleware provides full lifecycle hooks with access to both request and response.
type TypedMiddleware[TRequest, TResponse any] interface {
	Before(ctx context.Context, req *TRequest) (context.Context, error)
	After(ctx context.Context, req *TRequest, resp *TResponse, err error) (*TResponse, error)
}

// ConditionalFunc determines whether middleware should execute for a given request.
type ConditionalFunc func(*http.Request) bool

// MiddlewareScope defines the application scope of middleware.
type MiddlewareScope int

const (
	ScopeGlobal  MiddlewareScope = iota // Applied to all handlers
	ScopeGroup                          // Applied to route groups
	ScopeHandler                        // Applied to specific handlers
)

// MiddlewareConfig contains configuration options for middleware
type MiddlewareConfig struct {
	Priority    int                    // Execution order priority (-100 to 100)
	Conditional ConditionalFunc        // Optional condition for execution
	Scope       MiddlewareScope        // Application scope
	Name        string                 // Middleware identification
	Metadata    map[string]interface{} // Custom metadata
}

// MiddlewareEntry wraps middleware with its configuration.
type MiddlewareEntry struct {
	Middleware interface{}      // Middleware, TypedPreMiddleware, TypedPostMiddleware, or TypedMiddleware
	Config     MiddlewareConfig // Configuration options
}

// MiddlewareRegistry manages middleware chains for different scopes.
type MiddlewareRegistry struct {
	global   []MiddlewareEntry
	groups   map[string][]MiddlewareEntry
	handlers map[string][]MiddlewareEntry
	mu       sync.RWMutex
}

// NewMiddlewareRegistry creates a new middleware registry.
func NewMiddlewareRegistry() *MiddlewareRegistry {
	return &MiddlewareRegistry{
		global:   make([]MiddlewareEntry, 0),
		groups:   make(map[string][]MiddlewareEntry),
		handlers: make(map[string][]MiddlewareEntry),
	}
}

// RegisterGlobal registers global middleware.
func (r *MiddlewareRegistry) RegisterGlobal(entry MiddlewareEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.global = append(r.global, entry)
}

// RegisterGroup registers middleware for a route group.
func (r *MiddlewareRegistry) RegisterGroup(pattern string, entry MiddlewareEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.groups[pattern] == nil {
		r.groups[pattern] = make([]MiddlewareEntry, 0)
	}
	r.groups[pattern] = append(r.groups[pattern], entry)
}

// RegisterHandler registers middleware for a specific handler.
func (r *MiddlewareRegistry) RegisterHandler(pattern string, entry MiddlewareEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.handlers[pattern] == nil {
		r.handlers[pattern] = make([]MiddlewareEntry, 0)
	}
	r.handlers[pattern] = append(r.handlers[pattern], entry)
}

// GetGlobal returns all global middleware entries.
func (r *MiddlewareRegistry) GetGlobal() []MiddlewareEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]MiddlewareEntry, len(r.global))
	copy(result, r.global)

	return result
}

// GetGroups returns all group middleware entries.
func (r *MiddlewareRegistry) GetGroups() map[string][]MiddlewareEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string][]MiddlewareEntry)
	for k, v := range r.groups {
		result[k] = make([]MiddlewareEntry, len(v))
		copy(result[k], v)
	}

	return result
}

// GetHandlers returns all handler middleware entries.
func (r *MiddlewareRegistry) GetHandlers() map[string][]MiddlewareEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string][]MiddlewareEntry)
	for k, v := range r.handlers {
		result[k] = make([]MiddlewareEntry, len(v))
		copy(result[k], v)
	}

	return result
}

// MiddlewareBuilder provides a fluent API for building middleware chains.
type MiddlewareBuilder struct {
	entries []MiddlewareEntry
}

// NewMiddlewareBuilder creates a new middleware builder.
func NewMiddlewareBuilder() *MiddlewareBuilder {
	return &MiddlewareBuilder{entries: make([]MiddlewareEntry, 0)}
}

// MiddlewareOption configures middleware.
type MiddlewareOption func(*MiddlewareConfig)

// WithName sets the middleware name.
func WithName(name string) MiddlewareOption {
	return func(config *MiddlewareConfig) {
		config.Name = name
	}
}

// WithPriority sets the middleware priority.
func WithPriority(priority int) MiddlewareOption {
	return func(config *MiddlewareConfig) {
		config.Priority = priority
	}
}

// WithScope sets the middleware scope.
func WithScope(scope MiddlewareScope) MiddlewareOption {
	return func(config *MiddlewareConfig) {
		config.Scope = scope
	}
}

// WithCondition sets a conditional function for middleware execution.
func WithCondition(condition ConditionalFunc) MiddlewareOption {
	return func(config *MiddlewareConfig) {
		config.Conditional = condition
	}
}

// WithMetadata sets the middleware metadata.
func WithMetadata(metadata map[string]interface{}) MiddlewareOption {
	return func(config *MiddlewareConfig) {
		config.Metadata = metadata
	}
}

// WithMetadataKey adds a single key-value pair to middleware metadata.
func WithMetadataKey(key string, value interface{}) MiddlewareOption {
	return func(config *MiddlewareConfig) {
		if config.Metadata == nil {
			config.Metadata = make(map[string]interface{})
		}
		config.Metadata[key] = value
	}
}

// Add adds middleware to the builder with options.
func (b *MiddlewareBuilder) Add(mw interface{}, opts ...MiddlewareOption) *MiddlewareBuilder {
	config := MiddlewareConfig{Priority: 0, Scope: ScopeHandler}
	for _, opt := range opts {
		opt(&config)
	}

	b.entries = append(b.entries, MiddlewareEntry{
		Middleware: mw,
		Config:     config,
	})

	return b
}

// OnlyFor adds a condition to the last added middleware.
func (b *MiddlewareBuilder) OnlyFor(condition ConditionalFunc) *MiddlewareBuilder {
	if len(b.entries) > 0 {
		b.entries[len(b.entries)-1].Config.Conditional = condition
	}

	return b
}

// WithPriority sets the priority of the last added middleware.
func (b *MiddlewareBuilder) WithPriority(priority int) *MiddlewareBuilder {
	if len(b.entries) > 0 {
		b.entries[len(b.entries)-1].Config.Priority = priority
	}

	return b
}

// Build returns the middleware entries sorted by priority (highest first).
func (b *MiddlewareBuilder) Build() []MiddlewareEntry {
	// Sort by priority (higher priority executes first).
	sort.Slice(b.entries, func(i, j int) bool {
		return b.entries[i].Config.Priority > b.entries[j].Config.Priority
	})

	return b.entries
}

// TypedMiddlewareChain contains typed middleware organized by phase.
type TypedMiddlewareChain[TRequest, TResponse any] struct {
	preMiddleware  []TypedPreMiddleware[TRequest]
	postMiddleware []TypedPostMiddleware[TResponse]
	fullMiddleware []TypedMiddleware[TRequest, TResponse]
}

// extractTypedMiddleware extracts typed middleware from middleware entries.
func extractTypedMiddleware[TRequest, TResponse any](entries []MiddlewareEntry) TypedMiddlewareChain[TRequest, TResponse] {
	chain := TypedMiddlewareChain[TRequest, TResponse]{
		preMiddleware:  make([]TypedPreMiddleware[TRequest], 0),
		postMiddleware: make([]TypedPostMiddleware[TResponse], 0),
		fullMiddleware: make([]TypedMiddleware[TRequest, TResponse], 0),
	}

	for _, entry := range entries {
		// Check for the most specific interface first (TypedMiddleware).
		if fullMW, ok := entry.Middleware.(TypedMiddleware[TRequest, TResponse]); ok {
			chain.fullMiddleware = append(chain.fullMiddleware, fullMW)
		} else if preMW, ok := entry.Middleware.(TypedPreMiddleware[TRequest]); ok {
			chain.preMiddleware = append(chain.preMiddleware, preMW)
		} else if postMW, ok := entry.Middleware.(TypedPostMiddleware[TResponse]); ok {
			chain.postMiddleware = append(chain.postMiddleware, postMW)
		}
	}

	return chain
}
