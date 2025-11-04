//go:build ignore
// +build ignore

package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// User represents a user in the system
type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// CreateUserRequest represents the request to create a user
type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// In-memory storage for demo
var users = map[string]User{
	"1": {ID: "1", Name: "Alice Smith", Email: "alice@example.com"},
	"2": {ID: "2", Name: "Bob Johnson", Email: "bob@example.com"},
}
var nextID = 3

// GetUser handles GET /users/:id
func GetUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, gin.H{"error": "missing id"})
		return
	}

	user, exists := users[id]
	if !exists {
		c.JSON(404, gin.H{"error": "user not found"})
		return
	}

	c.JSON(200, user)
}

// ListUsers handles GET /users
func ListUsers(c *gin.Context) {
	limit := 10 // default
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	result := []User{}
	count := 0
	for _, user := range users {
		if count >= limit {
			break
		}
		result = append(result, user)
		count++
	}

	c.JSON(200, gin.H{
		"users": result,
		"total": len(users),
		"limit": limit,
	})
}

// CreateUser handles POST /users
func CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Manual validation
	if req.Name == "" || len(req.Name) < 2 {
		c.JSON(400, gin.H{"error": "name must be at least 2 characters"})
		return
	}
	if req.Email == "" {
		c.JSON(400, gin.H{"error": "email is required"})
		return
	}

	// Check for duplicate email
	for _, user := range users {
		if user.Email == req.Email {
			c.JSON(409, gin.H{"error": "user with this email already exists"})
			return
		}
	}

	// Create user
	id := strconv.Itoa(nextID)
	nextID++
	user := User{
		ID:    id,
		Name:  req.Name,
		Email: req.Email,
	}
	users[id] = user

	c.JSON(201, user)
}

// AuthMiddleware validates authentication
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(401, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		// Simple token validation (in real app, validate JWT etc.)
		if token != "Bearer valid-token" {
			c.JSON(401, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		// Set user context (simplified)
		c.Set("user_id", "current-user")
		c.Next()
	}
}

func main() {
	r := gin.Default()

	// Middleware
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// Public routes
	r.GET("/users", ListUsers)
	r.GET("/users/:id", GetUser)

	// Protected routes
	protected := r.Group("/")
	protected.Use(AuthMiddleware())
	{
		protected.POST("/users", CreateUser)
	}

	r.Run(":8080")
}

// Run with: go run gin-example.go
// Test with:
//   curl http://localhost:8080/users
//   curl http://localhost:8080/users/1
//   curl -H "Authorization: Bearer valid-token" -X POST http://localhost:8080/users -d '{"name":"Jane","email":"jane@example.com"}' -H "Content-Type: application/json"
