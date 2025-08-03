package implementations

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-chi/chi/v5"
	"github.com/labstack/echo/v4"
	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
)

// Common types used across all implementations
type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type GetUserRequest struct {
	ID string `path:"id" validate:"required"`
}

type CreateUserRequest struct {
	Name  string `json:"name" validate:"required,min=2,max=50"`
	Email string `json:"email" validate:"required,email"`
}

// Mock data store for consistent testing
var testUser = User{
	ID:    "123",
	Name:  "John Doe",
	Email: "john@example.com",
}

// TypedHTTP Implementation
type TypedHTTPGetUserHandler struct{}

func (h *TypedHTTPGetUserHandler) Handle(ctx context.Context, req GetUserRequest) (User, error) {
	return testUser, nil
}

type TypedHTTPCreateUserHandler struct{}

func (h *TypedHTTPCreateUserHandler) Handle(ctx context.Context, req CreateUserRequest) (User, error) {
	return User{
		ID:    "456",
		Name:  req.Name,
		Email: req.Email,
	}, nil
}

func setupTypedHTTP() *typedhttp.TypedRouter {
	router := typedhttp.NewRouter()

	getUserHandler := &TypedHTTPGetUserHandler{}
	createUserHandler := &TypedHTTPCreateUserHandler{}

	typedhttp.GET(router, "/users/{id}", getUserHandler)
	typedhttp.POST(router, "/users", createUserHandler)

	return router
}

// Gin Implementation
func ginGetUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, gin.H{"error": "missing id"})
		return
	}
	c.JSON(200, testUser)
}

func ginCreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Manual validation (simplified)
	if req.Name == "" || len(req.Name) < 2 {
		c.JSON(400, gin.H{"error": "name validation failed"})
		return
	}

	user := User{
		ID:    "456",
		Name:  req.Name,
		Email: req.Email,
	}
	c.JSON(200, user)
}

func setupGin() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.GET("/users/:id", ginGetUser)
	r.POST("/users", ginCreateUser)
	return r
}

// Echo Implementation
func echoGetUser(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(400, map[string]string{"error": "missing id"})
	}
	return c.JSON(200, testUser)
}

func echoCreateUser(c echo.Context) error {
	var req CreateUserRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	// Manual validation (simplified)
	if req.Name == "" || len(req.Name) < 2 {
		return c.JSON(400, map[string]string{"error": "name validation failed"})
	}

	user := User{
		ID:    "456",
		Name:  req.Name,
		Email: req.Email,
	}
	return c.JSON(200, user)
}

func setupEcho() *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.GET("/users/:id", echoGetUser)
	e.POST("/users", echoCreateUser)
	return e
}

// Chi Implementation
func chiGetUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing id", 400)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(testUser)
}

func chiCreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Manual validation (simplified)
	if req.Name == "" || len(req.Name) < 2 {
		http.Error(w, "name validation failed", 400)
		return
	}

	user := User{
		ID:    "456",
		Name:  req.Name,
		Email: req.Email,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func setupChi() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/users/{id}", chiGetUser)
	r.Post("/users", chiCreateUser)
	return r
}

// Benchmark Tests

func BenchmarkTypedHTTP_GetUser(b *testing.B) {
	router := setupTypedHTTP()

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

func BenchmarkGin_GetUser(b *testing.B) {
	router := setupGin()

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

func BenchmarkEcho_GetUser(b *testing.B) {
	router := setupEcho()

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

func BenchmarkChi_GetUser(b *testing.B) {
	router := setupChi()

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

// JSON POST Benchmarks
func BenchmarkTypedHTTP_JSONPost(b *testing.B) {
	router := setupTypedHTTP()

	jsonData := []byte(`{"name":"Jane Doe","email":"jane@example.com"}`)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/users", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 201 && w.Code != 200 {
			b.Fatalf("Expected 200/201, got %d", w.Code)
		}
	}
}

func BenchmarkGin_JSONPost(b *testing.B) {
	router := setupGin()

	jsonData := []byte(`{"name":"Jane Doe","email":"jane@example.com"}`)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/users", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			b.Fatalf("Expected 200, got %d", w.Code)
		}
	}
}

func BenchmarkEcho_JSONPost(b *testing.B) {
	router := setupEcho()

	jsonData := []byte(`{"name":"Jane Doe","email":"jane@example.com"}`)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/users", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			b.Fatalf("Expected 200, got %d", w.Code)
		}
	}
}

func BenchmarkChi_JSONPost(b *testing.B) {
	router := setupChi()

	jsonData := []byte(`{"name":"Jane Doe","email":"jane@example.com"}`)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/users", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			b.Fatalf("Expected 200, got %d", w.Code)
		}
	}
}

// Memory usage comparison - Direct handler testing for TypedHTTP
func BenchmarkTypedHTTP_DirectHandler(b *testing.B) {
	handler := &TypedHTTPGetUserHandler{}
	req := GetUserRequest{ID: "123"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := handler.Handle(context.Background(), req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Complex CRUD simulation
func BenchmarkTypedHTTP_CRUD(b *testing.B) {
	router := setupTypedHTTP()

	getUserReq := httptest.NewRequest("GET", "/users/123", nil)
	createUserReq := func() *http.Request {
		jsonData := []byte(`{"name":"Jane Doe","email":"jane@example.com"}`)
		req := httptest.NewRequest("POST", "/users", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		return req
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// GET request
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, getUserReq)

		// POST request
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, createUserReq())

		if w1.Code != 200 || (w2.Code != 200 && w2.Code != 201) {
			b.Fatalf("Unexpected status codes: GET=%d, POST=%d", w1.Code, w2.Code)
		}
	}
}

func BenchmarkGin_CRUD(b *testing.B) {
	router := setupGin()

	getUserReq := httptest.NewRequest("GET", "/users/123", nil)
	createUserReq := func() *http.Request {
		jsonData := []byte(`{"name":"Jane Doe","email":"jane@example.com"}`)
		req := httptest.NewRequest("POST", "/users", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		return req
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// GET request
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, getUserReq)

		// POST request
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, createUserReq())

		if w1.Code != 200 || w2.Code != 200 {
			b.Fatalf("Unexpected status codes: GET=%d, POST=%d", w1.Code, w2.Code)
		}
	}
}
