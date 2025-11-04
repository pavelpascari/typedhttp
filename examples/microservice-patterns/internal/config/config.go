package config

import (
	"github.com/pavelpascari/typedhttp/examples/microservice-patterns/internal/models"
	"github.com/pavelpascari/typedhttp/pkg/openapi"
)

// ServiceConfig represents configuration for a microservice
type ServiceConfig struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Port        string             `json:"port"`
	Type        models.ServiceType `json:"type"`
	OpenAPI     OpenAPIConfig      `json:"openapi"`
}

// OpenAPIConfig represents OpenAPI generation configuration
type OpenAPIConfig struct {
	Title       string           `json:"title"`
	Version     string           `json:"version"`
	Description string           `json:"description"`
	Servers     []openapi.Server `json:"servers"`
}

// GetServiceConfigurations returns predefined service configurations
func GetServiceConfigurations() map[string]ServiceConfig {
	return map[string]ServiceConfig{
		"public-api": {
			Name:        "public-api",
			Description: "Public API Gateway - Full middleware stack",
			Port:        "8080",
			Type:        models.PublicAPI,
			OpenAPI: OpenAPIConfig{
				Title:       "Public API Gateway",
				Version:     "1.0.0",
				Description: "Production-ready public API with comprehensive middleware stack",
				Servers: []openapi.Server{
					{URL: "https://api.example.com/v1", Description: "Production"},
					{URL: "http://localhost:8080", Description: "Development"},
				},
			},
		},
		"internal": {
			Name:        "internal",
			Description: "Internal Service - Minimal middleware for performance",
			Port:        "8081",
			Type:        models.InternalService,
			OpenAPI: OpenAPIConfig{
				Title:       "Internal Service API",
				Version:     "1.0.0",
				Description: "High-performance internal service with minimal middleware",
				Servers: []openapi.Server{
					{URL: "http://internal.example.com", Description: "Internal Network"},
					{URL: "http://localhost:8081", Description: "Development"},
				},
			},
		},
		"admin": {
			Name:        "admin",
			Description: "Admin API - Enhanced security and audit",
			Port:        "8082",
			Type:        models.AdminAPI,
			OpenAPI: OpenAPIConfig{
				Title:       "Admin API",
				Version:     "1.0.0",
				Description: "Administrative API with enhanced security and audit logging",
				Servers: []openapi.Server{
					{URL: "https://admin.example.com", Description: "Production"},
					{URL: "http://localhost:8082", Description: "Development"},
				},
			},
		},
		"health": {
			Name:        "health",
			Description: "Health Check - Minimal overhead",
			Port:        "8083",
			Type:        models.HealthCheckService,
			OpenAPI: OpenAPIConfig{
				Title:       "Health Check API",
				Version:     "1.0.0",
				Description: "Lightweight health check endpoints",
				Servers: []openapi.Server{
					{URL: "http://localhost:8083", Description: "Development"},
				},
			},
		},
	}
}

// ToOpenAPIConfig converts ServiceConfig to openapi.Config
func (sc ServiceConfig) ToOpenAPIConfig() *openapi.Config {
	return &openapi.Config{
		Info: openapi.Info{
			Title:       sc.OpenAPI.Title,
			Version:     sc.OpenAPI.Version,
			Description: sc.OpenAPI.Description,
		},
		Servers: sc.OpenAPI.Servers,
	}
}
