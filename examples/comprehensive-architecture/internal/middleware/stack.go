package middleware

import (
	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
)

// SetupMiddlewareStack creates the middleware stack for the application
func SetupMiddlewareStack() []typedhttp.MiddlewareEntry {
	return []typedhttp.MiddlewareEntry{
		// Layer 1: Request ID (highest priority)
		{
			Middleware: NewRequestIDMiddleware(),
			Config: typedhttp.MiddlewareConfig{
				Name:     "request_id",
				Priority: 100,
				Scope:    typedhttp.ScopeGlobal,
			},
		},

		// Layer 2: Response envelope (high priority)
		{
			Middleware: typedhttp.NewResponseEnvelopeMiddleware[any](
				typedhttp.WithRequestID(true),
				typedhttp.WithTimestamp(true),
				typedhttp.WithMeta(true),
			),
			Config: typedhttp.MiddlewareConfig{
				Name:     "response_envelope",
				Priority: 90,
				Scope:    typedhttp.ScopeGlobal,
			},
		},

		// Layer 3: Cache metadata (medium priority)
		{
			Middleware: NewCacheMetadataMiddleware[any](),
			Config: typedhttp.MiddlewareConfig{
				Name:     "cache_metadata",
				Priority: 50,
				Scope:    typedhttp.ScopeGlobal,
			},
		},

		// Layer 4: Audit logging (low priority)
		{
			Middleware: NewAuditMiddleware[any, any](),
			Config: typedhttp.MiddlewareConfig{
				Name:     "audit",
				Priority: 10,
				Scope:    typedhttp.ScopeGlobal,
			},
		},
	}
}