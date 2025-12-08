package typedhttp

import (
	"net/http"
	"strings"
)

// ComposableRouter extends TypedRouter with composition capabilities.
// This enables large teams to work on separate sub-routers and compose them together.
type ComposableRouter struct {
	*TypedRouter
	prefix     string
	middleware []MiddlewareEntry
	subrouters []*mountedRouter
}

// mountedRouter represents a sub-router mounted at a specific prefix.
type mountedRouter struct {
	prefix     string
	router     *ComposableRouter
	middleware []MiddlewareEntry
}

// NewComposableRouter creates a new composable router.
func NewComposableRouter(prefix string, middleware ...MiddlewareEntry) *ComposableRouter {
	return &ComposableRouter{
		TypedRouter: NewRouter(),
		prefix:      prefix,
		middleware:  middleware,
		subrouters:  make([]*mountedRouter, 0),
	}
}

// Mount adds a sub-router at the specified prefix with optional additional middleware.
// This enables team-based development where different teams can work on separate routers.
//
// Example:
//
//	app := NewComposableRouter("")
//	userTeam := NewComposableRouter("/users")  // Team: Identity
//	orderTeam := NewComposableRouter("/orders") // Team: Commerce
//
//	app.Mount("/api/v1", userTeam, orderTeam)
func (r *ComposableRouter) Mount(prefix string, routers ...*ComposableRouter) {
	for _, router := range routers {
		mounted := &mountedRouter{
			prefix:     r.combinePaths(prefix, router.prefix),
			router:     router,
			middleware: append(r.middleware, router.middleware...), // Inherit parent middleware
		}
		r.subrouters = append(r.subrouters, mounted)
	}
}

// MountWithMiddleware mounts sub-routers with additional middleware applied to all of them.
// This is useful for applying common middleware (like authentication) to a group of services.
//
// Example:
//
//	app.MountWithMiddleware("/admin", []MiddlewareEntry{adminAuth}, adminUserRouter, adminSystemRouter)
func (r *ComposableRouter) MountWithMiddleware(prefix string, middleware []MiddlewareEntry, routers ...*ComposableRouter) {
	for _, router := range routers {
		mounted := &mountedRouter{
			prefix:     r.combinePaths(prefix, router.prefix),
			router:     router,
			middleware: append(append(r.middleware, middleware...), router.middleware...), // Parent + mount + router middleware
		}
		r.subrouters = append(r.subrouters, mounted)
	}
}

// Finalize builds the final router by integrating all mounted sub-routers.
// This should be called after all mounting is complete.
func (r *ComposableRouter) Finalize() *TypedRouter {
	finalRouter := NewRouter()

	// Add handlers from this router
	r.addHandlersToFinal(finalRouter, r.prefix, r.middleware)

	// Add handlers from all mounted sub-routers
	for _, mounted := range r.subrouters {
		mounted.router.addHandlersToFinal(finalRouter, mounted.prefix, mounted.middleware)
	}

	return finalRouter
}

// addHandlersToFinal recursively adds handlers to the final router with proper prefixes and middleware.
func (r *ComposableRouter) addHandlersToFinal(final *TypedRouter, pathPrefix string, inheritedMiddleware []MiddlewareEntry) {
	// Add direct handlers from this router
	for i := range r.TypedRouter.handlers {
		handler := &r.TypedRouter.handlers[i]
		finalPath := r.combinePaths(pathPrefix, handler.Path)
		finalMiddleware := make([]MiddlewareEntry, 0, len(inheritedMiddleware)+len(handler.MiddlewareEntries))
		finalMiddleware = append(finalMiddleware, inheritedMiddleware...)
		finalMiddleware = append(finalMiddleware, handler.MiddlewareEntries...)

		// Create new handler registration with updated path and middleware
		finalHandler := HandlerRegistration{
			Method:            handler.Method,
			Path:              finalPath,
			RequestType:       handler.RequestType,
			ResponseType:      handler.ResponseType,
			Metadata:          handler.Metadata,
			Config:            handler.Config,
			MiddlewareEntries: finalMiddleware,
		}

		final.handlers = append(final.handlers, finalHandler)

		// Register with HTTP mux using the final path
		pattern := handler.Method + " " + finalPath
		// Note: We need to re-create the HTTP handler with the new middleware
		// This is a simplified version - in practice, we'd need to reconstruct the HTTPHandler
		final.mux.HandleFunc(pattern, func(w http.ResponseWriter, req *http.Request) {
			// Placeholder - would need to reconstruct handler with proper middleware
			http.Error(w, "Handler reconstruction needed", http.StatusNotImplemented)
		})
	}

	// Recursively add handlers from sub-routers
	for _, mounted := range r.subrouters {
		combinedPrefix := r.combinePaths(pathPrefix, mounted.prefix)
		combinedMiddleware := make([]MiddlewareEntry, 0, len(inheritedMiddleware)+len(mounted.middleware))
		combinedMiddleware = append(combinedMiddleware, inheritedMiddleware...)
		combinedMiddleware = append(combinedMiddleware, mounted.middleware...)
		mounted.router.addHandlersToFinal(final, combinedPrefix, combinedMiddleware)
	}
}

// GetAllHandlers returns all handlers from this router and all mounted sub-routers.
// This is useful for OpenAPI generation and debugging.
func (r *ComposableRouter) GetAllHandlers() []HandlerRegistration {
	return r.getAllHandlersWithPrefix("")
}

