package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/pavelpascari/typedhttp/examples/microservice-patterns/internal/config"
	"github.com/pavelpascari/typedhttp/examples/microservice-patterns/internal/models"
	"github.com/pavelpascari/typedhttp/examples/microservice-patterns/internal/openapi"
	"github.com/pavelpascari/typedhttp/examples/microservice-patterns/internal/router"
)

func main() {
	fmt.Println("🏗️ Microservice Architecture Patterns Demo")
	fmt.Println("==========================================")

	// Get service configurations
	services := config.GetServiceConfigurations()

	// For demo, we'll just show the public API
	// In real microservices, each would be a separate deployment
	publicAPIConfig := services["public-api"]

	// Setup router for public API
	appRouter := router.Setup(models.PublicAPI)

	// Generate OpenAPI spec
	generator := openapi.NewGenerator(publicAPIConfig)
	spec, err := generator.Generate(appRouter)
	if err != nil {
		log.Fatalf("Failed to generate OpenAPI spec: %v", err)
	}

	jsonSpec, err := generator.GenerateJSON(spec)
	if err != nil {
		log.Fatalf("Failed to generate JSON spec: %v", err)
	}

	// Create combined handler that serves both API routes and documentation
	combinedHandler := createCombinedHandler(appRouter, jsonSpec, services)

	// Start server
	fmt.Printf("📍 Public API Gateway running on: http://localhost:%s\n", publicAPIConfig.Port)
	fmt.Printf("📄 OpenAPI Spec: http://localhost:%s/openapi.json\n", publicAPIConfig.Port)
	fmt.Println("\n🎯 This demonstrates how different service types can use different middleware strategies")

	serverAddr := fmt.Sprintf(":%s", publicAPIConfig.Port)
	log.Fatal(http.ListenAndServe(serverAddr, combinedHandler))
}

