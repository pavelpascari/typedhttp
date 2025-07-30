package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRouter() *typedhttp.TypedRouter {
	router := typedhttp.NewRouter()
	
	getUserHandler := &GetUserHandler{}
	createUserHandler := &CreateUserHandler{}
	
	typedhttp.GET(router, "/users/{id}", getUserHandler,
		typedhttp.WithTags("users"),
		typedhttp.WithSummary("Get user by ID"),
		typedhttp.WithDefaultObservability(),
	)
	
	typedhttp.POST(router, "/users", createUserHandler,
		typedhttp.WithTags("users"),
		typedhttp.WithSummary("Create a new user"),
		typedhttp.WithDefaultObservability(),
		typedhttp.WithErrorMapper(&typedhttp.DefaultErrorMapper{}),
	)
	
	return router
}

func TestGetUserHandler_Success(t *testing.T) {
	handler := &GetUserHandler{}
	req := GetUserRequest{ID: "123"}
	
	resp, err := handler.Handle(context.Background(), req)
	
	require.NoError(t, err)
	assert.Equal(t, "123", resp.ID)
	assert.Equal(t, "John Doe", resp.Name)
	assert.Equal(t, "john@example.com", resp.Email)
	assert.Contains(t, resp.Message, "Found user with ID: 123")
}

func TestGetUserHandler_NotFound(t *testing.T) {
	handler := &GetUserHandler{}
	req := GetUserRequest{ID: "not-found"}
	
	_, err := handler.Handle(context.Background(), req)
	
	require.Error(t, err)
	
	var notFoundErr *typedhttp.NotFoundError
	assert.ErrorAs(t, err, &notFoundErr)
	assert.Equal(t, "user", notFoundErr.Resource)
	assert.Equal(t, "not-found", notFoundErr.ID)
}

func TestCreateUserHandler_Success(t *testing.T) {
	handler := &CreateUserHandler{}
	req := CreateUserRequest{
		Name:  "Jane Doe",
		Email: "jane@example.com",
	}
	
	resp, err := handler.Handle(context.Background(), req)
	
	require.NoError(t, err)
	assert.Equal(t, "user_12345", resp.ID)
	assert.Equal(t, "Jane Doe", resp.Name)
	assert.Equal(t, "jane@example.com", resp.Email)
	assert.Equal(t, "User created successfully", resp.Message)
}

func TestCreateUserHandler_Conflict(t *testing.T) {
	handler := &CreateUserHandler{}
	req := CreateUserRequest{
		Name:  "Bob Smith",
		Email: "duplicate@example.com",
	}
	
	_, err := handler.Handle(context.Background(), req)
	
	require.Error(t, err)
	
	var conflictErr *typedhttp.ConflictError
	assert.ErrorAs(t, err, &conflictErr)
	assert.Equal(t, "User with this email already exists", conflictErr.Message)
}

func TestHTTP_GetUser_Success(t *testing.T) {
	router := setupTestRouter()
	server := httptest.NewServer(router)
	defer server.Close()
	
	// Test successful GET request
	resp, err := http.Get(server.URL + "/users/123")
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	
	var userResp GetUserResponse
	err = json.NewDecoder(resp.Body).Decode(&userResp)
	require.NoError(t, err)
	
	assert.Equal(t, "123", userResp.ID)
	assert.Equal(t, "John Doe", userResp.Name)
	assert.Equal(t, "john@example.com", userResp.Email)
	assert.Contains(t, userResp.Message, "Found user with ID: 123")
}

func TestHTTP_GetUser_NotFound(t *testing.T) {
	router := setupTestRouter()
	server := httptest.NewServer(router)
	defer server.Close()
	
	// Test not found case
	resp, err := http.Get(server.URL + "/users/not-found")
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	
	var errorResp typedhttp.ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errorResp)
	require.NoError(t, err)
	
	assert.Equal(t, "NOT_FOUND", errorResp.Code)
	assert.Contains(t, errorResp.Error, "user with id 'not-found' not found")
}

