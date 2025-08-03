package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/pavelpascari/typedhttp/examples/comprehensive-architecture/internal/config"
	"github.com/pavelpascari/typedhttp/examples/comprehensive-architecture/internal/openapi"
	"github.com/pavelpascari/typedhttp/examples/comprehensive-architecture/internal/router"
)

func main() {
	// Load configuration
	cfg := config.NewDefaultConfig()

	// Setup router with all routes and middleware
	appRouter := router.Setup()

	// Setup OpenAPI generation
	apiGenerator := openapi.NewGenerator(cfg)
	spec, err := apiGenerator.Generate(appRouter)
	if err != nil {
		log.Fatalf("Failed to generate OpenAPI spec: %v", err)
	}

	// Generate JSON and YAML specifications
	jsonSpec, err := apiGenerator.GenerateJSON(spec)
	if err != nil {
		log.Fatalf("Failed to generate JSON spec: %v", err)
	}

	yamlSpec, err := apiGenerator.GenerateYAML(spec)
	if err != nil {
		log.Fatalf("Failed to generate YAML spec: %v", err)
	}

	// Create combined handler that serves both API routes and documentation
	combinedHandler := createCombinedHandler(appRouter, jsonSpec, yamlSpec)

	// Start server
	fmt.Println("🚀 Comprehensive TypedHTTP Architecture Example")
	fmt.Printf("📍 Server starting on http://%s:%s\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Println()
	fmt.Printf("🏠 Main Page: http://%s:%s\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("📚 API Docs:  http://%s:%s/docs\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("📄 OpenAPI:   http://%s:%s/openapi.json\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Println()
	fmt.Println("🔧 Features demonstrated:")
	fmt.Println("   ✓ Proper package organization")
	fmt.Println("   ✓ Layered middleware architecture")
	fmt.Println("   ✓ Response schema modification")
	fmt.Println("   ✓ Automatic OpenAPI generation")
	fmt.Println("   ✓ Configuration management")
	fmt.Println("   ✓ Production-ready patterns")

	serverAddr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	log.Fatal(http.ListenAndServe(serverAddr, combinedHandler))
}

// createCombinedHandler creates a handler that serves both API routes and documentation
func createCombinedHandler(apiRouter http.Handler, jsonSpec, yamlSpec []byte) http.Handler {
	// Create a new mux for documentation routes
	docMux := http.NewServeMux()

	// Register documentation routes
	docMux.HandleFunc("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonSpec)
	})

	docMux.HandleFunc("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-yaml")
		w.Write(yamlSpec)
	})

	// Documentation UI
	docMux.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		html := `<!DOCTYPE html>
<html>
<head>
    <title>E-commerce API Documentation</title>
    <meta charset="UTF-8">
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@3.52.5/swagger-ui.css" />
    <style>
        html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
        *, *:before, *:after { box-sizing: inherit; }
        body { margin:0; background: #fafafa; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@3.52.5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@3.52.5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            SwaggerUIBundle({
                url: '/openapi.json',
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout"
            });
        };
    </script>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.Write([]byte(html))
	})

	// Architecture guide
	docMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		html := `<!DOCTYPE html>
<html>
<head>
    <title>Comprehensive TypedHTTP Architecture Example</title>
    <meta charset="UTF-8">
    <style>
        body { 
            font-family: 'Segoe UI', system-ui, sans-serif; 
            margin: 0; 
            padding: 40px; 
            line-height: 1.6; 
            background: #f8f9fa;
        }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 40px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .package { background: #f8f9fa; padding: 15px; margin: 10px 0; border-radius: 6px; border-left: 4px solid #007bff; }
        .code { background: #f1f3f4; padding: 20px; border-radius: 6px; font-family: monospace; font-size: 14px; overflow-x: auto; }
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
            <a href="/docs">📚 API Docs</a>
            <a href="/openapi.json">📄 OpenAPI JSON</a>
            <a href="/openapi.yaml">📄 OpenAPI YAML</a>
        </div>

        <h1>🏗️ Comprehensive TypedHTTP Architecture Example</h1>
        
        <p>This example demonstrates <strong>proper Go project structure</strong> and <strong>production-ready patterns</strong> for building APIs with TypedHTTP.</p>

        <h2>📁 Package Structure</h2>
        
        <div class="package">
            <h3>internal/models/</h3>
            <p><strong>Purpose:</strong> Domain models and request/response types</p>
            <p><strong>Files:</strong> user.go, product.go, order.go</p>
        </div>

        <div class="package">
            <h3>internal/handlers/</h3>
            <p><strong>Purpose:</strong> Business logic handlers</p>
            <p><strong>Files:</strong> user.go, product.go, order.go</p>
        </div>

        <div class="package">
            <h3>internal/middleware/</h3>
            <p><strong>Purpose:</strong> Middleware implementations and stack configuration</p>
            <p><strong>Files:</strong> request_id.go, audit.go, cache.go, stack.go</p>
        </div>

        <div class="package">
            <h3>internal/router/</h3>
            <p><strong>Purpose:</strong> Route registration and middleware application</p>
            <p><strong>Files:</strong> router.go</p>
        </div>

        <div class="package">
            <h3>internal/config/</h3>
            <p><strong>Purpose:</strong> Application configuration management</p>
            <p><strong>Files:</strong> config.go</p>
        </div>

        <div class="package">
            <h3>internal/openapi/</h3>
            <p><strong>Purpose:</strong> OpenAPI specification generation</p>
            <p><strong>Files:</strong> generator.go</p>
        </div>

        <h2>🎯 Architecture Principles</h2>
        
        <ul>
            <li><strong>Separation of Concerns:</strong> Each package has a single responsibility</li>
            <li><strong>Dependency Direction:</strong> Dependencies flow inward (Clean Architecture)</li>
            <li><strong>Configuration Management:</strong> Centralized configuration with defaults</li>
            <li><strong>Interface-Based Design:</strong> Handlers implement TypedHTTP interfaces</li>
            <li><strong>Testability:</strong> Each package can be tested independently</li>
        </ul>

        <h2>🔧 Middleware Architecture</h2>
        
        <div class="code">Priority 100: Request ID Middleware
Priority  90: Response Envelope Middleware  
Priority  50: Cache Metadata Middleware
Priority  10: Audit Logging Middleware</div>

        <h2>🚀 Getting Started</h2>
        
        <ol>
            <li>Visit <a href="/docs">/docs</a> to explore the interactive API documentation</li>
            <li>Check the <a href="/openapi.json">OpenAPI specification</a> to see middleware schema transformations</li>
            <li>Review the source code to understand the package organization</li>
            <li>Adapt the structure for your specific needs</li>
        </ol>

        <p><em>This example demonstrates production-ready Go project structure and TypedHTTP best practices.</em></p>
    </div>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.Write([]byte(html))
	})

	// Return a handler that routes between documentation and API
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Route documentation paths to the documentation mux
		if r.URL.Path == "/" || r.URL.Path == "/docs" || r.URL.Path == "/openapi.json" || r.URL.Path == "/openapi.yaml" {
			docMux.ServeHTTP(w, r)
			return
		}

		// Route all other paths to the API router
		apiRouter.ServeHTTP(w, r)
	})
}
