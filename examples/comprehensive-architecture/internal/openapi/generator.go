package openapi

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/pavelpascari/typedhttp/examples/comprehensive-architecture/internal/config"
	"github.com/pavelpascari/typedhttp/pkg/openapi"
	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
)

// Generator wraps the OpenAPI generator with application-specific configuration
type Generator struct {
	generator *openapi.Generator
	config    *config.AppConfig
}

// NewGenerator creates a new OpenAPI generator
func NewGenerator(cfg *config.AppConfig) *Generator {
	return &Generator{
		generator: openapi.NewGenerator(cfg.ToOpenAPIConfig()),
		config:    cfg,
	}
}

// Generate generates the OpenAPI specification for the given router
func (g *Generator) Generate(router *typedhttp.TypedRouter) (*openapi3.T, error) {
	return g.generator.Generate(router)
}

// GenerateJSON generates the OpenAPI specification as JSON
func (g *Generator) GenerateJSON(spec *openapi3.T) ([]byte, error) {
	return g.generator.GenerateJSON(spec)
}

// GenerateYAML generates the OpenAPI specification as YAML
func (g *Generator) GenerateYAML(spec *openapi3.T) ([]byte, error) {
	return g.generator.GenerateYAML(spec)
}