func TestHTTP_CreateUser_Success(t *testing.T) {
	router := setupTestRouter()
	server := httptest.NewServer(router)
	defer server.Close()
	
	// Prepare request body
	createReq := CreateUserRequest{
		Name:  "Jane Doe",
		Email: "jane@example.com",
	}
	
	jsonBody, err := json.Marshal(createReq)
	require.NoError(t, err)
	
	// Test successful POST request
	resp, err := http.Post(
		server.URL+"/users",
		"application/json",
		bytes.NewReader(jsonBody),
	)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	
	var createResp CreateUserResponse
	err = json.NewDecoder(resp.Body).Decode(&createResp)
	require.NoError(t, err)
	
	assert.Equal(t, "user_12345", createResp.ID)
	assert.Equal(t, "Jane Doe", createResp.Name)
	assert.Equal(t, "jane@example.com", createResp.Email)
	assert.Equal(t, "User created successfully", createResp.Message)
}

func TestHTTP_CreateUser_Conflict(t *testing.T) {
	router := setupTestRouter()
	server := httptest.NewServer(router)
	defer server.Close()
	
	// Prepare request body with duplicate email
	createReq := CreateUserRequest{
		Name:  "Bob Smith",
		Email: "duplicate@example.com",
	}
	
	jsonBody, err := json.Marshal(createReq)
	require.NoError(t, err)
	
	// Test conflict case
	resp, err := http.Post(
		server.URL+"/users",
		"application/json",
		bytes.NewReader(jsonBody),
	)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	
	var errorResp typedhttp.ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errorResp)
	require.NoError(t, err)
	
	assert.Equal(t, "CONFLICT", errorResp.Code)
	assert.Equal(t, "User with this email already exists", errorResp.Error)
}

func TestHTTP_CreateUser_ValidationError(t *testing.T) {
	router := setupTestRouter()
	server := httptest.NewServer(router)
	defer server.Close()
	
	// Test validation error with invalid JSON
	invalidJSON := `{"name":"","email":"invalid-email"}`
	
	resp, err := http.Post(
		server.URL+"/users",
		"application/json",
		strings.NewReader(invalidJSON),
	)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	
	var errorResp typedhttp.ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errorResp)
	require.NoError(t, err)
	
	assert.Equal(t, "VALIDATION_ERROR", errorResp.Code)
	assert.Contains(t, errorResp.Error, "Validation failed")
}

func TestHTTP_CreateUser_InvalidJSON(t *testing.T) {
	router := setupTestRouter()
	server := httptest.NewServer(router)
	defer server.Close()
	
	// Test invalid JSON
	resp, err := http.Post(
		server.URL+"/users",
		"application/json",
		strings.NewReader("invalid json"),
	)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	
	var errorResp typedhttp.ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errorResp)
	require.NoError(t, err)
	
	assert.Equal(t, "INVALID_JSON", errorResp.Code)
	assert.Contains(t, errorResp.Error, "Invalid JSON")
}

func TestRouterRegistration(t *testing.T) {
	router := setupTestRouter()
	
	handlers := router.GetHandlers()
	require.Len(t, handlers, 2)
	
	// Check GET handler
	getHandler := handlers[0]
	assert.Equal(t, "GET", getHandler.Method)
	assert.Equal(t, "/users/{id}", getHandler.Path)
	assert.Equal(t, "Get user by ID", getHandler.Metadata.Summary)
	assert.Contains(t, getHandler.Metadata.Tags, "users")
	
	// Check POST handler  
	postHandler := handlers[1]
	assert.Equal(t, "POST", postHandler.Method)
	assert.Equal(t, "/users", postHandler.Path)
	assert.Equal(t, "Create a new user", postHandler.Metadata.Summary)
	assert.Contains(t, postHandler.Metadata.Tags, "users")
}