// createCombinedHandler creates a handler that serves both API routes and documentation
func createCombinedHandler(apiRouter http.Handler, jsonSpec []byte, services map[string]config.ServiceConfig) http.Handler {
	// Create a new mux for documentation routes
	docMux := http.NewServeMux()

	// Register documentation routes
	docMux.HandleFunc("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonSpec)
	})

	// Architecture documentation
	docMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		html := `<!DOCTYPE html>
<html>
<head>
    <title>Microservice Architecture Patterns</title>
    <meta charset="UTF-8">
    <style>
        body { font-family: 'Segoe UI', system-ui, sans-serif; margin: 0; padding: 40px; line-height: 1.6; background: #f8f9fa; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 40px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .service { background: #f8f9fa; padding: 20px; margin: 20px 0; border-radius: 8px; border-left: 4px solid #007bff; }
        .pattern { background: #e7f3ff; padding: 15px; margin: 15px 0; border-radius: 6px; }
        h1 { color: #2c3e50; border-bottom: 3px solid #3498db; padding-bottom: 10px; }
        h2 { color: #34495e; margin-top: 30px; }
        .nav { background: #343a40; padding: 15px; margin: -40px -40px 30px -40px; border-radius: 8px 8px 0 0; }
        .nav a { color: #fff; text-decoration: none; margin-right: 20px; padding: 8px 16px; border-radius: 4px; transition: background 0.2s; }
        .nav a:hover { background: #495057; }
    </style>
</head>
<body>
    <div class="container">
        <div class="nav">
            <a href="/">🏠 Home</a>
            <a href="/openapi.json">📄 OpenAPI JSON</a>
        </div>

        <h1>🏗️ Microservice Architecture Patterns</h1>
        
        <p>This example demonstrates <strong>different middleware strategies</strong> for different types of services in a microservice architecture.</p>

        <h2>📁 Package Structure</h2>
        
        <div class="service">
            <h3>internal/models/</h3>
            <p><strong>Purpose:</strong> Domain models and request/response types for all service types</p>
            <p><strong>Files:</strong> shared.go, public_api.go, internal_service.go, admin_api.go, health.go</p>
        </div>

        <div class="service">
            <h3>internal/handlers/</h3>
            <p><strong>Purpose:</strong> Business logic handlers for different service types</p>
            <p><strong>Files:</strong> public_api.go, internal_service.go, admin_api.go, health.go</p>
        </div>

        <div class="service">
            <h3>internal/middleware/</h3>
            <p><strong>Purpose:</strong> Service-specific middleware implementations and stacks</p>
            <p><strong>Files:</strong> security.go, rate_limit.go, tracking.go, audit.go, envelope.go, stack.go</p>
        </div>

        <div class="service">
            <h3>internal/router/</h3>
            <p><strong>Purpose:</strong> Service-aware route registration and middleware application</p>
            <p><strong>Files:</strong> router.go</p>
        </div>

        <div class="service">
            <h3>internal/config/</h3>
            <p><strong>Purpose:</strong> Service configuration management</p>
            <p><strong>Files:</strong> config.go</p>
        </div>

        <h2>🎯 Service Types & Middleware Strategies</h2>`

		for name, service := range services {
			html += fmt.Sprintf(`
        <div class="service">
            <h3>%s Service (Port %s)</h3>
            <p><strong>Purpose:</strong> %s</p>
            <p><strong>Architecture:</strong> %s</p>
        </div>`,
				name,
				service.Port,
				service.Description,
				getArchitectureDescription(service.Type))
		}

		html += `
        <h2>🔧 Middleware Patterns</h2>
        
        <div class="pattern">
            <h3>Public API Gateway Pattern</h3>
            <p>Full security stack with rate limiting, authentication, response enveloping, and comprehensive logging.</p>
        </div>
        
        <div class="pattern">
            <h3>Internal Service Pattern</h3>
            <p>Minimal middleware for maximum performance - only essential request tracking and simple response formatting.</p>
        </div>
        
        <div class="pattern">
            <h3>Admin API Pattern</h3>
            <p>Enhanced security with admin authentication, comprehensive audit logging, and detailed response metadata.</p>
        </div>
        
        <div class="pattern">
            <h3>Health Check Pattern</h3>
            <p>Ultra-minimal middleware - only basic request tracking to avoid impacting health check performance.</p>
        </div>

        <h2>📊 Architecture Benefits</h2>
        <ul>
            <li><strong>Separation of Concerns:</strong> Different middleware for different service responsibilities</li>
            <li><strong>Performance Optimization:</strong> Minimal overhead where needed</li>
            <li><strong>Security Layering:</strong> Enhanced security for sensitive endpoints</li>
            <li><strong>Audit Compliance:</strong> Comprehensive logging for admin operations</li>
            <li><strong>Accurate Documentation:</strong> OpenAPI specs reflect actual middleware transformations</li>
            <li><strong>Package Organization:</strong> Clean separation into conceptual elements</li>
        </ul>

        <p><em>This example demonstrates production-ready microservice architecture patterns with TypedHTTP.</em></p>
    </div>
</body>
</html>`

		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.Write([]byte(html))
	})

	// Return a handler that routes between documentation and API
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Route documentation paths to the documentation mux
		if r.URL.Path == "/" || r.URL.Path == "/openapi.json" {
			docMux.ServeHTTP(w, r)
			return
		}

		// Route all other paths to the API router
		apiRouter.ServeHTTP(w, r)
	})
}

func getArchitectureDescription(serviceType models.ServiceType) string {
	switch serviceType {
	case models.PublicAPI:
		return "Security → Rate Limiting → Tracking → Response Envelope"
	case models.InternalService:
		return "Tracking → Simple Response Format"
	case models.AdminAPI:
		return "Security → Admin Auth → Audit → Tracking → Admin Envelope"
	case models.HealthCheckService:
		return "Minimal Tracking Only"
	default:
		return "Custom Architecture"
	}
}
