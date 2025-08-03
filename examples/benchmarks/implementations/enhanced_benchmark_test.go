package implementations

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
)

// Enhanced benchmark types for different complexity levels

// Simple types (Level 1: Simple operations)
type SimpleUser struct {
	ID   string `json:"id" path:"id" validate:"required"`
	Name string `json:"name" validate:"required,min=2,max=50"`
}

// Medium complexity types (Level 2: Real-world operations)  
type UserProfile struct {
	ID       string   `json:"id" path:"id" validate:"required"`
	Name     string   `json:"name" validate:"required,min=2,max=50"`
	Email    string   `json:"email" validate:"required,email"`
	Age      int      `json:"age" validate:"min=18,max=120"`
	Tags     []string `json:"tags" validate:"dive,required,min=1"`
	Settings struct {
		Theme       string `json:"theme" validate:"oneof=light dark"`
		Language    string `json:"language" validate:"required,min=2,max=5"`
		Preferences map[string]interface{} `json:"preferences"`
	} `json:"settings"`
}

// Complex types (Level 3: Enterprise operations)
type EnterpriseResource struct {
	ID          string                 `json:"id" path:"id" validate:"required"`
	Name        string                 `json:"name" validate:"required,min=2,max=100"`
	Description string                 `json:"description" validate:"max=1000"`
	Metadata    map[string]interface{} `json:"metadata"`
	Permissions []Permission           `json:"permissions" validate:"dive"`
	CreatedBy   UserProfile            `json:"created_by"`
	UpdatedBy   UserProfile            `json:"updated_by"`
	Teams       []Team                 `json:"teams" validate:"dive"`
	Audit       AuditLog               `json:"audit"`
}

type Permission struct {
	Resource string   `json:"resource" validate:"required"`
	Actions  []string `json:"actions" validate:"dive,oneof=read write delete admin"`
	Scope    string   `json:"scope" validate:"oneof=global team user"`
}

type Team struct {
	ID      string        `json:"id" validate:"required"`
	Name    string        `json:"name" validate:"required,min=2,max=50"`
	Members []UserProfile `json:"members" validate:"dive"`
}

type AuditLog struct {
	CreatedAt string            `json:"created_at" validate:"required"`
	UpdatedAt string            `json:"updated_at" validate:"required"`
	Changes   map[string]string `json:"changes"`
}

// Payload generators for different sizes
func generateSmallPayload() []byte {
	user := SimpleUser{
		ID:   "user123",
		Name: "John Doe",
	}
	data, _ := json.Marshal(user)
	return data
}

func generateMediumPayload() []byte {
	profile := UserProfile{
		ID:    "user456",
		Name:  "Jane Smith",
		Email: "jane@example.com",
		Age:   30,
		Tags:  []string{"developer", "frontend", "react"},
		Settings: struct {
			Theme       string                 `json:"theme" validate:"oneof=light dark"`
			Language    string                 `json:"language" validate:"required,min=2,max=5"`
			Preferences map[string]interface{} `json:"preferences"`
		}{
			Theme:    "dark",
			Language: "en",
			Preferences: map[string]interface{}{
				"notifications": true,
				"sidebar":       "collapsed",
				"theme_color":   "#2563eb",
			},
		},
	}
	data, _ := json.Marshal(profile)
	return data
}

func generateLargePayload() []byte {
	resource := EnterpriseResource{
		ID:          "res789",
		Name:        "Production Database Cluster",
		Description: "Main production database cluster for the e-commerce platform with high availability, auto-scaling, and comprehensive monitoring. This resource handles customer data, orders, inventory, and analytics workloads.",
		Metadata: map[string]interface{}{
			"region":      "us-east-1",
			"environment": "production",
			"cost_center": "engineering",
			"owner":       "platform-team",
			"tags": map[string]string{
				"Application": "ecommerce",
				"Service":     "database",
				"Criticality": "high",
			},
		},
		Permissions: []Permission{
			{Resource: "database", Actions: []string{"read", "write"}, Scope: "team"},
			{Resource: "monitoring", Actions: []string{"read"}, Scope: "global"},
			{Resource: "backups", Actions: []string{"read", "write", "admin"}, Scope: "team"},
		},
		CreatedBy: UserProfile{
			ID:    "admin123",
			Name:  "Admin User",
			Email: "admin@company.com",
			Age:   35,
			Tags:  []string{"admin", "platform", "devops"},
		},
		UpdatedBy: UserProfile{
			ID:    "ops456",
			Name:  "Operations Team",
			Email: "ops@company.com",
			Age:   32,
			Tags:  []string{"operations", "monitoring", "sre"},
		},
		Teams: []Team{
			{
				ID:   "team1",
				Name: "Platform Engineering",
				Members: []UserProfile{
					{ID: "eng1", Name: "Engineer One", Email: "eng1@company.com", Age: 28, Tags: []string{"backend", "go", "kubernetes"}},
					{ID: "eng2", Name: "Engineer Two", Email: "eng2@company.com", Age: 31, Tags: []string{"frontend", "react", "typescript"}},
					{ID: "eng3", Name: "Engineer Three", Email: "eng3@company.com", Age: 29, Tags: []string{"devops", "aws", "terraform"}},
				},
			},
		},
		Audit: AuditLog{
			CreatedAt: "2023-01-15T10:30:00Z",
			UpdatedAt: "2023-06-20T14:45:00Z",
			Changes: map[string]string{
				"description": "Updated cluster description with more details",
				"permissions": "Added backup management permissions",
				"teams":       "Added new team member",
			},
		},
	}
	data, _ := json.Marshal(resource)
	return data
}

