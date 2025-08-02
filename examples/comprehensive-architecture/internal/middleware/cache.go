package middleware

import (
	"context"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

// CacheMetadataMiddleware demonstrates middleware that modifies response structure
type CacheMetadataMiddleware[TResponse any] struct{}

// CachedResponse wraps a response with cache metadata
type CachedResponse[T any] struct {
	Data      T         `json:"data"`
	CachedAt  time.Time `json:"cached_at"`
	ExpiresAt time.Time `json:"expires_at"`
	CacheHit  bool      `json:"cache_hit"`
}

// NewCacheMetadataMiddleware creates a new cache metadata middleware
func NewCacheMetadataMiddleware[TResponse any]() *CacheMetadataMiddleware[TResponse] {
	return &CacheMetadataMiddleware[TResponse]{}
}

// After implements the TypedPostMiddleware interface
func (m *CacheMetadataMiddleware[TResponse]) After(ctx context.Context, resp *TResponse) (*CachedResponse[TResponse], error) {
	return &CachedResponse[TResponse]{
		Data:      *resp,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(5 * time.Minute),
		CacheHit:  false, // Simplified - would check actual cache
	}, nil
}

// ModifyResponseSchema implements the ResponseSchemaModifier interface
func (m *CacheMetadataMiddleware[TResponse]) ModifyResponseSchema(ctx context.Context, originalSchema *openapi3.SchemaRef) (*openapi3.SchemaRef, error) {
	return &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			Type:        &openapi3.Types{"object"},
			Description: "Response with cache metadata",
			Properties: map[string]*openapi3.SchemaRef{
				"data": {
					Value: &openapi3.Schema{
						Description: "The actual response data",
						OneOf:       []*openapi3.SchemaRef{originalSchema},
					},
				},
				"cached_at": {
					Value: &openapi3.Schema{
						Type:        &openapi3.Types{"string"},
						Format:      "date-time",
						Description: "When the response was cached",
					},
				},
				"expires_at": {
					Value: &openapi3.Schema{
						Type:        &openapi3.Types{"string"},
						Format:      "date-time",
						Description: "When the cache expires",
					},
				},
				"cache_hit": {
					Value: &openapi3.Schema{
						Type:        &openapi3.Types{"boolean"},
						Description: "Whether this was a cache hit",
					},
				},
			},
			Required: []string{"data", "cached_at", "expires_at", "cache_hit"},
		},
	}, nil
}