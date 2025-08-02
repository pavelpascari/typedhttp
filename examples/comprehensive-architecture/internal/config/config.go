package config

import "github.com/pavelpascari/typedhttp/pkg/openapi"

// AppConfig holds the application configuration
type AppConfig struct {
	Server   ServerConfig   `yaml:"server"`
	OpenAPI  OpenAPIConfig  `yaml:"openapi"`
	Logging  LoggingConfig  `yaml:"logging"`
	Database DatabaseConfig `yaml:"database"`
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Port string `yaml:"port" default:"8080"`
	Host string `yaml:"host" default:"localhost"`
}

// OpenAPIConfig holds OpenAPI generation configuration
type OpenAPIConfig struct {
	Title       string `yaml:"title" default:"Comprehensive E-commerce API"`
	Version     string `yaml:"version" default:"1.0.0"`
	Description string `yaml:"description"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level" default:"info"`
	Format string `yaml:"format" default:"json"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string `yaml:"host" default:"localhost"`
	Port     int    `yaml:"port" default:"5432"`
	Name     string `yaml:"name" default:"ecommerce"`
	User     string `yaml:"user" default:"postgres"`
	Password string `yaml:"password"`
	SSLMode  string `yaml:"ssl_mode" default:"disable"`
}

// NewDefaultConfig returns a configuration with default values
func NewDefaultConfig() *AppConfig {
	return &AppConfig{
		Server: ServerConfig{
			Port: "8080",
			Host: "localhost",
		},
		OpenAPI: OpenAPIConfig{
			Title:       "Comprehensive E-commerce API",
			Version:     "1.0.0",
			Description: "A comprehensive example demonstrating middleware architecture and OpenAPI generation with typedhttp",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
		Database: DatabaseConfig{
			Host:    "localhost",
			Port:    5432,
			Name:    "ecommerce",
			User:    "postgres",
			SSLMode: "disable",
		},
	}
}

// ToOpenAPIConfig converts the app config to an OpenAPI config
func (c *AppConfig) ToOpenAPIConfig() *openapi.Config {
	return &openapi.Config{
		Info: openapi.Info{
			Title:       c.OpenAPI.Title,
			Version:     c.OpenAPI.Version,
			Description: c.OpenAPI.Description,
			Contact: &openapi.Contact{
				Name:  "API Team",
				Email: "api-team@example.com",
				URL:   "https://example.com/contact",
			},
			License: &openapi.License{
				Name: "MIT",
				URL:  "https://opensource.org/licenses/MIT",
			},
		},
		Servers: []openapi.Server{
			{
				URL:         "https://api.example.com/v1",
				Description: "Production server",
			},
			{
				URL:         "https://staging-api.example.com/v1",
				Description: "Staging server",
			},
			{
				URL:         "http://localhost:8080",
				Description: "Development server",
			},
		},
		Security: map[string]openapi.SecurityScheme{
			"bearerAuth": {
				Type:         "http",
				Scheme:       "bearer",
				BearerFormat: "JWT",
			},
			"apiKey": {
				Type: "apiKey",
				In:   "header",
				Name: "X-API-Key",
			},
		},
	}
}