func generateXLargePayload() []byte {
	// Generate a very large payload by duplicating teams
	resource := EnterpriseResource{
		ID:          "res999",
		Name:        "Enterprise Resource Management System",
		Description: strings.Repeat("This is a comprehensive enterprise resource management system that handles multiple aspects of business operations including customer relationship management, inventory tracking, financial reporting, human resources, and supply chain management. ", 10),
		Metadata:    make(map[string]interface{}),
		Teams:       make([]Team, 0, 20),
	}

	// Add large metadata
	for i := 0; i < 50; i++ {
		resource.Metadata[generateKey(i)] = generateValue(i)
	}

	// Add many teams with many members
	for teamIdx := 0; teamIdx < 20; teamIdx++ {
		team := Team{
			ID:      generateTeamID(teamIdx),
			Name:    generateTeamName(teamIdx),
			Members: make([]UserProfile, 0, 10),
		}
		
		for memberIdx := 0; memberIdx < 10; memberIdx++ {
			member := UserProfile{
				ID:    generateMemberID(teamIdx, memberIdx),
				Name:  generateMemberName(teamIdx, memberIdx),
				Email: generateMemberEmail(teamIdx, memberIdx),
				Age:   25 + (memberIdx * 3),
				Tags:  []string{"team" + generateTeamID(teamIdx), "member", "enterprise"},
			}
			team.Members = append(team.Members, member)
		}
		resource.Teams = append(resource.Teams, team)
	}

	data, _ := json.Marshal(resource)
	return data
}

// Helper functions for generating large payloads
func generateKey(i int) string {
	return "key_" + strings.Repeat("abcdefghijklmnop", i%5+1) + "_" + string(rune(i))
}

func generateValue(i int) interface{} {
	switch i % 4 {
	case 0:
		return "value_" + strings.Repeat("qrstuvwxyz", i%3+1)
	case 1:
		return i * 100
	case 2:
		return i%2 == 0
	default:
		return map[string]interface{}{
			"nested": "data",
			"index":  i,
		}
	}
}

func generateTeamID(i int) string {
	return "team_" + string(rune(65+i%26)) + string(rune(48+i%10))
}

func generateTeamName(i int) string {
	names := []string{"Engineering", "Operations", "Sales", "Marketing", "Support", "Product", "Design", "Security", "Quality", "Research"}
	return names[i%len(names)] + " Team " + string(rune(65+i%26))
}

func generateMemberID(teamIdx, memberIdx int) string {
	return "member_" + string(rune(65+teamIdx%26)) + string(rune(48+memberIdx%10))
}

func generateMemberName(teamIdx, memberIdx int) string {
	firstNames := []string{"Alice", "Bob", "Charlie", "Diana", "Edward", "Fiona", "George", "Helen", "Ivan", "Julia"}
	lastNames := []string{"Anderson", "Brown", "Clark", "Davis", "Evans", "Foster", "Garcia", "Harris", "Johnson", "King"}
	return firstNames[memberIdx%len(firstNames)] + " " + lastNames[teamIdx%len(lastNames)]
}

func generateMemberEmail(teamIdx, memberIdx int) string {
	return generateMemberName(teamIdx, memberIdx) + "@company.com"
}

// TypedHTTP handlers for enhanced benchmarks
type SimpleGetHandler struct{}
func (h *SimpleGetHandler) Handle(ctx context.Context, req struct{ ID string `path:"id"` }) (SimpleUser, error) {
	return SimpleUser{ID: req.ID, Name: "Test User"}, nil
}

type MediumPostHandler struct{}
func (h *MediumPostHandler) Handle(ctx context.Context, req UserProfile) (UserProfile, error) {
	req.ID = "generated_id"
	return req, nil
}

