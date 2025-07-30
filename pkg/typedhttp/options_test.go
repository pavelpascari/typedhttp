package typedhttp_test

import (
	"net/http"
	"testing"

	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
	"github.com/stretchr/testify/assert"
)

func TestWithDecoder(t *testing.T) {
	decoder := typedhttp.NewJSONDecoder[TestRequest](nil)
	config := &typedhttp.HandlerConfig{}

	option := typedhttp.WithDecoder(decoder)
	option(config)

	assert.Equal(t, decoder, config.Decoder)
}

func TestWithEncoder(t *testing.T) {
	encoder := typedhttp.NewJSONEncoder[TestResponse]()
	config := &typedhttp.HandlerConfig{}

	option := typedhttp.WithEncoder(encoder)
	option(config)

	assert.Equal(t, encoder, config.Encoder)
}

func TestWithErrorMapper(t *testing.T) {
	errorMapper := &typedhttp.DefaultErrorMapper{}
	config := &typedhttp.HandlerConfig{}

	option := typedhttp.WithErrorMapper(errorMapper)
	option(config)

	assert.Equal(t, errorMapper, config.ErrorMapper)
}

func TestWithMiddleware(t *testing.T) {
	middleware1 := func(next http.Handler) http.Handler { return next }
	middleware2 := func(next http.Handler) http.Handler { return next }

	config := &typedhttp.HandlerConfig{}

	option := typedhttp.WithMiddleware(middleware1, middleware2)
	option(config)

	assert.Len(t, config.Middleware, 2)
}

func TestWithOpenAPI(t *testing.T) {
	metadata := typedhttp.OpenAPIMetadata{
		Summary:     "Test endpoint",
		Description: "A test endpoint for testing",
		Tags:        []string{"test"},
	}

	config := &typedhttp.HandlerConfig{}

	option := typedhttp.WithOpenAPI(&metadata)
	option(config)

	assert.Equal(t, metadata, config.Metadata)
}

func TestWithTags(t *testing.T) {
	config := &typedhttp.HandlerConfig{}

	option := typedhttp.WithTags("api", "v1", "users")
	option(config)

	expected := []string{"api", "v1", "users"}
	assert.Equal(t, expected, config.Metadata.Tags)
}

func TestWithSummary(t *testing.T) {
	config := &typedhttp.HandlerConfig{}

	option := typedhttp.WithSummary("Get user by ID")
	option(config)

	assert.Equal(t, "Get user by ID", config.Metadata.Summary)
}

func TestWithDescription(t *testing.T) {
	config := &typedhttp.HandlerConfig{}

	option := typedhttp.WithDescription("Retrieves a user by their unique identifier")
	option(config)

	assert.Equal(t, "Retrieves a user by their unique identifier", config.Metadata.Description)
}

func TestWithObservability(t *testing.T) {
	obsConfig := typedhttp.ObservabilityConfig{
		Tracing: true,
		Metrics: true,
		Logging: false,
		TraceAttributes: map[string]interface{}{
			"service": "user-api",
		},
		MetricLabels: map[string]string{
			"version": "v1",
		},
	}

	config := &typedhttp.HandlerConfig{}

	option := typedhttp.WithObservability(obsConfig)
	option(config)

	assert.Equal(t, obsConfig, config.Observability)
}

func TestWithDefaultObservability(t *testing.T) {
	config := &typedhttp.HandlerConfig{}

	option := typedhttp.WithDefaultObservability()
	option(config)

	assert.True(t, config.Observability.Tracing)
	assert.True(t, config.Observability.Metrics)
	assert.True(t, config.Observability.Logging)
}

func TestWithTracing(t *testing.T) {
	config := &typedhttp.HandlerConfig{}

	option := typedhttp.WithTracing()
	option(config)

	assert.True(t, config.Observability.Tracing)
}

func TestWithMetrics(t *testing.T) {
	config := &typedhttp.HandlerConfig{}

	option := typedhttp.WithMetrics()
	option(config)

	assert.True(t, config.Observability.Metrics)
}

func TestWithLogging(t *testing.T) {
	config := &typedhttp.HandlerConfig{}

	option := typedhttp.WithLogging()
	option(config)

	assert.True(t, config.Observability.Logging)
}

func TestWithTraceAttributes(t *testing.T) {
	attributes := map[string]interface{}{
		"service":   "user-api",
		"version":   "v1.0.0",
		"operation": "get_user",
	}

	config := &typedhttp.HandlerConfig{}

	option := typedhttp.WithTraceAttributes(attributes)
	option(config)

	assert.Equal(t, attributes, config.Observability.TraceAttributes)
}

func TestWithTraceAttributes_Merge(t *testing.T) {
	config := &typedhttp.HandlerConfig{
		Observability: typedhttp.ObservabilityConfig{
			TraceAttributes: map[string]interface{}{
				"existing": "value",
			},
		},
	}

	newAttributes := map[string]interface{}{
		"service": "user-api",
		"version": "v1.0.0",
	}

	option := typedhttp.WithTraceAttributes(newAttributes)
	option(config)

	expected := map[string]interface{}{
		"existing": "value",
		"service":  "user-api",
		"version":  "v1.0.0",
	}

	assert.Equal(t, expected, config.Observability.TraceAttributes)
}

func TestWithMetricLabels(t *testing.T) {
	labels := map[string]string{
		"service": "user-api",
		"version": "v1",
		"env":     "prod",
	}

	config := &typedhttp.HandlerConfig{}

	option := typedhttp.WithMetricLabels(labels)
	option(config)

	assert.Equal(t, labels, config.Observability.MetricLabels)
}

func TestWithMetricLabels_Merge(t *testing.T) {
	config := &typedhttp.HandlerConfig{
		Observability: typedhttp.ObservabilityConfig{
			MetricLabels: map[string]string{
				"existing": "value",
			},
		},
	}

	newLabels := map[string]string{
		"service": "user-api",
		"version": "v1",
	}

	option := typedhttp.WithMetricLabels(newLabels)
	option(config)

	expected := map[string]string{
		"existing": "value",
		"service":  "user-api",
		"version":  "v1",
	}

	assert.Equal(t, expected, config.Observability.MetricLabels)
}

func TestCombinedOptions(t *testing.T) {
	decoder := typedhttp.NewJSONDecoder[TestRequest](nil)
	encoder := typedhttp.NewJSONEncoder[TestResponse]()
	errorMapper := &typedhttp.DefaultErrorMapper{}

	config := &typedhttp.HandlerConfig{}

	options := []typedhttp.HandlerOption{
		typedhttp.WithDecoder(decoder),
		typedhttp.WithEncoder(encoder),
		typedhttp.WithErrorMapper(errorMapper),
		typedhttp.WithTags("api", "users"),
		typedhttp.WithSummary("Create user"),
		typedhttp.WithDefaultObservability(),
	}

	for _, option := range options {
		option(config)
	}

	assert.Equal(t, decoder, config.Decoder)
	assert.Equal(t, encoder, config.Encoder)
	assert.Equal(t, errorMapper, config.ErrorMapper)
	assert.Equal(t, []string{"api", "users"}, config.Metadata.Tags)
	assert.Equal(t, "Create user", config.Metadata.Summary)
	assert.True(t, config.Observability.Tracing)
	assert.True(t, config.Observability.Metrics)
	assert.True(t, config.Observability.Logging)
}
