package typedhttp

// WithDecoder sets a custom request decoder for the handler.
func WithDecoder[T any](decoder RequestDecoder[T]) HandlerOption {
	return func(cfg *HandlerConfig) {
		cfg.Decoder = decoder
	}
}

// WithEncoder sets a custom response encoder for the handler.
func WithEncoder[T any](encoder ResponseEncoder[T]) HandlerOption {
	return func(cfg *HandlerConfig) {
		cfg.Encoder = encoder
	}
}

// WithErrorMapper sets a custom error mapper for the handler.
func WithErrorMapper(mapper ErrorMapper) HandlerOption {
	return func(cfg *HandlerConfig) {
		cfg.ErrorMapper = mapper
	}
}

// WithMiddleware adds middleware to the handler chain.
func WithMiddleware(middleware ...Middleware) HandlerOption {
	return func(cfg *HandlerConfig) {
		cfg.Middleware = append(cfg.Middleware, middleware...)
	}
}

// WithOpenAPI sets OpenAPI metadata for the handler.
func WithOpenAPI(metadata *OpenAPIMetadata) HandlerOption {
	return func(cfg *HandlerConfig) {
		cfg.Metadata = *metadata
	}
}

// WithTags sets OpenAPI tags for the handler.
func WithTags(tags ...string) HandlerOption {
	return func(cfg *HandlerConfig) {
		cfg.Metadata.Tags = tags
	}
}

// WithSummary sets the OpenAPI summary for the handler.
func WithSummary(summary string) HandlerOption {
	return func(cfg *HandlerConfig) {
		cfg.Metadata.Summary = summary
	}
}

// WithDescription sets the OpenAPI description for the handler.
func WithDescription(description string) HandlerOption {
	return func(cfg *HandlerConfig) {
		cfg.Metadata.Description = description
	}
}

// WithObservability sets observability configuration for the handler.
func WithObservability(config ObservabilityConfig) HandlerOption {
	return func(cfg *HandlerConfig) {
		cfg.Observability = config
	}
}

// WithDefaultObservability enables default observability features.
func WithDefaultObservability() HandlerOption {
	return WithObservability(ObservabilityConfig{
		Tracing: true,
		Metrics: true,
		Logging: true,
	})
}

// WithTracing enables request tracing for the handler.
func WithTracing() HandlerOption {
	return func(cfg *HandlerConfig) {
		cfg.Observability.Tracing = true
	}
}

// WithMetrics enables metrics collection for the handler.
func WithMetrics() HandlerOption {
	return func(cfg *HandlerConfig) {
		cfg.Observability.Metrics = true
	}
}

// WithLogging enables structured logging for the handler.
func WithLogging() HandlerOption {
	return func(cfg *HandlerConfig) {
		cfg.Observability.Logging = true
	}
}

// WithTypedPreMiddleware adds a typed pre-middleware to the handler.
func WithTypedPreMiddleware[TRequest any](middleware TypedPreMiddleware[TRequest]) HandlerOption {
	return func(cfg *HandlerConfig) {
		entry := MiddlewareEntry{
			Middleware: middleware,
			Config: MiddlewareConfig{
				Name:  "typed_pre_middleware",
				Scope: ScopeHandler,
			},
		}
		cfg.TypedMiddleware = append(cfg.TypedMiddleware, entry)
	}
}

// WithTypedPostMiddleware adds a typed post-middleware to the handler.
func WithTypedPostMiddleware[TResponse any](middleware TypedPostMiddleware[TResponse]) HandlerOption {
	return func(cfg *HandlerConfig) {
		entry := MiddlewareEntry{
			Middleware: middleware,
			Config: MiddlewareConfig{
				Name:  "typed_post_middleware",
				Scope: ScopeHandler,
			},
		}
		cfg.TypedMiddleware = append(cfg.TypedMiddleware, entry)
	}
}

// WithTypedFullMiddleware adds a typed full middleware to the handler.
func WithTypedFullMiddleware[TRequest, TResponse any](middleware TypedMiddleware[TRequest, TResponse]) HandlerOption {
	return func(cfg *HandlerConfig) {
		entry := MiddlewareEntry{
			Middleware: middleware,
			Config: MiddlewareConfig{
				Name:  "typed_full_middleware",
				Scope: ScopeHandler,
			},
		}
		cfg.TypedMiddleware = append(cfg.TypedMiddleware, entry)
	}
}

// WithTraceAttributes adds custom trace attributes.
func WithTraceAttributes(attributes map[string]interface{}) HandlerOption {
	return func(cfg *HandlerConfig) {
		if cfg.Observability.TraceAttributes == nil {
			cfg.Observability.TraceAttributes = make(map[string]interface{})
		}
		for k, v := range attributes {
			cfg.Observability.TraceAttributes[k] = v
		}
	}
}

// WithMetricLabels adds custom metric labels.
func WithMetricLabels(labels map[string]string) HandlerOption {
	return func(cfg *HandlerConfig) {
		if cfg.Observability.MetricLabels == nil {
			cfg.Observability.MetricLabels = make(map[string]string)
		}
		for k, v := range labels {
			cfg.Observability.MetricLabels[k] = v
		}
	}
}