// getAllHandlersWithPrefix is a helper that collects handlers with the given prefix.
func (r *ComposableRouter) getAllHandlersWithPrefix(pathPrefix string) []HandlerRegistration {
	var allHandlers []HandlerRegistration

	// For direct handlers in this router, use the provided pathPrefix + this router's prefix
	// But only if this router isn't mounted (pathPrefix is empty) or if we need to add our prefix
	var currentPrefix string
	if pathPrefix == "" {
		// This is the root call, so use our prefix
		currentPrefix = r.prefix
	} else {
		// This router is mounted, and pathPrefix already includes our prefix, so just use pathPrefix
		currentPrefix = pathPrefix
	}

	// Collect handlers from this router
	for i := range r.TypedRouter.handlers {
		handler := &r.TypedRouter.handlers[i]
		finalHandler := *handler
		finalHandler.Path = r.combinePaths(currentPrefix, handler.Path)
		finalHandler.MiddlewareEntries = make([]MiddlewareEntry, 0, len(r.middleware)+len(handler.MiddlewareEntries))
		finalHandler.MiddlewareEntries = append(finalHandler.MiddlewareEntries, r.middleware...)
		finalHandler.MiddlewareEntries = append(finalHandler.MiddlewareEntries, handler.MiddlewareEntries...)
		allHandlers = append(allHandlers, finalHandler)
	}

	// Recursively collect handlers from sub-routers
	for _, mounted := range r.subrouters {
		// For mounted routers, we need to combine the current pathPrefix with the mounted prefix
		// because the mounted prefix is relative to where the router was mounted, not absolute
		var fullMountedPrefix string
		if pathPrefix == "" {
			fullMountedPrefix = mounted.prefix
		} else {
			fullMountedPrefix = r.combinePaths(pathPrefix, mounted.prefix)
		}

		subHandlers := mounted.router.getAllHandlersWithPrefix(fullMountedPrefix)
		for i := range subHandlers {
			handler := &subHandlers[i]
			finalHandler := *handler
			// Path is already calculated in the recursive call, just add middleware
			finalHandler.MiddlewareEntries = make([]MiddlewareEntry, 0, len(mounted.middleware)+len(handler.MiddlewareEntries))
			finalHandler.MiddlewareEntries = append(finalHandler.MiddlewareEntries, mounted.middleware...)
			finalHandler.MiddlewareEntries = append(finalHandler.MiddlewareEntries, handler.MiddlewareEntries...)
			allHandlers = append(allHandlers, finalHandler)
		}
	}

	return allHandlers
}

// combinePaths safely combines two URL paths, handling slashes correctly.
func (r *ComposableRouter) combinePaths(prefix, path string) string {
	// Handle empty cases
	if prefix == "" {
		return path
	}
	if path == "" {
		return prefix
	}

	// Ensure prefix starts with / but doesn't end with /
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	prefix = strings.TrimSuffix(prefix, "/")

	// Ensure path starts with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return prefix + path
}

// TeamRouter creates a domain router specifically designed for team ownership.
// This includes team-specific naming conventions and isolation patterns.
func TeamRouter(teamName, pathPrefix string, middleware ...MiddlewareEntry) *ComposableRouter {
	// Add team identification to middleware for observability
	teamMiddleware := append([]MiddlewareEntry{
		{
			Middleware: &teamIdentificationMiddleware{teamName: teamName},
			Config: MiddlewareConfig{
				Name:     "team_identification",
				Priority: 95,
				Scope:    ScopeGlobal,
			},
		},
	}, middleware...)

	return NewComposableRouter(pathPrefix, teamMiddleware...)
}

// teamIdentificationMiddleware adds team identification headers for observability.
type teamIdentificationMiddleware struct {
	teamName string
}

func (m *teamIdentificationMiddleware) Before(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add team identification to response headers for debugging
		w.Header().Set("X-Team", m.teamName)
		next.ServeHTTP(w, r)
	})
}

// DomainComposition provides a higher-level API for organizing routers by business domain.
type DomainComposition struct {
	domains map[string]*ComposableRouter
	app     *ComposableRouter
}

// NewDomainComposition creates a new domain-based composition system.
func NewDomainComposition() *DomainComposition {
	return &DomainComposition{
		domains: make(map[string]*ComposableRouter),
		app:     NewComposableRouter(""),
	}
}

// AddDomain registers a domain router (e.g., "users", "orders", "products").
func (dc *DomainComposition) AddDomain(domain string, router *ComposableRouter) {
	dc.domains[domain] = router
}

// AddTeamDomain creates and registers a team-owned domain router.
func (dc *DomainComposition) AddTeamDomain(teamName, domain, pathPrefix string, middleware ...MiddlewareEntry) *ComposableRouter {
	router := TeamRouter(teamName, pathPrefix, middleware...)
	dc.domains[domain] = router
	return router
}

// Compose creates the final application router by mounting all domains.
func (dc *DomainComposition) Compose(apiPrefix string) *TypedRouter {
	var routers []*ComposableRouter
	for _, router := range dc.domains {
		routers = append(routers, router)
	}

	dc.app.Mount(apiPrefix, routers...)
	return dc.app.Finalize()
}

// GetDomain returns a domain router for additional configuration.
func (dc *DomainComposition) GetDomain(domain string) *ComposableRouter {
	return dc.domains[domain]
}