type ComplexPostHandler struct{}
func (h *ComplexPostHandler) Handle(ctx context.Context, req EnterpriseResource) (EnterpriseResource, error) {
	req.ID = "generated_complex_id"
	return req, nil
}

// Setup functions for TypedHTTP routers
func setupTypedHTTPSimple() *typedhttp.TypedRouter {
	router := typedhttp.NewRouter()
	handler := &SimpleGetHandler{}
	typedhttp.GET(router, "/users/{id}", handler)
	return router
}

func setupTypedHTTPMedium() *typedhttp.TypedRouter {
	router := typedhttp.NewRouter()
	handler := &MediumPostHandler{}
	typedhttp.POST(router, "/users", handler)
	return router
}

func setupTypedHTTPComplex() *typedhttp.TypedRouter {
	router := typedhttp.NewRouter()
	handler := &ComplexPostHandler{}
	typedhttp.POST(router, "/resources", handler)
	return router
}

// Gin handlers for comparison
func ginSimpleGet(c *gin.Context) {
	id := c.Param("id")
	user := SimpleUser{ID: id, Name: "Test User"}
	c.JSON(200, user)
}

func ginMediumPost(c *gin.Context) {
	var profile UserProfile
	if err := c.ShouldBindJSON(&profile); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	profile.ID = "generated_id"
	c.JSON(201, profile)
}

func ginComplexPost(c *gin.Context) {
	var resource EnterpriseResource
	if err := c.ShouldBindJSON(&resource); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	resource.ID = "generated_complex_id"
	c.JSON(201, resource)
}

func setupGinSimple() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.GET("/users/:id", ginSimpleGet)
	return r
}

func setupGinMedium() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.POST("/users", ginMediumPost)
	return r
}

func setupGinComplex() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.POST("/resources", ginComplexPost)
	return r
}

// Enhanced Benchmarks

// Level 1: Simple GET operations
func BenchmarkTypedHTTP_Simple_GET(b *testing.B) {
	router := setupTypedHTTPSimple()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/users/123", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			b.Fatalf("Expected 200, got %d", w.Code)
		}
	}
}

func BenchmarkGin_Simple_GET(b *testing.B) {
	router := setupGinSimple()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/users/123", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			b.Fatalf("Expected 200, got %d", w.Code)
		}
	}
}

// Level 2: Medium complexity with validation
func BenchmarkTypedHTTP_Medium_POST_1KB(b *testing.B) {
	router := setupTypedHTTPMedium()
	payload := generateMediumPayload()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/users", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 201 && w.Code != 200 {
			b.Fatalf("Expected 200/201, got %d", w.Code)
		}
	}
}

func BenchmarkGin_Medium_POST_1KB(b *testing.B) {
	router := setupGinMedium()
	payload := generateMediumPayload()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/users", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 201 {
			b.Fatalf("Expected 201, got %d", w.Code)
		}
	}
}

// Level 3: Complex operations with large payloads
func BenchmarkTypedHTTP_Complex_POST_10KB(b *testing.B) {
	router := setupTypedHTTPComplex()
	payload := generateLargePayload()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/resources", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 201 && w.Code != 200 {
			b.Fatalf("Expected 200/201, got %d", w.Code)
		}
	}
}

func BenchmarkGin_Complex_POST_10KB(b *testing.B) {
	router := setupGinComplex()
	payload := generateLargePayload()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/resources", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 201 {
			b.Fatalf("Expected 201, got %d", w.Code)
		}
	}
}

// Level 4: Extra large payloads
func BenchmarkTypedHTTP_XLarge_POST_100KB(b *testing.B) {
	router := setupTypedHTTPComplex()
	payload := generateXLargePayload()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/resources", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 201 && w.Code != 200 {
			b.Fatalf("Expected 200/201, got %d", w.Code)
		}
	}
}

func BenchmarkGin_XLarge_POST_100KB(b *testing.B) {
	router := setupGinComplex()
	payload := generateXLargePayload()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/resources", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 201 {
			b.Fatalf("Expected 201, got %d", w.Code)
		}
	}
}

// Payload size information
func BenchmarkPayloadSizes(b *testing.B) {
	b.Skip("This is just for documentation")
	
	small := generateSmallPayload()
	medium := generateMediumPayload()
	large := generateLargePayload()
	xlarge := generateXLargePayload()
	
	b.Logf("Small payload size: %d bytes", len(small))
	b.Logf("Medium payload size: %d bytes", len(medium))
	b.Logf("Large payload size: %d bytes", len(large))
	b.Logf("XLarge payload size: %d bytes", len(xlarge))
}