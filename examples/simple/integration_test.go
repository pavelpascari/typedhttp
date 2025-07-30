package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_RealServer(t *testing.T) {
	// Start a real HTTP server
	router := setupTestRouter()
	server := httptest.NewServer(router)
	defer server.Close()

	// Test GET /users/123
	t.Run("GET /users/123", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/users/123")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var userResp GetUserResponse
		err = json.Unmarshal(body, &userResp)
		require.NoError(t, err)

		assert.Equal(t, "123", userResp.ID)
		assert.Equal(t, "John Doe", userResp.Name)
		assert.Equal(t, "john@example.com", userResp.Email)
	})

	// Test POST /users
	t.Run("POST /users", func(t *testing.T) {
		createReq := CreateUserRequest{
			Name:  "Integration Test User",
			Email: "integration@test.com",
		}

		jsonBody, err := json.Marshal(createReq)
		require.NoError(t, err)

		resp, err := http.Post(
			server.URL+"/users",
			"application/json",
			bytes.NewReader(jsonBody),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var createResp CreateUserResponse
		err = json.Unmarshal(body, &createResp)
		require.NoError(t, err)

		assert.Equal(t, "user_12345", createResp.ID)
		assert.Equal(t, "Integration Test User", createResp.Name)
		assert.Equal(t, "integration@test.com", createResp.Email)
		assert.Equal(t, "User created successfully", createResp.Message)
	})

	// Test error handling
	t.Run("GET /users/not-found", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/users/not-found")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		t.Logf("Error response: %s", string(body))

		// Just verify it's a valid JSON error response
		var errorMap map[string]interface{}
		err = json.Unmarshal(body, &errorMap)
		require.NoError(t, err)
		assert.Contains(t, errorMap, "error")
		assert.Contains(t, errorMap, "code")
	})
}

func TestIntegration_ContentTypes(t *testing.T) {
	router := setupTestRouter()
	server := httptest.NewServer(router)
	defer server.Close()

	// Test that responses have correct content type
	resp, err := http.Get(server.URL + "/users/123")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
}

func TestIntegration_ConcurrentRequests(t *testing.T) {
	router := setupTestRouter()
	server := httptest.NewServer(router)
	defer server.Close()

	// Test concurrent requests to ensure thread safety
	const numRequests = 10
	resultChan := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			resp, err := http.Get(server.URL + "/users/123")
			if err != nil {
				resultChan <- err
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				resultChan <- assert.AnError
				return
			}

			resultChan <- nil
		}(i)
	}

	// Wait for all requests to complete
	timeout := time.After(5 * time.Second)
	for i := 0; i < numRequests; i++ {
		select {
		case err := <-resultChan:
			assert.NoError(t, err, "Request %d failed", i)
		case <-timeout:
			t.Fatal("Timeout waiting for concurrent requests")
		}
	}
}
