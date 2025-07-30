package typedhttp_test

import (
	"net/http"
	"testing"

	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
	"github.com/stretchr/testify/assert"
)

func TestNewRouter(t *testing.T) {
	router := typedhttp.NewRouter()

	assert.NotNil(t, router)
	assert.Implements(t, (*http.Handler)(nil), router)
}

func TestTypedRouter_RegisterHandler(t *testing.T) {
	router := typedhttp.NewRouter()
	handler := &TestHandler{}

	typedhttp.RegisterHandler(router, "GET", "/test", handler)

	handlers := router.GetHandlers()
	assert.Len(t, handlers, 1)
	assert.Equal(t, "GET", handlers[0].Method)
	assert.Equal(t, "/test", handlers[0].Path)
}

func TestTypedRouter_GET(t *testing.T) {
	router := typedhttp.NewRouter()
	handler := &TestHandler{}

	typedhttp.GET(router, "/users", handler)

	handlers := router.GetHandlers()
	assert.Len(t, handlers, 1)
	assert.Equal(t, "GET", handlers[0].Method)
	assert.Equal(t, "/users", handlers[0].Path)
}

func TestTypedRouter_POST(t *testing.T) {
	router := typedhttp.NewRouter()
	handler := &TestHandler{}

	typedhttp.POST(router, "/users", handler)

	handlers := router.GetHandlers()
	assert.Len(t, handlers, 1)
	assert.Equal(t, "POST", handlers[0].Method)
	assert.Equal(t, "/users", handlers[0].Path)
}

func TestTypedRouter_PUT(t *testing.T) {
	router := typedhttp.NewRouter()
	handler := &TestHandler{}

	typedhttp.PUT(router, "/users/123", handler)

	handlers := router.GetHandlers()
	assert.Len(t, handlers, 1)
	assert.Equal(t, "PUT", handlers[0].Method)
	assert.Equal(t, "/users/123", handlers[0].Path)
}

func TestTypedRouter_PATCH(t *testing.T) {
	router := typedhttp.NewRouter()
	handler := &TestHandler{}

	typedhttp.PATCH(router, "/users/123", handler)

	handlers := router.GetHandlers()
	assert.Len(t, handlers, 1)
	assert.Equal(t, "PATCH", handlers[0].Method)
	assert.Equal(t, "/users/123", handlers[0].Path)
}

func TestTypedRouter_DELETE(t *testing.T) {
	router := typedhttp.NewRouter()
	handler := &TestHandler{}

	typedhttp.DELETE(router, "/users/123", handler)

	handlers := router.GetHandlers()
	assert.Len(t, handlers, 1)
	assert.Equal(t, "DELETE", handlers[0].Method)
	assert.Equal(t, "/users/123", handlers[0].Path)
}

func TestTypedRouter_HEAD(t *testing.T) {
	router := typedhttp.NewRouter()
	handler := &TestHandler{}

	typedhttp.HEAD(router, "/users", handler)

	handlers := router.GetHandlers()
	assert.Len(t, handlers, 1)
	assert.Equal(t, "HEAD", handlers[0].Method)
	assert.Equal(t, "/users", handlers[0].Path)
}

func TestTypedRouter_OPTIONS(t *testing.T) {
	router := typedhttp.NewRouter()
	handler := &TestHandler{}

	typedhttp.OPTIONS(router, "/users", handler)

	handlers := router.GetHandlers()
	assert.Len(t, handlers, 1)
	assert.Equal(t, "OPTIONS", handlers[0].Method)
	assert.Equal(t, "/users", handlers[0].Path)
}

func TestTypedRouter_WithOptions(t *testing.T) {
	router := typedhttp.NewRouter()
	handler := &TestHandler{}

	typedhttp.GET(router, "/users", handler,
		typedhttp.WithTags("users"),
		typedhttp.WithSummary("Get all users"),
		typedhttp.WithDefaultObservability(),
	)

	handlers := router.GetHandlers()
	assert.Len(t, handlers, 1)
	assert.Equal(t, []string{"users"}, handlers[0].Metadata.Tags)
	assert.Equal(t, "Get all users", handlers[0].Metadata.Summary)
}

func TestTypedRouter_MultipleHandlers(t *testing.T) {
	router := typedhttp.NewRouter()
	handler := &TestHandler{}

	typedhttp.GET(router, "/users", handler)
	typedhttp.POST(router, "/users", handler)
	typedhttp.GET(router, "/users/{id}", handler)

	handlers := router.GetHandlers()
	assert.Len(t, handlers, 3)

	methods := make([]string, len(handlers))
	paths := make([]string, len(handlers))
	for i, h := range handlers {
		methods[i] = h.Method
		paths[i] = h.Path
	}

	assert.Contains(t, methods, "GET")
	assert.Contains(t, methods, "POST")
	assert.Contains(t, paths, "/users")
	assert.Contains(t, paths, "/users/{id}")
}
